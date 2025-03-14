package svcutil

import (
	"strings"
	"time"
)

type EventType int

const (
	EventTypeUnknown EventType = iota
	EventTypeKeepAliveRenewed
	EventTypeKeepAliveStopped
	EventTypeUnableToCheckTTL
	EventTypeLeaseExpired
	EventTypeKeepAliveRestarted
	EventTypeUnableToRestartKeepAlive
	EventTypeUnableToReacquireLease
	EventTypeLeaseReacquired
	EventTypeLeaseIsTaken
)

type Events interface {
	OnEvent(EventType, string, error)
}

type options struct {
	serviceName     string
	etcdDialTimeout time.Duration
	etcdLeaseTTL    int
	locksPrefix     string
	configPrefix    string
	hostsPrefix     string
	mutexesPrefix   string
	idsPrefix       string
	endpoints       []string
	username        string
	password        string
	retryInterval   time.Duration
	events          Events
}

type noOpEvents struct{}

func (e *noOpEvents) OnEvent(_ EventType, _ string, _ error) {
	// No-op
}

func NewOptions() *options {
	return &options{
		etcdDialTimeout: 5 * time.Second,
		etcdLeaseTTL:    30,
		locksPrefix:     "/locks/",
		configPrefix:    "/configs/",
		hostsPrefix:     "/hosts/",
		mutexesPrefix:   "/mutexes/",
		idsPrefix:       "/ids/",
		retryInterval:   30 * time.Second,
		events:          &noOpEvents{},
	}
}

func Service(s string) func(*options) *options {
	return func(l *options) *options {
		l.serviceName = s
		return l
	}
}

func DialTimeout(t time.Duration) func(*options) *options {
	return func(l *options) *options {
		l.etcdDialTimeout = t
		return l
	}
}

func LeaseTTL(t int) func(*options) *options {
	return func(l *options) *options {
		l.etcdLeaseTTL = t
		return l
	}
}
func LocksPrefix(p string) func(*options) *options {
	return func(l *options) *options {
		l.locksPrefix = p
		return l
	}
}

func ConfigPrefix(p string) func(*options) *options {
	return func(l *options) *options {
		l.configPrefix = p
		return l
	}
}

func HostsPrefix(p string) func(*options) *options {
	return func(l *options) *options {
		l.hostsPrefix = p
		return l
	}
}

func MutexesPrefix(p string) func(*options) *options {
	return func(l *options) *options {
		l.mutexesPrefix = p
		return l
	}
}

func IDsPrefix(p string) func(*options) *options {
	return func(l *options) *options {
		l.idsPrefix = p
		return l
	}
}

func Endpoints(e string) func(*options) *options {
	return func(l *options) *options {
		l.endpoints = strings.Split(e, ",")
		return l
	}
}

func Username(u string) func(*options) *options {
	return func(l *options) *options {
		l.username = u
		return l
	}
}

func Password(p string) func(*options) *options {
	return func(l *options) *options {
		l.password = p
		return l
	}
}

func RetryInterval(t time.Duration) func(*options) *options {
	return func(l *options) *options {
		l.retryInterval = t
		return l
	}
}

func OnEvents(e Events) func(*options) *options {
	return func(l *options) *options {
		l.events = e
		return l
	}
}
