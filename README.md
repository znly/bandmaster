<p align="center">
  <img src="resources/pics/bandmaster.png" alt="Bandmaster"/>
</p>

# BandMaster ![Status](https://img.shields.io/badge/status-stable-green.svg?style=plastic) [![Build Status](http://img.shields.io/travis/znly/bandmaster.svg?style=plastic)](https://travis-ci.org/znly/bandmaster) [![Coverage Status](https://coveralls.io/repos/github/znly/bandmaster/badge.svg?branch=master)](https://coveralls.io/github/znly/bandmaster?branch=master) [![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=plastic)](http://godoc.org/github.com/znly/bandmaster)

*BandMaster* is a simple and easily extendable Go library for managing runtime services & dependencies such as reliance on external datastores/APIs/MQs/custom-things via **a single, consistent set of APIs**.

It provides a fully tested & thread-safe package that implements some of the most-commonly needed features when dealing with 3rd-party clients, including but not limited to:
- consistent, type-safe, environment-based configuration for everything
- configurable number of retries & support for exponential backoff to recover from temporary initialization failures
- a blocking status API (via `chan` + `select{}`) so you can wait for one or more services to be ready
- designed to ease the creation & integration of custom services
- automatic parallelization & synchronization of the boot & shutdown phases
- dependency-tree semantics to define relationships between services
- auto-detection of missing & circular dependencies
- a global, thread-safe service registry so packages and goroutines can safely share clients
- full support of `context` for clean cancellation of boot & shutdown processes (using e.g. signals)
- idempotent start & stop methods
- ...and more!

*BandMaster* comes with a standard library of services including:
- Memcached via [rainycape/memcache](https://github.com/rainycape/memcache)
- Redis via [garyburd/redigo](https://github.com/garyburd/redigo)
- CQL-based datastores (e.g. Cassandra & ScyllaDB) via [gocql/gocql](https://github.com/gocql/gocql)
- NATS via [nats-io/go-nats](https://github.com/nats-io/go-nats)
- Kafka via [bsm/sarama-cluster](https://github.com/bsm/sarama-cluster)
- ElasticSearch-v1 via [gopkg.in/olivere/elastic.v2](https://gopkg.in/olivere/elastic.v2)
- ElasticSearch-v2 via [gopkg.in/olivere/elastic.v3](https://gopkg.in/olivere/elastic.v3)
- ElasticSearch-v5 via [gopkg.in/olivere/elastic.v5](https://gopkg.in/olivere/elastic.v5)

Any of these services would be configured and instanciated the exact same way:
```Go
	memcachedEnv, _ := bm_memcached.NewEnv("MC_EXAMPLE")
	redisEnv, _ := bm_redis.NewEnv("RD_EXAMPLE")
	cqlEnv, _ := bm_cql.NewEnv("CQL_EXAMPLE")
	natsEnv, _ := bm_nats.NewEnv("NATS_EXAMPLE")
	kafkaEnv, _ := bm_kafka.NewEnv("KAFKA_EXAMPLE")
	es1Env, _ := bm_es1.NewEnv("ES1_EXAMPLE")
	es2Env, _ := bm_es2.NewEnv("ES2_EXAMPLE")
	es5Env, _ := bm_es5.NewEnv("ES5_EXAMPLE")

	m.AddService("mc-1", true, bm_memcached.New(memcachedEnv.Config()))
	m.AddService("rd-1", true, bm_redis.New(redisEnv.Config()))
	m.AddService("cql-1", true, bm_cql.New(cqlEnv.Config()))
	m.AddService("nts-1", true, bm_nats.New(natsEnv.Config()))
	m.AddService("kfk-1", true, bm_kafka.New(kafkaEnv.Config()))
	m.AddService("es1-1", true, bm_es1.New(es1Env.Config()))
	m.AddService("es2-1", true, bm_es2.New(es2Env.Config()))
	m.AddService("es5-1", true, bm_es5.New(es5Env.Config()))
```

In addition to these standard implementations, *BandMaster* provides a straightforward API so that you can easily implement your own services; see [this section](#implementing-a-custom-service) for more details.

---

**Table of Contents:**
<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Usage](#usage)
  - [Quickstart](#quickstart)
  - [Implementing a custom service](#implementing-a-custom-service)
  - [Error handling](#error-handling)
  - [Logging](#logging)
- [Contributing](#contributing)
  - [Running tests](#running-tests)
- [Authors](#authors)
- [License](#license-)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Usage

### Quickstart

This example shows some basic usage of *BandMaster* that should cover 99.9% of the use-cases out there:
```Go
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
// once the channel returned by StartAll gets closed, we know for a fact
// that all of our services (minus the ones that returned an error) are
// properly shutdown
for err := range m.StopAll(ctx) {
  zap.L().Info(err.Error())
}
```

It should output the following when ran, explaining pretty straightforwardly what's actually going on:
```
{"level":"info","msg":"starting service...","service":"mc-2"}
{"level":"info","msg":"starting service...","service":"rd-1"}
{"level":"info","msg":"starting service...","service":"mc-1"}
{"level":"info","msg":"starting service...","service":"mc-x"}
{"level":"info","msg":"starting service...","service":"mc-3"}
{"level":"info","msg":"service successfully started","service":"'mc-2' [optional]"}
{"level":"info","msg":"service successfully started","service":"'mc-3' [required]"}
{"level":"info","msg":"service successfully started","service":"'rd-1' [required]"}
{"level":"info","msg":"service successfully started","service":"'mc-1' [required]"}
{"level":"info","msg":"service failed to start, retrying in 100ms...","service":"mc-x","error":"dial tcp 127.0.0.1:0: connect: can't assign requested address","attempt":1}
{"level":"info","msg":"service failed to start, retrying in 200ms...","service":"mc-x","error":"dial tcp 127.0.0.1:0: connect: can't assign requested address","attempt":2}
{"level":"warn","msg":"service failed to start","service":"mc-x","error":"dial tcp 127.0.0.1:0: connect: can't assign requested address","attempt":3}
{"level":"error","msg":"couldn't start required service","error":"`mc-x`: service failed to start: dial tcp 127.0.0.1:0: connect: can't assign requested address","service":"mc-x"}
{"level":"info","msg":"doing stuff with our new services...","memcacheds":"mc-1:0xc4200e85a0 mc-2:0xc4200e85e0 mc-3:0xc4200e8620","redis":"rd-1:0xc420011090"}
{"level":"info","msg":"doing stuff with our new services...","memcacheds":"mc-1:0xc4200e85a0 mc-2:0xc4200e85e0 mc-3:0xc4200e8620","redis":"rd-1:0xc420011090"}
{"level":"info","msg":"doing stuff with our new services...","memcacheds":"mc-1:0xc4200e85a0 mc-2:0xc4200e85e0 mc-3:0xc4200e8620","redis":"rd-1:0xc420011090"}
{"level":"info","msg":"stopping service...","service":"mc-3"}
{"level":"info","msg":"stopping service...","service":"mc-x"}
{"level":"info","msg":"stopping service...","service":"mc-2"}
{"level":"info","msg":"stopping service...","service":"mc-1"}
{"level":"info","msg":"service successfully stopped","service":"'mc-2' [optional]"}
{"level":"info","msg":"stopping service...","service":"rd-1"}
{"level":"info","msg":"service successfully stopped","service":"'mc-3' [required]"}
{"level":"info","msg":"service successfully stopped","service":"'rd-1' [required]"}
{"level":"info","msg":"service successfully stopped","service":"'mc-1' [required]"}
{"level":"info","msg":"service successfully stopped","service":"'mc-x' [required]"}
```

### Implementing a custom service

The simplest way to implement a new service is to copy & modify one of the already existing ones.
Each service is implemented in its own subpackage, all of which you can find in the [services/](./services/) directory.

The [`nats`](./services/nats) package is a good starting point for your future implementations.

Services are always composed of 3 files:
- **env.go**  
This defines an `Env` structure that handles all of the service's configuration and can easily be converted to its native configuration type (e.g. `nats.Options`).
Make sure to always provide sane defaults that will ease development in a local environment.
- **service.go**  
It defines the actual `Service` structure that plugs into *BandMaster*'s machinery by pseudo-inheriting (embedding) from `bandmaster.ServiceBase`.
This also defines the self-explanatory `Start` & `Stop` methods, as well as a `Client` (or similarly named) method that allows the caller to retrieve the actual client managed by the service (e.g. `nats.Conn`).
- **service_test.go**  
Finally, this plugs the newly implemented service into the generic *BandMaster* test suite via `services.TestService_Generic`.
These tests will make sure that the service's behavior is consistent with every other services' supported by *BandMaster*, in any circumstances.


### Error handling

*BandMaster* uses the [`pkg/errors`](https://github.com/pkg/errors) package to handle error propagation throughout the call stack; please take a look at the related documentation for more information on how to properly handle these errors.

### Logging

*BandMaster* does some logging whenever a service or one of its dependency undergoes a change of state or it anything went wrong; for that, it uses the global logger from Uber's [*Zap*](https://github.com/uber-go/zap) package.
You can thus control the behavior of *BandMaster*'s logger however you like by calling [`zap.ReplaceGlobals`](https://godoc.org/go.uber.org/zap#ReplaceGlobals) at your convenience.

For more information, see *Zap*'s [documentation](https://godoc.org/go.uber.org/zap).

## Contributing

Contributions of any kind are welcome; especially additions to the library of Service implementations, and improvements to the env-based configuration of existing services.

*BandMaster* is pretty-much frozen in terms of features; if you still find it to be lacking something, please file an issue to discuss it first.
Also, do not hesitate to open an issue if some piece of documentation looks either unclear or incomplete to you, nay is just missing entirely.

*Code contributions must be thoroughly tested and documented.*

### Running tests

```sh
$ docker-compose -f test/docker-compose.yml up
$ make test
```

## Authors

See [AUTHORS](./AUTHORS) for the list of contributors.

## License ![License](https://img.shields.io/badge/license-Apache2-blue.svg?style=plastic)

The Apache License version 2.0 (Apache2) - see [LICENSE](./LICENSE) for more details.

Copyright (c) 2017	Zenly	<hello@zen.ly> [@zenlyapp](https://twitter.com/zenlyapp)
