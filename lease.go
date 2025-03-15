package svcutil

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
)

type Lease struct {
	client     *EtcdClient
	r          *Range
	appContext context.Context

	wg      sync.WaitGroup
	stopper chan struct{}
	breaker chan bool

	closer   func()
	lease    clientv3.LeaseID
	leaseKey string

	value string
}

type reacquireResult int

const (
	reacquireSuccess reacquireResult = iota
	reacquireFailure
	reacquireLeaseTaken
)

func NewLease(r *Range, etcd *EtcdClient, appContext context.Context) *Lease {
	return &Lease{
		client:     etcd,
		r:          r,
		appContext: appContext,
		stopper:    make(chan struct{}),
		breaker:    make(chan bool, 1),
	}
}

func (i *Lease) Close() {
	close(i.stopper)
	i.wg.Wait()
}

func (i *Lease) keyPrefix() string {
	if i.r.Type == RangeTypeID {
		return fmt.Sprintf("%s%s%s", i.client.options.locksPrefix, i.client.options.serviceName, i.client.options.idsPrefix)
	} else {
		return fmt.Sprintf("%s%s%s%s/", i.client.options.locksPrefix, i.client.options.serviceName, i.client.options.hostsPrefix, Hostname())
	}
}

func (i *Lease) keepAliveWorker(kl <-chan *clientv3.LeaseKeepAliveResponse) {
	for range kl {
	}

	select {
	case i.breaker <- true:
	default:
	}
}

func (i *Lease) worker() {
	defer i.wg.Done()

	leaseAlive := true
	keepAlive := true
	tk := time.NewTicker(time.Duration(i.client.options.etcdLeaseTTL) * time.Second / 2)
workerloop:
	for {
		select {
		case <-i.stopper:
			break workerloop
		case <-i.breaker:
			if !keepAlive {
				continue
			}

			keepAlive = false
			if i.closer != nil {
				i.closer()
				i.closer = nil
			}
		case <-tk.C:
			if keepAlive {
				// everything is functioning
				continue
			}

			if leaseAlive {
				// check if the lease is still alive
				ctx, cancel := context.WithTimeout(i.appContext, i.client.options.etcdDialTimeout)
				resp, err := i.client.etcd.TimeToLive(ctx, i.lease)
				cancel()
				if err != nil {
					continue
				}

				if resp.TTL <= 0 {
					// lease is expired
					i.client.options.events.OnEvent(EventTypeLeaseExpired, i.value)
					leaseAlive = false
				} else {
					// lease is still alive, re-establish keep-alive
					keepAliveContext, keepAliveCancel := context.WithCancel(context.Background())
					kl, err := i.client.etcd.KeepAlive(keepAliveContext, i.lease)
					if err != nil {
						keepAliveCancel()
						continue
					}

					i.closer = keepAliveCancel
					keepAlive = true
					go i.keepAliveWorker(kl)
					continue
				}
			}

			if !leaseAlive {
				switch i.reacquire() {
				case reacquireSuccess:
					i.client.options.events.OnEvent(EventTypeLeaseReacquired, i.value)
					leaseAlive = true
					keepAlive = true
				case reacquireFailure:
					continue
				case reacquireLeaseTaken:
					i.client.options.events.OnEvent(EventTypeLeaseIsTakenOver, i.value)
					break workerloop
				}
			}
		}
	}

	if i.closer != nil {
		i.closer()
		i.closer = nil
	}

	if leaseAlive {
		ctx, cancel := context.WithTimeout(i.appContext, i.client.options.etcdDialTimeout)
		defer cancel()
		i.client.etcd.Revoke(ctx, i.lease)
	}
}

func (i *Lease) Obtain(ctx context.Context) (string, error) {
	lease := clientv3.NewLease(i.client.etcd)
	resp, err := lease.Grant(ctx, int64(i.client.options.etcdLeaseTTL))
	if err != nil {
		return "", err
	}

	key := i.keyPrefix()

	ids := make([]string, len(i.r.Values))
	copy(ids, i.r.Values)
	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	for _, id := range ids {
		idLockKey := key + id

		txn := i.client.etcd.Txn(ctx).
			If(clientv3.Compare(clientv3.CreateRevision(idLockKey), "=", 0)).
			Then(clientv3.OpPut(idLockKey, "locked", clientv3.WithLease(resp.ID))).
			Else()

		txnResp, err := txn.Commit()
		if err != nil {
			return "", err
		}

		if txnResp.Succeeded {
			keepAliveContext, cancel := context.WithCancel(context.Background())
			kl, err := i.client.etcd.KeepAlive(keepAliveContext, resp.ID)
			if err != nil {
				cancel()
				return "", err
			}

			go i.keepAliveWorker(kl)

			i.value = id
			i.closer = cancel
			i.lease = resp.ID
			i.leaseKey = idLockKey

			i.wg.Add(1)
			go i.worker()

			return id, nil
		}
	}

	return "", ErrNoAvailableIDs
}

func (i *Lease) Wait(ctx context.Context) (string, error) {
	for {
		id, err := i.Obtain(ctx)
		if err == nil {
			return id, nil
		}

		if err != ErrNoAvailableIDs {
			return "", err
		}

		wctx, cancel := context.WithCancel(ctx)
		watchChan := i.client.etcd.Watch(wctx, i.keyPrefix(), clientv3.WithPrefix())

		select {
		case <-watchChan:
		case <-time.After(i.client.options.retryInterval):
		case <-ctx.Done():
			cancel()
			return "", ctx.Err()
		}

		cancel()
	}
}

func (i *Lease) reacquire() reacquireResult {
	ctx, cancel := context.WithTimeout(i.appContext, i.client.options.etcdDialTimeout)
	defer cancel()

	lease := clientv3.NewLease(i.client.etcd)
	resp, err := lease.Grant(ctx, int64(i.client.options.etcdLeaseTTL))
	if err != nil {
		return reacquireFailure
	}

	txn := i.client.etcd.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(i.leaseKey), "=", 0)).
		Then(clientv3.OpPut(i.leaseKey, "locked", clientv3.WithLease(resp.ID))).
		Else()

	txnResp, err := txn.Commit()
	if err != nil {
		return reacquireFailure
	}

	if txnResp.Succeeded {
		keepAliveContext, keepAliveCancel := context.WithCancel(context.Background())
		kl, err := i.client.etcd.KeepAlive(keepAliveContext, resp.ID)
		if err != nil {
			keepAliveCancel()
			return reacquireFailure
		}

		go i.keepAliveWorker(kl)

		i.closer = keepAliveCancel
		i.lease = resp.ID

		return reacquireSuccess
	}

	return reacquireLeaseTaken
}
