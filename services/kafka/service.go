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

package kafka

import (
	"context"
	"sync"

	"github.com/Shopify/sarama"
	sarama_cluster "github.com/bsm/sarama-cluster"
	"github.com/znly/bandmaster"
)

// -----------------------------------------------------------------------------

// Service implements a Kafka service based on the 'bsm/sarama-cluster' package.
type Service struct {
	*bandmaster.ServiceBase // "inheritance"

	// TODO(cmc): behavior during restarts
	ctx       context.Context // lifecycle
	canceller context.CancelFunc

	conf *Config

	c *sarama_cluster.Consumer
	p sarama.AsyncProducer
}

// Config contains the necessary configuration for a Kafka service.
type Config struct {
	ClusterConf *sarama_cluster.Config

	Addrs           []string
	ConsumerTopics  []string
	ConsumerGroupID string
}

// New creates a new Kafka service using the provided configuration.
// You may use the helpers for environment-based configuration to get a
// pre-configured `Config` with sane defaults.
//
// New doesn't open any connection, doesn't do any kind of I/O, nor does it
// check the validity of the passed configuration; i.e. it cannot fail.
func New(conf *Config) bandmaster.Service {
	ctx, canceller := context.WithCancel(context.Background())
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(), // "inheritance"

		ctx:       ctx,
		canceller: canceller,

		conf: conf,
	}
}

// -----------------------------------------------------------------------------

// Start checks the validity of the configuration then creates a new Kafka
// consumer as well as an asynchronous producer: if everything goes smoothly,
// the service is marked as 'started'; otherwise, an error is returned.
//
//
// Start is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Start(context.Context, map[string]bandmaster.Service) error {
	var err error
	if err = s.conf.ClusterConf.Validate(); err != nil {
		return err
	}
	if s.c == nil { // idempotency
		if len(s.conf.ConsumerGroupID) > 0 && len(s.conf.ConsumerTopics) > 0 {
			s.c, err = sarama_cluster.NewConsumer(s.conf.Addrs,
				s.conf.ConsumerGroupID, s.conf.ConsumerTopics, s.conf.ClusterConf,
			)
			if err != nil {
				return err
			}
		}
	}
	if s.p == nil { // idempotency
		s.p, err = sarama.NewAsyncProducer(
			s.conf.Addrs, &s.conf.ClusterConf.Config)
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop closes the underlying Kafka producer & consumer: if everything goes
// smoothly, the service is marked as 'stopped'; otherwise, an error is
// returned.
//
//
// Stop is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Stop(context.Context) error {
	s.canceller()
	if s.c != nil {
		// NOTE(cmc): this fixes an issue with `Consumer.Close()` not properly
		// draining all of its channels, which ends up blocking the entire
		// shutdown process when using ConsumerModePartitions.
		// Refer to https://github.com/bsm/sarama-cluster/issues/193 for more
		// information.
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for range s.c.Partitions() {
			}
			wg.Done()
		}()
		if err := s.c.Close(); err != nil {
			return err
		}
		wg.Wait()
		s.c = nil // idempotency & restart support
	}
	if s.p != nil {
		if err := s.p.Close(); err != nil {
			return err
		}
		s.c = nil // idempotency & restart support
	}
	return nil
}

// -----------------------------------------------------------------------------

// Consumer returns the underlying Kafka consumer of the given service.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `kafka.Service`.
func Consumer(s bandmaster.Service) *sarama_cluster.Consumer {
	return s.(*Service).c // allowed to panic
}

// Producer returns the underlying Kafka async-producer of the given service.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `kafka.Service`.
func Producer(s bandmaster.Service) sarama.AsyncProducer {
	return s.(*Service).p // allowed to panic
}

// Conf returns the underlying configuration of the given service.
//
// NOTE: This will panic if `s` is not a `kafka.Service`.
func Conf(s bandmaster.Service) *Config {
	return s.(*Service).conf // allowed to panic
}
