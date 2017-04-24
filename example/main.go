package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/pkg/errors"
	"github.com/znly/bandmaster"
	"github.com/znly/bandmaster/memcached"
	"github.com/znly/bandmaster/winner"
)

func main() {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)

	m := bandmaster.GlobalMaestro

	m.AddService("wn-1", false, winner.New(time.Second*10), "mc-1", "mc-3")
	m.AddService("mc-1", true, memcached.New(time.Second, "localhost:11212"))
	m.AddService("mc-2", false, memcached.New(time.Second, "localhost:11211"), "mc-1")
	m.AddService("mc-3", true, memcached.New(time.Second, "localhost:11211"), "mc-2")

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
