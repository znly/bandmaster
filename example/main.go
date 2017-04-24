package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/pkg/errors"
	"github.com/znly/bandmaster"
	"github.com/znly/bandmaster/memcached"
	"github.com/znly/bandmaster/redis"
	"github.com/znly/bandmaster/waiter"
)

func main() {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)

	m := bandmaster.GlobalMaestro

	m.AddService("wn-1", false, waiter.New(waiter.DefaultConfig()), "mc-1", "mc-3")
	m.AddService("mc-1", true, memcached.New(time.Second, "localhost:11211"))
	m.AddService("mc-2", false, memcached.New(memcached.DefaultConfig()), "mc-1")
	m.AddService("mc-3", true, memcached.New(memcached.DefaultConfig()), "mc-2")
	m.AddService("rd-1", true, redis.New(redis.DefaultConfig("redis://localhost:6379/0")), "mc-2")

	ctx, canceller := context.WithTimeout(context.Background(), time.Second*5)
	for err := range m.StartAll(ctx) {
		err = errors.Cause(err)
		switch e := err.(type) {
		case *bandmaster.Error:
			if e.Service().Required() {
				zap.L().Error(e.Error())
			} else {
				zap.L().Info(e.Error())
			}
		}
	}
	canceller()
}
