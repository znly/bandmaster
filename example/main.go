package main

import (
	"time"

	"github.com/znly/bandmaster"
	"github.com/znly/bandmaster/memcached"
	"github.com/znly/bandmaster/winner"
)

func main() {
	m := bandmaster.NewMaestro()
	m.AddService("mc-1", memcached.New(time.Second, "localhost:11211"))
	m.AddService("mc-2", memcached.New(time.Second, "localhost:11211"))
	m.AddService("mc-3", memcached.New(time.Second, "localhost:11211"))
	m.AddService("wn-1", winner.New(time.Second*10),
		bandmaster.NewServiceDependency("mc-1", true),
		bandmaster.NewServiceDependency("mc-3", false),
	)
}
