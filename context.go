package svcutil

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type ProcessContextScope string

type ProcessContext struct {
	wg       *sync.WaitGroup
	ctx      context.Context
	shutdown context.CancelFunc
}

func NewProcessContext() *ProcessContext {
	ctx, shutdown := context.WithCancel(context.Background())
	return &ProcessContext{
		ctx:      ctx,
		shutdown: shutdown,
		wg:       &sync.WaitGroup{},
	}
}

func (b *ProcessContext) Context() context.Context {
	return context.WithValue(b.ctx, ProcessContextScope("scope"), "process")
}

func (b *ProcessContext) ComponentStarted() {
	b.wg.Add(1)
}

func (b *ProcessContext) ComponentFinished() {
	b.wg.Done()
}

func (b *ProcessContext) Shutdown() {
	b.shutdown()
}

func (b *ProcessContext) Done() <-chan struct{} {
	return b.ctx.Done()
}

func (b *ProcessContext) WaitForComponentsToFinish() {
	b.wg.Wait()
}

func WaitForShutdown(processCtx *ProcessContext) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sig:
	case <-processCtx.Done():
	}

	signal.Reset(syscall.SIGINT, syscall.SIGTERM)

	processCtx.Shutdown()
	processCtx.WaitForComponentsToFinish()
}
