package testutil

import (
	"context"

	"github.com/TeamTenuki/twiddler/stream"
	"github.com/TeamTenuki/twiddler/watcher"
)

var _ watcher.Watcher = &Watcher{}

type Watcher struct {
	c chan []stream.Stream
}

func NewWatcher() *Watcher {
	return &Watcher{
		c: make(chan []stream.Stream),
	}
}

func (p *Watcher) Watch(c context.Context) error {
	return nil
}

func (p *Watcher) Source() <-chan []stream.Stream {
	return p.c
}

func (p *Watcher) Send(ss []stream.Stream) {
	p.c <- ss
}

func (p *Watcher) Close() error {
	close(p.c)

	return nil
}
