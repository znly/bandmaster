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
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	sarama_cluster "github.com/bsm/sarama-cluster"
	"github.com/znly/bandmaster"
	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------

// Service implements a Kafka service based on the 'bsm/sarama-cluster' package.
type Service struct {
	*bandmaster.ServiceBase // inheritance

	// TODO(cmc): behavior during restarts
	ctx       context.Context // lifecycle
	canceller context.CancelFunc

	conf            *sarama_cluster.Config
	addrs           []string
	consumerTopics  []string
	consumerGroupID string

	c *sarama_cluster.Consumer
	p sarama.AsyncProducer
}

// DefaultConfig returns a Kafka cluster config with the following defaults:
//
// This configuration stores committed offsets using the default retention time
// configured on the server (`offsets.retention.minutes`) but doesn't commit
// them automatically: the consumer is thus in charge of marking & committing
// processed offsets (e.g. using an `OffsetStash`).
//
// `chanBufSize` sets the producer's & consumer's ChannelBufferSize.
// `version` sets the Kafka protocol version. See
// godoc.org/github.com/Shopify/sarama for reference.
func DefaultConfig(
	chanBufSize uint, version sarama.KafkaVersion,
) *sarama_cluster.Config {
	config := sarama_cluster.NewConfig()

	/* CONSUMER */

	// If enabled, rebalance notification will be returned on the
	// Notifications channel.
	config.Group.Return.Notifications = true
	// If enabled, any errors that occurred while consuming are returned on
	// the Errors channel.
	config.Consumer.Return.Errors = true
	// The initial offset to use if no offset was previously committed.
	// Should be OffsetNewest or OffsetOldest.
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	// How frequently to commit updated offsets.
	config.Consumer.Offsets.CommitInterval = time.Hour * 24 * 365 // 1 year
	// The retention duration for committed offsets. If zero, disabled
	// (in which case the `offsets.retention.minutes` option on the
	// broker will be used).  Kafka only supports precision up to
	// milliseconds; nanoseconds will be truncated. Requires Kafka
	// broker version 0.9.0 or later.
	config.Consumer.Offsets.Retention = 0
	// How long to wait after a failing to read from a partition before
	// trying again.
	config.Consumer.Retry.Backoff = 1 * time.Second

	/* PRODUCER */

	// If enabled, successfully delivered messages will be returned on the
	// successes channel.
	config.Producer.Return.Successes = false
	// If enabled, messages that failed to deliver will be returned on the
	// Errors channel, including error.
	config.Producer.Return.Errors = true

	/* COMMON */

	// The number of events to buffer in internal and external channels. This
	// permits the producer and consumer to continue processing some messages
	// in the background while user code is working, greatly improving throughput.
	config.ChannelBufferSize = int(chanBufSize)

	// The version of Kafka that Sarama will assume it is running against.
	// Defaults to the oldest supported stable version. Since Kafka provides
	// backwards-compatibility, setting it to a version older than you have
	// will not break anything, although it may prevent you from using the
	// latest features. Setting it to a version greater than you are actually
	// running may lead to random breakage.
	config.Version = version

	return config
}

// New creates a new service using the provided Kafka cluster configuration.
// Use `DefaultConfig` to get a pre-configured configuration.
//
// New doesn't open any connection, doesn't do any kind of I/O, nor does it
// check the validity of the passed configuration; i.e. it cannot fail.
//
// Both `consumerTopics` & `consumerGroupID` are optional: if one of them is not
// specified, no consumer will be created during initialization.
func New(conf *sarama_cluster.Config,
	addrs []string, consumerTopics []string, consumerGroupID string,
) bandmaster.Service {
	ctx, canceller := context.WithCancel(context.Background())
	return &Service{
		ServiceBase: bandmaster.NewServiceBase(), // inheritance

		ctx:       ctx,
		canceller: canceller,

		conf:            conf,
		addrs:           addrs,
		consumerTopics:  consumerTopics,
		consumerGroupID: consumerGroupID,
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
	if err = s.conf.Validate(); err != nil {
		return err
	}
	if s.c == nil { // idempotency
		if len(s.consumerGroupID) > 0 && len(s.consumerTopics) > 0 {
			s.c, err = sarama_cluster.NewConsumer(
				s.addrs, s.consumerGroupID, s.consumerTopics, s.conf,
			)
			if err != nil {
				return err
			}
		}
	}
	if s.p == nil { // idempotency
		s.p, err = sarama.NewAsyncProducer(s.addrs, &s.conf.Config)
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
		if err := s.c.Close(); err != nil {
			return err
		}
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

// WatchConsumerEvents starts the logging routines that watch the notification
// and error events of the underlying Kafka consumer if the corresponding
// options have been enabled in its configuration.
//
// This function waits for the service to be ready: the given context defines
// the deadline for this wait period.
//
// Calling this function more than once on a given `kafka.Service` will result
// in undefined behavior (not the nice kind).
//
// NOTE: This will panic if `s` is not a `kafka.Service`.
func WatchConsumerEvents(ctx context.Context, s bandmaster.Service) error {
	ss := s.(*Service) // allowed to panic
	if err := <-ss.Started(ctx); err != nil {
		return err
	}

	notify := func(verb string, cycle uint64, partitions map[string][]int32) {
		if len(partitions) <= 0 { // no-op
			return
		}
		for topic, parts := range partitions {
			zap.L().Info(fmt.Sprintf("partitions %s", verb),
				zap.Uint64("cycle", cycle),
				zap.String("topic", topic),
				zap.Int32s("partitions", parts),
			)
		}
	}

	if ss.conf.Group.Return.Notifications {
		go func() {
			var nbCycle uint64 = 1
			for {
				select {
				case <-ss.ctx.Done():
					return
				case n, ok := <-ss.c.Notifications():
					if !ok {
						return
					}
					notify("requested", nbCycle, n.Claimed)
					notify("released", nbCycle, n.Released)
					notify("claimed", nbCycle, n.Current)
					nbCycle++
				}
			}
		}()
	}
	if ss.conf.Consumer.Return.Errors {
		go func() {
			for {
				select {
				case <-ss.ctx.Done():
					return
				case err, ok := <-ss.c.Errors():
					if !ok {
						return
					}
					switch e := err.(type) {
					case *sarama.ConsumerError:
						zap.L().Error(e.Error(),
							zap.String("topic", e.Topic),
							zap.Int32("partition", e.Partition),
						)
					default:
						zap.L().Error(e.Error())
					}
				}
			}
		}()
	}

	return nil
}

// WatchProducerEvents starts the logging routines that watch the notification
// and error events of the underlying Kafka producer, if the corresponding
// options have been enabled in its configuration.
//
// This function waits for the service to be ready, it will block if it's not.
//
// Calling this function more than once on a given `kafka.Service` will result
// in undefined behavior (not the nice kind).
//
// NOTE: This will panic if `s` is not a `kafka.Service`.
func WatchProducerEvents(ctx context.Context, s bandmaster.Service) error {
	ss := s.(*Service) // allowed to panic
	if err := <-ss.Started(ctx); err != nil {
		return err
	}

	if ss.conf.Producer.Return.Errors {
		go func() {
			for {
				select {
				case <-ss.ctx.Done():
					return
				case err, ok := <-ss.p.Errors():
					if !ok {
						return
					}
					// TODO(cmc): explain error handling idiom
					if errC, ok := err.Msg.Metadata.(chan<- error); ok {
						errC <- err.Err
					} else {
						zap.L().Error(err.Error(),
							zap.String("topic", err.Msg.Topic),
							zap.Int32("partition", err.Msg.Partition),
							zap.Int64("offset", err.Msg.Offset),
							zap.String("key", fmt.Sprintf("%v", err.Msg.Key)),
						)
					}
				}
			}
		}()
	}
	if ss.conf.Producer.Return.Successes {
		go func() {
			for {
				select {
				case <-ss.ctx.Done():
					return
				case msg, ok := <-ss.p.Successes():
					if !ok {
						return
					}
					// TODO(cmc): explain error handling idiom
					if errC, ok := msg.Metadata.(chan<- error); ok {
						close(errC)
					} else {
						zap.L().Debug("message pushed",
							zap.String("topic", msg.Topic),
							zap.Int32("partition", msg.Partition),
							zap.Int64("offset", msg.Offset),
							zap.String("key", fmt.Sprintf("%v", msg.Key)),
						)
					}
				}
			}
		}()
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

// Config returns the underlying configuration of the given service.
//
// NOTE: This will panic if `s` is not a `kafka.Service`.
func Config(s bandmaster.Service) *sarama_cluster.Config {
	return s.(*Service).conf // allowed to panic
}
