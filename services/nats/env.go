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

package nats

import (
	"net"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nats-io/go-nats"
	"github.com/pkg/errors"
)

// -----------------------------------------------------------------------------

// Env can be used to configure a NATS session via the environment.
//
// It comes with sane defaults for a local development set-up.
type Env struct {
	Addrs            []string      `envconfig:"ADDRS" default:"nats://localhost:4222"`
	AllowReconnect   bool          `envconfig:"ALLOW_RECONNECT" default:"true"`
	MaxReconnect     int           `envconfig:"MAX_RECONNECT" default:"60"`
	ReconnectWait    time.Duration `envconfig:"RECONNECT_WAIT" default:"2s"`
	Timeout          time.Duration `envconfig:"TIMEOUT" default:"2s"`
	PingInterval     time.Duration `envconfig:"PING_INTERVAL" default:"2m"`
	MaxPingsOut      int           `envconfig:"MAX_PINGS_OUT" default:"2"`
	SubChanLen       int           `envconfig:"SUBCHAN_LEN" default:"8192"`
	ReconnectBufSize int           `envconfig:"RECONNECT_BUFSIZE" default:"8388608"` // 8MB
	DialerTimeout    time.Duration `envconfig:"DIALER_TIMEOUT" default:"2s"`
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

// Config returns a `nats.Options` using the values from the environment.
func (e *Env) Config() *nats.Options {
	var opts nats.Options
	opts.Servers = e.Addrs
	opts.AllowReconnect = e.AllowReconnect
	opts.MaxReconnect = e.MaxReconnect
	opts.ReconnectWait = e.ReconnectWait
	opts.Timeout = e.Timeout
	opts.PingInterval = e.PingInterval
	opts.MaxPingsOut = e.MaxPingsOut
	opts.SubChanLen = e.SubChanLen
	opts.ReconnectBufSize = e.ReconnectBufSize
	opts.Dialer = &net.Dialer{Timeout: e.DialerTimeout}
	return &opts
}
