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
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	sarama_cluster "github.com/bsm/sarama-cluster"
	"github.com/fatih/camelcase"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// -----------------------------------------------------------------------------

// Env can be used to configure a kafka session via the environment.
//
// It comes with sane default for a local development set-up.
type Env struct {
	/* Common */
	Version Version  `envconfig:"VERSION" default:"V0_9_0_1"`
	Addrs   []string `envconfig:"ADDRS" default:"localhost:9092"`
	Bufsize int      `envconfig:"BUFSIZE" default:"4096"`

	/* Producer */
	ProdPushTopics   []string    `envconfig:"PROD_PUSH_TOPICS" default:""`
	ProdNotifSuccess bool        `envconfig:"PROD_NOTIF_SUCCESS" default:"false"`
	ProdNotifError   bool        `envconfig:"PROD_NOTIF_ERROR" default:"true"`
	ProdCompression  Compression `envconfig:"PROD_COMPRESSION" default:"none"`

	/* Consumer */
	ConsPullTopics           []string      `envconfig:"CONS_PULL_TOPICS" default:""`
	ConsGroupID              string        `envconfig:"CONS_GROUP_ID" default:""`
	ConsRoutines             uint          `envconfig:"CONS_ROUTINES" default:"32"`
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

// Config returns a `sarama_cluster.Config` using the values from the environment.
func (e *Env) Config() *sarama_cluster.Config {
	config := sarama_cluster.NewConfig()

	/* CONSUMER */

	// If enabled, rebalance notification will be returned on the
	// Notifications channel.
	config.Group.Return.Notifications = e.ConsNotifRebalance
	// If enabled, any errors that occurred while consuming are returned on
	// the Errors channel.
	config.Consumer.Return.Errors = e.ConsNotifError
	// The initial offset to use if no offset was previously committed.
	// Should be OffsetNewest or OffsetOldest.
	config.Consumer.Offsets.Initial = e.ConsOffsetInitial
	// How frequently to commit updated offsets.
	config.Consumer.Offsets.CommitInterval = e.ConsOffsetCommitInterval
	// The retention duration for committed offsets. If zero, disabled
	// (in which case the `offsets.retention.minutes` option on the
	// broker will be used).  Kafka only supports precision up to
	// milliseconds; nanoseconds will be truncated. Requires Kafka
	// broker version 0.9.0 or later.
	config.Consumer.Offsets.Retention = e.ConsOffsetRetention
	// How long to wait after a failing to read from a partition before
	// trying again.
	config.Consumer.Retry.Backoff = e.ConsRetryBackoff

	/* PRODUCER */

	// If enabled, successfully delivered messages will be returned on the
	// successes channel.
	config.Producer.Return.Successes = e.ProdNotifSuccess
	// If enabled, messages that failed to deliver will be returned on the
	// Errors channel, including error.
	config.Producer.Return.Errors = e.ProdNotifError
	// The type of compression to use on messages (defaults to no compression).
	// Similar to `compression.codec` setting of the JVM producer.
	config.Producer.Compression = sarama.CompressionCodec(e.ProdCompression)

	/* COMMON */

	// The number of events to buffer in internal and external channels. This
	// permits the producer and consumer to continue processing some messages
	// in the background while user code is working, greatly improving throughput.
	config.ChannelBufferSize = e.Bufsize

	// The version of Kafka that Sarama will assume it is running against.
	// Defaults to the oldest supported stable version. Since Kafka provides
	// backwards-compatibility, setting it to a version older than you have
	// will not break anything, although it may prevent you from using the
	// latest features. Setting it to a version greater than you are actually
	// running may lead to random breakage.
	config.Version = sarama.KafkaVersion(e.Version)

	return config
}

// -----------------------------------------------------------------------------

func (e *Env) String() string {
	cv := reflect.ValueOf(*e)
	fields := reflect.TypeOf(*e)
	fieldStrs := make([]string, fields.NumField())
	for i := 0; i < fields.NumField(); i++ {
		f := fields.Field(i)
		if len(f.Name) > 0 && strings.ToLower(f.Name[:1]) == f.Name[:1] {
			continue // private field
		}
		fNameParts := camelcase.Split(f.Name)
		for i, fnp := range fNameParts {
			fNameParts[i] = strings.ToUpper(fnp)
		}
		fName := strings.Join(fNameParts, "_")
		itf := cv.Field(i).Interface()
		if _, ok := itf.(fmt.Stringer); ok {
			fieldStrs[i] = fmt.Sprintf("%s = %s", fName, itf)
		} else {
			fieldStrs[i] = fmt.Sprintf("%s = %v", fName, itf)
		}
	}
	return strings.Join(fieldStrs, "\n") + "\n"
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
