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

	"github.com/Shopify/sarama"
	sarama_cluster "github.com/bsm/sarama-cluster"
	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------

// WatchConsumerEvents starts the logging routines that watch the notification
// and error events of the given Kafka consumer, iff the corresponding options
// have been enabled in its configuration.
func WatchConsumerEvents(
	ctx context.Context,
	conf *Config,
	cons *sarama_cluster.Consumer,
) error {
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

	if conf.ClusterConf.Group.Return.Notifications {
		go func() {
			var nbCycle uint64 = 1
			for n := range cons.Notifications() {
				notify("requested", nbCycle, n.Claimed)
				notify("released", nbCycle, n.Released)
				notify("claimed", nbCycle, n.Current)
				nbCycle++
			}
		}()
	}
	if conf.ClusterConf.Consumer.Return.Errors {
		go func() {
			for err := range cons.Errors() {
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
		}()
	}

	return nil
}

// WatchAsyncProducerEvents starts the logging routines that watch the
// notification and error events of the given Kafka producer, iff the
// corresponding options have been enabled in its configuration.
func WatchAsyncProducerEvents(
	ctx context.Context,
	conf *Config,
	prod sarama.AsyncProducer,
) error {
	if conf.ClusterConf.Producer.Return.Errors {
		go func() {
			for err := range prod.Errors() {
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
		}()
	}
	if conf.ClusterConf.Producer.Return.Successes {
		go func() {
			for msg := range prod.Successes() {
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
		}()
	}

	return nil
}
