package svcutil

import (
	"encoding/json"
	"errors"
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
}

var ErrWrongEtcdAddress = errors.New("wrong etcd address")
var ErrMutexAlreadyAcquired = errors.New("mutex already acquired")
var ErrEtcdTimeout = errors.New("etcd timeout")
var ErrInvalidConfigPointer = errors.New("invalid config pointer")

func NewEtcdClient(opt ...func(*options) *options) (*EtcdClient, error) {
	o := NewOptions()

	for _, decorator := range opt {
		o = decorator(o)
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
	c.session.Close()
	c.etcd.Close()
}

func (c *EtcdClient) AcquireLock(name string) error {
	key := c.options.locksPrefix + name

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
	key := c.options.locksPrefix + name

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

func getJSONTags(v any) map[string]string {
	tags := make(map[string]string)
	val := reflect.TypeOf(v)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			tags[field.Name] = jsonTag
		}
	}

	return tags
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
		resp, err := c.etcd.Get(ctx, c.options.configPrefix+jsonTag)
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
