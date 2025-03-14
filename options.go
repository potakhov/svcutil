package svcutil

import (
	"strings"
	"time"
)

type options struct {
	etcdTimeout  time.Duration
	etcdLeaseTTL int
	locksPrefix  string
	configPrefix string
	endpoints    []string
	username     string
	password     string
}

func NewOptions() *options {
	return &options{
		etcdTimeout:  5 * time.Second,
		etcdLeaseTTL: 60,
		locksPrefix:  "/locks/",
		configPrefix: "/configs/",
	}
}

func Timeout(t time.Duration) func(*options) *options {
	return func(l *options) *options {
		l.etcdTimeout = t
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
