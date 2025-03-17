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
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	concurrency "go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type Service struct {
	etcd    *clientv3.Client
	session *concurrency.Session
	options *options

	mutexes map[string]*muRecord
	lock    sync.Mutex
	stopper chan struct{}
	wg      sync.WaitGroup
}

type ConfigurationType int

const (
	ConfigurationTypeService ConfigurationType = iota
	ConfigurationTypeScope
	ConfigurationTypeHost
)

var ErrServiceNameNotSpecified = errors.New("service name is not specified")
var ErrWrongEtcdAddress = errors.New("wrong etcd address")
var ErrMutexAlreadyAcquired = errors.New("mutex already acquired")
var ErrEtcdTimeout = errors.New("etcd timeout")
var ErrInvalidConfigPointer = errors.New("invalid config pointer")
var ErrEmptyValue = errors.New("empty value")
var ErrNoAvailableIDs = errors.New("no available IDs")
var ErrSessionNotAvailable = errors.New("session not available")

type muRecord struct {
	mu    *concurrency.Mutex
	donec chan struct{}
}

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
		mutexes: make(map[string]*muRecord),
		stopper: make(chan struct{}),
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

	err = cli.createSession()
	if err != nil {
		cli.etcd.Close()
		return nil, err
	}

	cli.wg.Add(1)
	go cli.monitorSession()

	return cli, nil
}

func (c *Service) Close() {
	close(c.stopper)
	c.wg.Wait()

	if c.session != nil {
		c.session.Close()
	}

	c.etcd.Close()
}

func (c *Service) createSession() error {
	session, err := concurrency.NewSession(c.etcd, concurrency.WithTTL(c.options.etcdLeaseTTL))
	if err != nil {
		return err
	}

	c.lock.Lock()
	c.session = session
	c.lock.Unlock()

	return nil
}

func (c *Service) monitorSession() {
	defer c.wg.Done()

	ch := c.session.Done()

	for {
		select {
		case <-c.stopper:
			return
		case <-ch:
			c.lock.Lock()
			oldMutexes := c.mutexes
			c.mutexes = make(map[string]*muRecord)
			if c.session != nil {
				go c.session.Close()
				c.session = nil
			}
			c.lock.Unlock()

			for _, mrec := range oldMutexes {
				// in case if session is lost we kill all mutexes and notify all waiters
				close(mrec.donec)
			}

			for {
				err := c.createSession()
				if err == nil {
					break
				}

				select {
				case <-c.stopper:
					return
				case <-time.After(c.options.retryInterval):
				}
			}

			ch = c.session.Done()
		}
	}
}

func (c *Service) AcquireLock(ctx context.Context, name string) (<-chan struct{}, error) {
	key := fmt.Sprintf("%s%s%s%s", c.options.locksPrefix, c.options.serviceName, c.options.mutexesPrefix, name)

	c.lock.Lock()
	if c.session == nil {
		c.lock.Unlock()
		return nil, ErrSessionNotAvailable
	}

	_, ok := c.mutexes[key]
	if ok {
		c.lock.Unlock()
		return nil, ErrMutexAlreadyAcquired
	}
	c.lock.Unlock()

	mutex := concurrency.NewMutex(c.session, key)
	err := mutex.TryLock(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, ErrEtcdTimeout
		}

		if err == concurrency.ErrLocked {
			return nil, ErrMutexAlreadyAcquired
		}

		return nil, err
	}

	mrec := &muRecord{
		mu:    mutex,
		donec: make(chan struct{}),
	}

	c.lock.Lock()
	c.mutexes[key] = mrec
	c.lock.Unlock()

	return mrec.donec, nil
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

	err := mutex.mu.Unlock(ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrEtcdTimeout
		}

		return err
	}

	c.lock.Lock()
	mutex, ok = c.mutexes[key]
	if ok {
		close(mutex.donec)
		delete(c.mutexes, key)
	}
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

func (c *Service) LoadConfig(ctx context.Context, ct ConfigurationType, cfg any) error {
	var path string

	switch ct {
	case ConfigurationTypeService:
		path = c.options.configPrefix + c.options.serviceName + "/"
	case ConfigurationTypeScope:
		if c.options.serviceScope != "" {
			path = c.options.configPrefix + c.options.serviceScope + "/"
		} else {
			path = c.options.configPrefix + c.options.serviceName + "/"
		}
	case ConfigurationTypeHost:
		path = c.options.hostsPrefix + c.options.serviceName + "/" + Hostname() + "/"
	}

	return c.loadConfig(ctx, cfg, path)
}

func (c *Service) ID(id string) ID {
	var err error
	var idval int

	idval, err = strconv.Atoi(id)
	if err != nil || idval < 0 {
		idval = 0
	}

	return NewID(idval, c.options.serviceName)
}
