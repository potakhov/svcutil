package svcutil

import (
	"strings"
	"time"
)

type options struct {
	serviceName     string
	serviceScope    string
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
}

func NewOptions() *options {
	return &options{
		etcdDialTimeout: 5 * time.Second,
		etcdLeaseTTL:    30,
		locksPrefix:     "/lock/",
		configPrefix:    "/config/",
		hostsPrefix:     "/host/",
		mutexesPrefix:   "/mutex/",
		idsPrefix:       "/id/",
		retryInterval:   15 * time.Second,
	}
}

func Name(s string) func(*options) *options {
	return func(l *options) *options {
		l.serviceName = s
		return l
	}
}

func Scope(s string) func(*options) *options {
	return func(l *options) *options {
		l.serviceScope = s
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

func EtcdEndpoints(e string) func(*options) *options {
	return func(l *options) *options {
		l.endpoints = strings.Split(e, ",")
		return l
	}
}

func EtcdUsername(u string) func(*options) *options {
	return func(l *options) *options {
		l.username = u
		return l
	}
}

func EtcdPassword(p string) func(*options) *options {
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
