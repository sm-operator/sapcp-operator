package config

import (
	"github.com/kelseyhightower/envconfig"
	"sync"
	"time"
)

var (
	loadOnce sync.Once
	config   Config
)

type Config struct {
	SyncPeriod   time.Duration `envconfig:"sync_period"`
	PollInterval time.Duration `envconfig:"poll_interval"`
}

func Get() Config {
	loadOnce.Do(func() {
		config = Config{ // default values
			SyncPeriod:   60 * time.Second,
			PollInterval: 10 * time.Second,
		}
		envconfig.MustProcess("", &config)
	})
	return config
}
