package svcutil

import (
	"encoding/json"
	"errors"
	"fmt"
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

type Service struct {
	etcd    *clientv3.Client
	session *concurrency.Session
	options *options

	mutexes map[string]*concurrency.Mutex
	lock    sync.Mutex
}

var ErrServiceNameNotSpecified = errors.New("service name is not specified")
var ErrWrongEtcdAddress = errors.New("wrong etcd address")
var ErrMutexAlreadyAcquired = errors.New("mutex already acquired")
var ErrEtcdTimeout = errors.New("etcd timeout")
var ErrInvalidConfigPointer = errors.New("invalid config pointer")
var ErrEmptyValue = errors.New("empty value")
var ErrNoAvailableIDs = errors.New("no available IDs")

func NewService(opt ...func(*options) *options) (*Service, error) {
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

	cli := &Service{
		options: o,
		mutexes: make(map[string]*concurrency.Mutex),
	}

	var err error
	cli.etcd, err = clientv3.New(clientv3.Config{
		Endpoints:   o.endpoints,
		DialTimeout: o.etcdDialTimeout,
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

func (c *Service) Close() {
	c.session.Close()
	c.etcd.Close()
}

func (c *Service) AcquireLock(ctx context.Context, name string) error {
	key := fmt.Sprintf("%s%s%s%s", c.options.locksPrefix, c.options.serviceName, c.options.mutexesPrefix, name)

	c.lock.Lock()
	_, ok := c.mutexes[key]
	if ok {
		c.lock.Unlock()
		return ErrMutexAlreadyAcquired
	}
	c.lock.Unlock()

	mutex := concurrency.NewMutex(c.session, key)
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

func (c *Service) ReleaseLock(ctx context.Context, name string) error {
	key := fmt.Sprintf("%s%s%s%s", c.options.locksPrefix, c.options.serviceName, c.options.mutexesPrefix, name)

	c.lock.Lock()
	mutex, ok := c.mutexes[key]
	if !ok {
		c.lock.Unlock()
		return nil
	}
	c.lock.Unlock()

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

func (c *Service) loadConfig(ctx context.Context, cfg any, path string) error {
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

	for fieldName, jsonTag := range tags {
		key := path + jsonTag
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

func (c *Service) LoadConfig(ctx context.Context, cfg any) error {
	path := c.options.configPrefix + c.options.serviceName + "/"
	return c.loadConfig(ctx, cfg, path)
}

func (c *Service) LoadScopeConfig(ctx context.Context, cfg any) error {
	var path string
	if c.options.serviceScope != "" {
		path = c.options.configPrefix + c.options.serviceScope + "/"
	} else {
		path = c.options.configPrefix + c.options.serviceName + "/"
	}
	return c.loadConfig(ctx, cfg, path)
}

func (c *Service) LoadHostConfig(ctx context.Context, cfg any) error {
	path := c.options.hostsPrefix + c.options.serviceName + "/" + Hostname() + "/"
	return c.loadConfig(ctx, cfg, path)
}

func (c *Service) ID(id string) ID {
	var sid ID
	sid.Hostname = Hostname()

	if id != "" {
		var err error
		sid.ID, err = strconv.Atoi(id)
		if err != nil || sid.ID < 0 {
			sid.ID = 0
		}
	}

	if sid.ID > 0 {
		sid.Service = fmt.Sprintf("%s-%s-%d", sid.Hostname, c.options.serviceName, sid.ID)
	} else {
		sid.Service = fmt.Sprintf("%s-%s", sid.Hostname, c.options.serviceName)
	}

	return sid
}
