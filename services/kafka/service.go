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
	"time"

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

	clients   []*sarama_cluster.Client
	lastBoot  time.Time
	curClient uint
}

// Config contains the necessary configuration for a Kafka service.
//
// TODO(cmc): explain NB_CLIENTS
type Config struct {
	ClusterConf *sarama_cluster.Config

	Addrs     []string
	NbClients uint
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

		conf:      conf,
		curClient: 0,
	}
}

// -----------------------------------------------------------------------------

// Start checks the validity of the configuration then creates a total of
// NB_CLIENTS Kafka clients: if everything goes smoothly, the service is marked
// as 'started'; otherwise, an error is returned.
//
//
// Start is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Start(context.Context, map[string]bandmaster.Service) error {
	var err error
	if err = s.conf.ClusterConf.Validate(); err != nil {
		return err
	}
	if s.clients == nil { // idempotency
		for i := 0; i < int(s.conf.NbClients); i++ {
			c, err := sarama_cluster.NewClient(s.conf.Addrs, s.conf.ClusterConf)
			if err != nil {
				return err
			}
			s.clients = append(s.clients, c)
			s.lastBoot = time.Now()
		}
	}

	return nil
}

// Stop closes the underlying Kafka clients: if everything goes smoothly, the
// service is marked as 'stopped'; otherwise, an error is returned.
//
//
// Stop is used by BandMaster's internal machinery, it shouldn't ever be called
// directly by the end-user of the service.
func (s *Service) Stop(context.Context) error {
	s.canceller()
	if s.clients != nil {
		// TODO(cmc): document this mess
		if time.Since(s.lastBoot) < time.Second {
			time.Sleep(time.Second)
		}
		for i := int(s.curClient - 1); i >= 0; i-- {
			if err := s.clients[i].Close(); err != nil {
				return err
			}
			s.curClient--
		}
		s.clients = nil // idempotency & restart support
	}
	return nil
}

// -----------------------------------------------------------------------------

// Consumer creates and returns a clustered Kafka consumer using one of the
// still available underlying clients.
// It will return nil if all of the underlying clients are already in use.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `kafka.Service`.
func Consumer(
	s bandmaster.Service,
	groupID string,
	topics ...string,
) *sarama_cluster.Consumer {
	service := s.(*Service) // allowed to panic
	if service.clients == nil {
		return nil
	}

	if service.curClient < uint(len(service.clients)) {
		client := service.clients[service.curClient]
		cons, err := sarama_cluster.NewConsumerFromClient(client,
			groupID, topics)
		if err != nil {
			panic(err)
		}
		service.curClient++
		return cons
	}

	return nil
}

// AsyncProducer creates and returns an async Kafka producer using one of the
// still available underlying clients.
// It will return nil if all of the underlying clients are already in use.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `kafka.Service`.
func AsyncProducer(s bandmaster.Service) sarama.AsyncProducer {
	service := s.(*Service) // allowed to panic
	if service.clients == nil {
		return nil
	}

	if service.curClient < uint(len(service.clients)) {
		client := service.clients[service.curClient]
		prod, err := sarama.NewAsyncProducerFromClient(client.Client)
		if err != nil {
			panic(err) // TODO(cmc)
		}
		service.curClient++
		return prod
	}

	return nil
}

// SyncProducer creates and returns an async Kafka producer using one of the
// still available underlying clients.
// It will return nil if all of the underlying clients are already in use.
//
// It assumes that the service is ready; i.e. it might return nil if it's
// actually not.
//
// NOTE: This will panic if `s` is not a `kafka.Service`.
func SyncProducer(s bandmaster.Service) sarama.SyncProducer {
	service := s.(*Service) // allowed to panic
	if service.clients == nil {
		return nil
	}

	if service.curClient < uint(len(service.clients)) {
		client := service.clients[service.curClient]
		service.curClient++
		prod, err := sarama.NewSyncProducerFromClient(client.Client)
		if err != nil {
			panic(err) // TODO(cmc)
		}
		return prod
	}

	return nil
}

// Conf returns the underlying configuration of the given service.
//
// NOTE: This will panic if `s` is not a `kafka.Service`.
func Conf(s bandmaster.Service) *Config {
	return s.(*Service).conf // allowed to panic
}
