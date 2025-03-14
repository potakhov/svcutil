package svcutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
	concurrency "go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type EtcdClient struct {
	etcd    *clientv3.Client
	session *concurrency.Session
	options *options

	mutexes map[string]*concurrency.Mutex
	lock    sync.Mutex

	idCloser func()
	idLease  clientv3.LeaseID
	id       ServiceID
}

var ErrServiceNameNotSpecified = errors.New("service name is not specified")
var ErrWrongEtcdAddress = errors.New("wrong etcd address")
var ErrMutexAlreadyAcquired = errors.New("mutex already acquired")
var ErrEtcdTimeout = errors.New("etcd timeout")
var ErrInvalidConfigPointer = errors.New("invalid config pointer")
var ErrEmptyValue = errors.New("empty value")
var ErrNoAvailableIDs = errors.New("no available IDs")

func NewEtcdClient(opt ...func(*options) *options) (*EtcdClient, error) {
	o := NewOptions()

	for _, decorator := range opt {
		o = decorator(o)
	}

	if o.serviceName == "" {
		return nil, ErrServiceNameNotSpecified
	}

	if len(o.endpoints) == 0 {
		o.endpoints = strings.Split(os.Getenv("ETCD_ADDRESS"), ",")
	}

	if o.username == "" {
		o.username = os.Getenv("ETCD_USER")
	}

	if o.password == "" {
		o.password = os.Getenv("ETCD_PASSWORD")
	}

	if len(o.endpoints) == 0 {
		return nil, ErrWrongEtcdAddress
	}

	cli := &EtcdClient{
		options: o,
		mutexes: make(map[string]*concurrency.Mutex),
	}

	var err error
	cli.etcd, err = clientv3.New(clientv3.Config{
		Endpoints:   o.endpoints,
		DialTimeout: o.etcdTimeout,
		Username:    o.username,
		Password:    o.password,
		Logger:      zap.NewNop(),
	})

	if err != nil {
		return nil, err
	}

	cli.session, err = concurrency.NewSession(cli.etcd, concurrency.WithTTL(o.etcdLeaseTTL))
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func (c *EtcdClient) Close() {
	if c.idCloser != nil {
		c.idCloser()
		ctx, cancel := context.WithTimeout(context.Background(), c.options.etcdTimeout)
		defer cancel()
		c.etcd.Revoke(ctx, c.idLease)
	}

	c.session.Close()
	c.etcd.Close()
}

func (c *EtcdClient) AcquireLock(name string) error {
	key := fmt.Sprintf("%s%s%s/%s", c.options.locksPrefix, c.options.serviceName, c.options.mutexesPrefix, name)

	c.lock.Lock()
	_, ok := c.mutexes[key]
	if ok {
		c.lock.Unlock()
		return ErrMutexAlreadyAcquired
	}
	c.lock.Unlock()

	mutex := concurrency.NewMutex(c.session, key)
	ctx, cancel := context.WithTimeout(context.Background(), c.options.etcdTimeout)
	defer cancel()

	err := mutex.TryLock(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrEtcdTimeout
		}

		if err == concurrency.ErrLocked {
			return ErrMutexAlreadyAcquired
		}

		return err
	}

	c.lock.Lock()
	c.mutexes[key] = mutex
	c.lock.Unlock()

	return nil
}

func (c *EtcdClient) ReleaseLock(name string) error {
	key := fmt.Sprintf("%s%s%s/%s", c.options.locksPrefix, c.options.serviceName, c.options.mutexesPrefix, name)

	c.lock.Lock()
	mutex, ok := c.mutexes[key]
	if !ok {
		c.lock.Unlock()
		return nil
	}
	c.lock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), c.options.etcdTimeout)
	defer cancel()
	err := mutex.Unlock(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrEtcdTimeout
		}

		return err
	}

	c.lock.Lock()
	delete(c.mutexes, key)
	c.lock.Unlock()

	return nil
}

func (c *EtcdClient) LoadConfig(cfg any) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr {
		return ErrInvalidConfigPointer
	}

	if v.Elem().Kind() != reflect.Struct {
		return ErrInvalidConfigPointer
	}

	tags := getJSONTags(cfg)
	if len(tags) == 0 {
		return ErrInvalidConfigPointer
	}

	cfgValue := v.Elem()

	ctx, cancel := context.WithTimeout(context.Background(), c.options.etcdTimeout)
	defer cancel()

	for fieldName, jsonTag := range tags {
		key := fmt.Sprintf("%s%s/%s", c.options.configPrefix, c.options.serviceName, jsonTag)
		resp, err := c.etcd.Get(ctx, key)
		if err != nil {
			return err
		}

		if len(resp.Kvs) > 0 {
			field := cfgValue.FieldByName(fieldName)
			if field.CanSet() {
				value := string(resp.Kvs[0].Value)

				switch field.Kind() {
				case reflect.String:
					field.SetString(value)
				case reflect.Int, reflect.Int64:
					var intVal int64
					if err := json.Unmarshal([]byte(value), &intVal); err == nil {
						field.SetInt(intVal)
					}
				case reflect.Bool:
					boolVal, err := strconv.ParseBool(value)
					if err == nil {
						field.SetBool(boolVal)
					}
				default:
				}
			}
		}
	}

	return nil
}

func (c *EtcdClient) GetHostValue(key string) (string, error) {
	idsKey := fmt.Sprintf("%s%s/%s/%s", c.options.hostsPrefix, c.options.serviceName, Hostname(), key)

	ctx, cancel := context.WithTimeout(context.Background(), c.options.etcdTimeout)
	defer cancel()

	respKV, err := c.etcd.Get(ctx, idsKey)
	if err != nil {
		return "", err
	}

	if len(respKV.Kvs) == 0 {
		return "", ErrEmptyValue
	}

	return string(respKV.Kvs[0].Value), nil
}

func (c *EtcdClient) LeaseServiceID(r Range) (ServiceID, error) {
	if r.Type != RangeTypeID {
		return ServiceID{}, ErrInvalidRange
	}

	if c.idCloser != nil {
		return c.id, nil
	}

	lease := clientv3.NewLease(c.etcd)
	ctx, cancel := context.WithTimeout(context.Background(), c.options.etcdTimeout)
	resp, err := lease.Grant(ctx, int64(c.options.etcdLeaseTTL))
	cancel()
	if err != nil {
		return ServiceID{}, err
	}

	key := fmt.Sprintf("%s%s%s/", c.options.locksPrefix, c.options.serviceName, c.options.idsPrefix)

	ids := make([]string, len(r.Values))
	copy(ids, r.Values)
	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	for _, id := range ids {
		idLockKey := key + id

		ctx, cancel = context.WithTimeout(context.Background(), c.options.etcdTimeout)
		txn := c.etcd.Txn(ctx).
			If(clientv3.Compare(clientv3.CreateRevision(idLockKey), "=", 0)).
			Then(clientv3.OpPut(idLockKey, "locked", clientv3.WithLease(resp.ID))).
			Else()

		txnResp, err := txn.Commit()
		cancel()
		if err != nil {
			return ServiceID{}, err
		}

		if txnResp.Succeeded {
			ctx, cancel := context.WithCancel(context.Background())
			keepAlive, err := c.etcd.KeepAlive(ctx, resp.ID)
			if err != nil || keepAlive == nil {
				cancel()
				return ServiceID{}, err
			}

			go func() {
				for range keepAlive {
				}
			}()

			idInt, _ := strconv.Atoi(id)
			c.id = NewServiceID(c.options.serviceName, idInt)
			c.idCloser = cancel
			c.idLease = resp.ID

			return c.id, nil
		}
	}

	return ServiceID{}, ErrNoAvailableIDs
}
