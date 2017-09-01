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
	"strings"
	"time"

	"github.com/Shopify/sarama"
	sarama_cluster "github.com/bsm/sarama-cluster"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// -----------------------------------------------------------------------------

// Env can be used to configure a Kafka session via the environment.
//
// It comes with sane defaults for a local development set-up.
type Env struct {
	/* Common */
	Version Version  `envconfig:"VERSION" default:"V0_10_1_0"`
	Addrs   []string `envconfig:"ADDRS" default:"localhost:9092"`
	Bufsize int      `envconfig:"BUFSIZE" default:"4096"`

	/* Producer */
	ProdNotifSuccess bool        `envconfig:"PROD_NOTIF_SUCCESS" default:"false"`
	ProdNotifError   bool        `envconfig:"PROD_NOTIF_ERROR" default:"true"`
	ProdCompression  Compression `envconfig:"PROD_COMPRESSION" default:"none"`

	/* Consumer */
	ConsTopics               []string      `envconfig:"CONS_TOPICS" default:""`
	ConsGroupID              string        `envconfig:"CONS_GROUP_ID" default:""`
	ConsNotifRebalance       bool          `envconfig:"CONS_NOTIF_REBALANCE" default:"true"`
	ConsNotifError           bool          `envconfig:"CONS_NOTIF_ERROR" default:"true"`
	ConsOffsetInitial        int64         `envconfig:"CONS_OFFSET_INITIAL" default:"-1"` // OffsetNewest
	ConsOffsetCommitInterval time.Duration `envconfig:"CONS_OFFSET_COMMIT_INTERVAL" default:"5m"`
	ConsOffsetRetention      time.Duration `envconfig:"CONS_OFFSET_RETENTION" default:"0"`
	ConsRetryBackoff         time.Duration `envconfig:"CONS_RETRY_BACKOFF" default:"1s"`
}

// NewEnv parses the environment and returns a new `Env` structure.
//
// `prefix` defines the prefix for the environment keys, e.g. with a 'XX' prefix,
// 'REPLICAS' would become 'XX_REPLICAS'.
func NewEnv(prefix string) (*Env, error) {
	e := &Env{}
	if err := envconfig.Process(prefix, e); err != nil {
		return nil, errors.WithStack(err)
	}
	return e, nil
}

// Config returns a `Config` using the values from the environment.
func (e *Env) Config() *Config {
	clusterConf := sarama_cluster.NewConfig()

	/* CONSUMER */

	// If enabled, rebalance notification will be returned on the
	// Notifications channel.
	clusterConf.Group.Return.Notifications = e.ConsNotifRebalance
	// If enabled, any errors that occurred while consuming are returned on
	// the Errors channel.
	clusterConf.Consumer.Return.Errors = e.ConsNotifError
	// The initial offset to use if no offset was previously committed.
	// Should be OffsetNewest or OffsetOldest.
	clusterConf.Consumer.Offsets.Initial = e.ConsOffsetInitial
	// How frequently to commit updated offsets.
	clusterConf.Consumer.Offsets.CommitInterval = e.ConsOffsetCommitInterval
	// The retention duration for committed offsets. If zero, disabled
	// (in which case the `offsets.retention.minutes` option on the
	// broker will be used).  Kafka only supports precision up to
	// milliseconds; nanoseconds will be truncated. Requires Kafka
	// broker version 0.9.0 or later.
	clusterConf.Consumer.Offsets.Retention = e.ConsOffsetRetention
	// How long to wait after a failing to read from a partition before
	// trying again.
	clusterConf.Consumer.Retry.Backoff = e.ConsRetryBackoff

	/* PRODUCER */

	// If enabled, successfully delivered messages will be returned on the
	// successes channel.
	clusterConf.Producer.Return.Successes = e.ProdNotifSuccess
	// If enabled, messages that failed to deliver will be returned on the
	// Errors channel, including error.
	clusterConf.Producer.Return.Errors = e.ProdNotifError
	// The type of compression to use on messages (defaults to no compression).
	// Similar to `compression.codec` setting of the JVM producer.
	clusterConf.Producer.Compression = sarama.CompressionCodec(e.ProdCompression)

	/* COMMON */

	// The number of events to buffer in internal and external channels. This
	// permits the producer and consumer to continue processing some messages
	// in the background while user code is working, greatly improving throughput.
	clusterConf.ChannelBufferSize = e.Bufsize

	// The version of Kafka that Sarama will assume it is running against.
	// Defaults to the oldest supported stable version. Since Kafka provides
	// backwards-compatibility, setting it to a version older than you have
	// will not break anything, although it may prevent you from using the
	// latest features. Setting it to a version greater than you are actually
	// running may lead to random breakage.
	clusterConf.Version = sarama.KafkaVersion(e.Version)

	conf := &Config{
		ClusterConf:     clusterConf,
		Addrs:           e.Addrs,
		ConsumerTopics:  e.ConsTopics,
		ConsumerGroupID: e.ConsGroupID,
	}

	return conf
}

// -----------------------------------------------------------------------------

// Version is used to configure sarama's kafka version value from the
// environment.
type Version sarama.KafkaVersion

func (v *Version) Decode(vv string) error {
	switch strings.ToUpper(vv) {
	case "V0_8_2_0":
		*v = Version(sarama.V0_8_2_0)
	case "V0_8_2_1":
		*v = Version(sarama.V0_8_2_1)
	case "V0_8_2_2":
		*v = Version(sarama.V0_8_2_2)
	case "V0_9_0_0":
		*v = Version(sarama.V0_9_0_0)
	case "V0_9_0_1":
		*v = Version(sarama.V0_9_0_1)
	case "V0_10_0_0":
		*v = Version(sarama.V0_10_0_0)
	case "V0_10_0_1":
		*v = Version(sarama.V0_10_0_1)
	case "V0_10_1_0":
		*v = Version(sarama.V0_10_1_0)
	default:
		return errors.Errorf("`%s`: unsupported version", v)
	}
	return nil
}
func (v Version) String() string {
	switch sarama.KafkaVersion(v) {
	case sarama.V0_8_2_0:
		return "V0_8_2_0"
	case sarama.V0_8_2_1:
		return "V0_8_2_1"
	case sarama.V0_8_2_2:
		return "V0_8_2_2"
	case sarama.V0_9_0_0:
		return "V0_9_0_0"
	case sarama.V0_9_0_1:
		return "V0_9_0_1"
	case sarama.V0_10_0_0:
		return "V0_10_0_0"
	case sarama.V0_10_0_1:
		return "V0_10_0_1"
	case sarama.V0_10_1_0:
		return "V0_10_1_0"
	}
	return "Unknown"
}

// Compression is used to configure sarama's compression codec value from the
// environment.
type Compression sarama.CompressionCodec

func (sccd *Compression) Decode(v string) error {
	switch strings.ToLower(v) {
	case "none":
		*sccd = Compression(sarama.CompressionNone)
	case "snappy":
		*sccd = Compression(sarama.CompressionSnappy)
	case "lz4":
		*sccd = Compression(sarama.CompressionLZ4)
	case "gzip":
		*sccd = Compression(sarama.CompressionGZIP)
	default:
		return errors.Errorf("`%s`: unsupported compression codec", v)
	}
	return nil
}
func (sccd Compression) String() string {
	switch sarama.CompressionCodec(sccd) {
	case sarama.CompressionNone:
		return "none"
	case sarama.CompressionSnappy:
		return "snappy"
	case sarama.CompressionLZ4:
		return "lz4"
	case sarama.CompressionGZIP:
		return "gzip"
	}
	return "Unknown"
}
