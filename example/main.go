// Copyright © 2017 Zenly <hello@zen.ly>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pkg/errors"
	"github.com/znly/bandmaster"
	bm_memcached "github.com/znly/bandmaster/services/memcached"
	bm_redis "github.com/znly/bandmaster/services/redis"
)

// -----------------------------------------------------------------------------

/* This example expects a memcached as well as a redis instance to be
 * up & running, respectively listening on 'localhost:11211' & 'localhost:6379'.
 *
 * docker-compose -f ../test/docker-compose.yml up -d redis memcached
 *
 */

func newLogger() *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg), os.Stdout, zap.InfoLevel)
	return zap.New(core)
}

func main() {
	// build logger with deterministic output for this example
	zap.ReplaceGlobals(newLogger())

	// get package-level Maestro instance
	m := bandmaster.GlobalMaestro()

	// get environment or default configuration for memcached & redis
	memcachedEnv, _ := bm_memcached.NewEnv("MC_EXAMPLE")
	redisEnv, _ := bm_redis.NewEnv("RD_EXAMPLE")

	// add a memcached service called 'mc-1' that depends on 'rd-1' which
	// does not yet exist
	m.AddService("mc-1", true, bm_memcached.New(memcachedEnv.Config()), "rd-1")
	// add a memcached service called 'mc-2' with no dependencies
	m.AddService("mc-2", false, bm_memcached.New(memcachedEnv.Config()))
	// add a memcached service called 'mc-3' that depends on 'mc-2'
	m.AddService("mc-3", true, bm_memcached.New(memcachedEnv.Config()), "mc-2")
	// add a redis service called 'rd-1' that depends on 'mc-3', and hence
	// also indirectly depends on on 'mc-2'
	m.AddService("rd-1", true, bm_redis.New(redisEnv.Config()), "mc-3")

	// add a final memcached service called 'mc-x' that just directly depends
	// on everything else, cannot possibly boot successfully, and has some
	// exponential backoff configured
	conf := memcachedEnv.Config()
	conf.Addrs = []string{"localhost:0"}
	m.AddServiceWithBackoff(
		"mc-x", true,
		3, time.Millisecond*100, bm_memcached.New(conf),
		"mc-1", "mc-2", "mc-3", "rd-1")

	/* Obviously, memcached instances depending on other memcached instances
	 * doesn't make any kind of sense, but that's just for the sake of example
	 */

	// give it 5sec max to start everything
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	// once the channel returned by StartAll gets closed, we know for a fact
	// that all of our services (minus the ones that returned an error) are
	// ready for action
	for err := range m.StartAll(ctx) {
		e, ok := errors.Cause(err).(*bandmaster.Error)
		if ok {
			// if the service is marked as required, we should start worrying
			if e.Service.Required() {
				zap.L().Error("couldn't start required service",
					zap.Error(e), zap.String("service", e.Service.Name()))
			} else {
				zap.L().Info("couldn't start optional service",
					zap.Error(e), zap.String("service", e.Service.Name()))
			}
		}
	}

	// since StartAll's channel is closed, our services must be ready by now
	mc1, mc2, mc3 := m.Service("mc-1"), m.Service("mc-2"), m.Service("mc-3")
	rd1 := m.Service("rd-1")
	for i := 0; i < 3; i++ {
		zap.L().Info("doing stuff with our new services...",
			zap.String("memcacheds", fmt.Sprintf("mc-1:%p mc-2:%p mc-3:%p", mc1, mc2, mc3)),
			zap.String("redis", fmt.Sprintf("rd-1:%p", rd1)))
		time.Sleep(time.Second)
	}

	// give it 5sec max to stop everything
	ctx, _ = context.WithTimeout(context.Background(), time.Second*5)
	// once the channel returned by StopAll gets closed, we know for a fact
	// that all of our services (minus the ones that returned an error) are
	// properly shutdown
	for err := range m.StopAll(ctx) {
		zap.L().Info(err.Error())
	}

	// Output:
	// {"level":"info","msg":"starting service...","service":"mc-x"}
	// {"level":"info","msg":"starting service...","service":"mc-2"}
	// {"level":"info","msg":"starting service...","service":"mc-3"}
	// {"level":"info","msg":"starting service...","service":"mc-1"}
	// {"level":"info","msg":"starting service...","service":"rd-1"}
	// {"level":"info","msg":"service successfully started","service":"'mc-2' [optional]"}
	// {"level":"info","msg":"service successfully started","service":"'mc-3' [required]"}
	// {"level":"info","msg":"service successfully started","service":"'rd-1' [required]"}
	// {"level":"info","msg":"service successfully started","service":"'mc-1' [required]"}
	// {"level":"info","msg":"service failed to start, retrying in 100ms...","service":"mc-x","attempts":1,"error":"dial tcp 127.0.0.1:0: connect: can't assign requested address"}
	// {"level":"info","msg":"service failed to start, retrying in 200ms...","service":"mc-x","attempts":2,"error":"dial tcp 127.0.0.1:0: connect: can't assign requested address"}
	// {"level":"warn","msg":"service failed to start","service":"mc-x","attempts":3,"error":"dial tcp 127.0.0.1:0: connect: can't assign requested address"}
	// {"level":"error","msg":"couldn't start required service","error":"`mc-x`: service failed to start: dial tcp 127.0.0.1:0: connect: can't assign requested address","service":"mc-x"}
	// {"level":"info","msg":"doing stuff with our new services...","memcacheds":"mc-1:0xc4200e85a0 mc-2:0xc4200e85e0 mc-3:0xc4200e8620","redis":"rd-1:0xc420011010"}
	// {"level":"info","msg":"doing stuff with our new services...","memcacheds":"mc-1:0xc4200e85a0 mc-2:0xc4200e85e0 mc-3:0xc4200e8620","redis":"rd-1:0xc420011010"}
	// {"level":"info","msg":"doing stuff with our new services...","memcacheds":"mc-1:0xc4200e85a0 mc-2:0xc4200e85e0 mc-3:0xc4200e8620","redis":"rd-1:0xc420011010"}
	// {"level":"info","msg":"stopping service...","service":"mc-3"}
	// {"level":"info","msg":"stopping service...","service":"rd-1"}
	// {"level":"info","msg":"service is already stopped","service":"'mc-x' [required]"}
	// {"level":"info","msg":"stopping service...","service":"mc-2"}
	// {"level":"info","msg":"stopping service...","service":"mc-1"}
	// {"level":"info","msg":"service successfully stopped","service":"'mc-1' [required]"}
	// {"level":"info","msg":"service successfully stopped","service":"'rd-1' [required]"}
	// {"level":"info","msg":"service successfully stopped","service":"'mc-3' [required]"}
	// {"level":"info","msg":"service successfully stopped","service":"'mc-2' [optional]"}
}
