// Copyright (c) 2018 The Jaeger Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package badger

import (
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Options store storage plugin related configs
type Options struct {
	Primary NamespaceConfig `mapstructure:",squash"`
	// This storage plugin does not support additional namespaces
}

// NamespaceConfig is badger's internal configuration data
type NamespaceConfig struct {
	namespace      string
	SpanStoreTTL   time.Duration `mapstructure:"span_store_ttl"`
	ValueDirectory string        `mapstructure:"directory_value"`
	KeyDirectory   string        `mapstructure:"directory_key"`
	// Setting this to true will ignore ValueDirectory and KeyDirectory
	Ephemeral             bool          `mapstructure:"ephemeral"`
	SyncWrites            bool          `mapstructure:"consistency"`
	MaintenanceInterval   time.Duration `mapstructure:"maintenance_interval"`
	MetricsUpdateInterval time.Duration `mapstructure:"metrics_update_interval"`
	ReadOnly              bool          `mapstructure:"read_only"`
}

// TODO deprecated flag to be removed
const truncateWarning = "(deprecated, will be removed after 2021-09-21 or in release v1.26.0, whichever is later)"

const (
	defaultMaintenanceInterval   time.Duration = 5 * time.Minute
	defaultMetricsUpdateInterval time.Duration = 10 * time.Second
	defaultTTL                   time.Duration = time.Hour * 72
)

const (
	suffixKeyDirectory        = ".directory-key"
	suffixValueDirectory      = ".directory-value"
	suffixEphemeral           = ".ephemeral"
	suffixSpanstoreTTL        = ".span-store-ttl"
	suffixSyncWrite           = ".consistency"
	suffixMaintenanceInterval = ".maintenance-interval"
	suffixMetricsInterval     = ".metrics-update-interval" // Intended only for testing purposes
	suffixTruncate            = ".truncate"
	suffixReadOnly            = ".read-only"
	defaultDataDir            = string(os.PathSeparator) + "data"
	defaultValueDir           = defaultDataDir + string(os.PathSeparator) + "values"
	defaultKeysDir            = defaultDataDir + string(os.PathSeparator) + "keys"
)

// NewOptions creates a new Options struct.
func NewOptions(primaryNamespace string, otherNamespaces ...string) *Options {

	defaultBadgerDataDir := getCurrentExecutableDir()

	options := &Options{
		Primary: NamespaceConfig{
			namespace:             primaryNamespace,
			SpanStoreTTL:          defaultTTL,
			SyncWrites:            false, // Performance over durability
			Ephemeral:             true,  // Default is ephemeral storage
			ValueDirectory:        defaultBadgerDataDir + defaultValueDir,
			KeyDirectory:          defaultBadgerDataDir + defaultKeysDir,
			MaintenanceInterval:   defaultMaintenanceInterval,
			MetricsUpdateInterval: defaultMetricsUpdateInterval,
		},
	}

	return options
}

func getCurrentExecutableDir() string {
	// We ignore the error, this will fail later when trying to start the store
	exec, _ := os.Executable()
	return filepath.Dir(exec)
}

// AddFlags adds flags for Options
func (opt *Options) AddFlags(flagSet *flag.FlagSet) {
	addFlags(flagSet, opt.Primary)
}

func addFlags(flagSet *flag.FlagSet, nsConfig NamespaceConfig) {
	flagSet.Bool(
		nsConfig.namespace+suffixEphemeral,
		nsConfig.Ephemeral,
		"Mark this storage ephemeral, data is stored in tmpfs.",
	)
	flagSet.Duration(
		nsConfig.namespace+suffixSpanstoreTTL,
		nsConfig.SpanStoreTTL,
		"How long to store the data. Format is time.Duration (https://golang.org/pkg/time/#Duration)",
	)
	flagSet.String(
		nsConfig.namespace+suffixKeyDirectory,
		nsConfig.KeyDirectory,
		"Path to store the keys (indexes), this directory should reside in SSD disk. Set ephemeral to false if you want to define this setting.",
	)
	flagSet.String(
		nsConfig.namespace+suffixValueDirectory,
		nsConfig.ValueDirectory,
		"Path to store the values (spans). Set ephemeral to false if you want to define this setting.",
	)
	flagSet.Bool(
		nsConfig.namespace+suffixSyncWrite,
		nsConfig.SyncWrites,
		"If all writes should be synced immediately to physical disk. This will impact write performance.",
	)
	flagSet.Duration(
		nsConfig.namespace+suffixMaintenanceInterval,
		nsConfig.MaintenanceInterval,
		"How often the maintenance thread for values is ran. Format is time.Duration (https://golang.org/pkg/time/#Duration)",
	)
	flagSet.Duration(
		nsConfig.namespace+suffixMetricsInterval,
		nsConfig.MetricsUpdateInterval,
		"How often the badger metrics are collected by Jaeger. Format is time.Duration (https://golang.org/pkg/time/#Duration)",
	)
	flagSet.Bool(
		nsConfig.namespace+suffixTruncate,
		false,
		truncateWarning+" If write-ahead-log should be truncated on restart. This will cause data loss.",
	)
	flagSet.Bool(
		nsConfig.namespace+suffixReadOnly,
		nsConfig.ReadOnly,
		"Allows to open badger database in read only mode. Multiple instances can open same database in read-only mode. Values still in the write-ahead-log must be replayed before opening.",
	)
}

// InitFromViper initializes Options with properties from viper
func (opt *Options) InitFromViper(v *viper.Viper, logger *zap.Logger) {
	initFromViper(&opt.Primary, v, logger)
}

func initFromViper(cfg *NamespaceConfig, v *viper.Viper, logger *zap.Logger) {
	cfg.Ephemeral = v.GetBool(cfg.namespace + suffixEphemeral)
	cfg.KeyDirectory = v.GetString(cfg.namespace + suffixKeyDirectory)
	cfg.ValueDirectory = v.GetString(cfg.namespace + suffixValueDirectory)
	cfg.SyncWrites = v.GetBool(cfg.namespace + suffixSyncWrite)
	cfg.SpanStoreTTL = v.GetDuration(cfg.namespace + suffixSpanstoreTTL)
	cfg.MaintenanceInterval = v.GetDuration(cfg.namespace + suffixMaintenanceInterval)
	cfg.MetricsUpdateInterval = v.GetDuration(cfg.namespace + suffixMetricsInterval)
	cfg.ReadOnly = v.GetBool(cfg.namespace + suffixReadOnly)
	if v.IsSet(cfg.namespace + suffixTruncate) {
		logger.Warn("NOTE: Deprecated flag --badger.truncate passed " + truncateWarning)
	}
}

// GetPrimary returns the primary namespace configuration
func (opt *Options) GetPrimary() NamespaceConfig {
	return opt.Primary
}
