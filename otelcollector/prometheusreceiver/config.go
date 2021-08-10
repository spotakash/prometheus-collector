// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusreceiver

import (
	"fmt"
	"time"
	"os"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
	"errors"
	//"go.uber.org/zap"
	"go.opentelemetry.io/collector/config"
	//"github.com/gracewehner/prometheusreceiver/internal"
)

const (
	// The key for Prometheus scraping configs.
	prometheusConfigKey = "config"
)

// Config defines configuration for Prometheus receiver.
type Config struct {
	config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	PrometheusConfig        *promconfig.Config       `mapstructure:"-"`
	BufferPeriod            time.Duration            `mapstructure:"buffer_period"`
	BufferCount             int                      `mapstructure:"buffer_count"`
	UseStartTimeMetric      bool                     `mapstructure:"use_start_time_metric"`
	StartTimeMetricRegex    string                   `mapstructure:"start_time_metric_regex"`

	// ConfigPlaceholder is just an entry to make the configuration pass a check
	// that requires that all keys present in the config actually exist on the
	// structure, ie.: it will error if an unknown key is present.
	ConfigPlaceholder interface{} `mapstructure:"config"`
	//logger  *zap.Logger
}

var _ config.Receiver = (*Config)(nil)
var _ config.CustomUnmarshable = (*Config)(nil)

func checkFileExists(fn string) error {
	// Nothing set, nothing to error on.
	if fn == "" {
		return nil
	}
	_, err := os.Stat(fn)
	return err
}


// Validate checks the receiver configuration is valid
func (cfg *Config) Validate() error {
	//cfg.logger = internal.NewZapToGokitLogAdapter(cfg.logger)
	if cfg.PrometheusConfig == nil {
		return nil // noop receiver
	}
	if len(cfg.PrometheusConfig.ScrapeConfigs) == 0 {
		return errors.New("no Prometheus scrape_configs")
	}
	//cfg.logger.Info("Starting custom validation...\n")
	fmt.Printf("Starting custom validation...\n")
	for _, scfg := range cfg.PrometheusConfig.ScrapeConfigs {
		fmt.Printf("in bearer token file validation-HttpClientConfig- %v...\n",scfg.HTTPClientConfig)
		fmt.Printf("scrape config - %v...\n",scfg)
		if err := checkFileExists(scfg.HTTPClientConfig.BearerTokenFile); err != nil {
			fmt.Printf("error checking bearer token file %q - %s", scfg.HTTPClientConfig.BearerTokenFile, err)
			return errors.New("error checking bearer token file")
		}
	}
	return nil
}

// Unmarshal a config.Parser into the config struct.
func (cfg *Config) Unmarshal(componentParser *config.Parser) error {
	if componentParser == nil {
		return nil
	}
	// We need custom unmarshaling because prometheus "config" subkey defines its own
	// YAML unmarshaling routines so we need to do it explicitly.

	err := componentParser.UnmarshalExact(cfg)
	if err != nil {
		return fmt.Errorf("prometheus receiver failed to parse config: %s", err)
	}

	// Unmarshal prometheus's config values. Since prometheus uses `yaml` tags, so use `yaml`.
	promCfgMap := cast.ToStringMap(componentParser.Get(prometheusConfigKey))
	if len(promCfgMap) == 0 {
		return nil
	}
	out, err := yaml.Marshal(promCfgMap)
	if err != nil {
		return fmt.Errorf("prometheus receiver failed to marshal config to yaml: %s", err)
	}

	err = yaml.UnmarshalStrict(out, &cfg.PrometheusConfig)
	if err != nil {
		return fmt.Errorf("prometheus receiver failed to unmarshal yaml to prometheus config: %s", err)
	}
	return nil
}
