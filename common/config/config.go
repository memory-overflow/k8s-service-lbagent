package config

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/jinzhu/configor"
)

var (
	_config *config
	once    sync.Once
)

const defaultConfigPath = "conf/config.yaml"

// Get ...
func Get() *config {
	if _config == nil {
		Init(defaultConfigPath)
	}
	return _config
}

// Routes
type Route struct {
	URI         string `yaml:"uri"`
	Limit       int    `yaml:"limit"`
	ServiceName string `yaml:"service_name"`
	Namespace   string `yaml:"namespace"`
	HttpPort    int    `yaml:"http_port"`
}

type config struct {
	Debug          bool    `yaml:"debug" json:"debug" env:"DEBUG"`
	Port           int     `yaml:"port" json:"port" env:"PORT"`
	LogFile        string  `yaml:"log_file"`
	KubeConfigFile string  `yaml:"kube_config_file"`
	Routes         []Route `yaml:"routes"`
}

// Init ...
func Init(configPath string) {
	once.Do(func() {
		loader := configor.New(&configor.Config{
			AutoReload:         true,
			AutoReloadInterval: 5 * time.Second,
			AutoReloadCallback: reload,
		})
		_config = &config{}
		err := loader.Load(_config, configPath)
		if err != nil {
			GetLogger().Sugar().Fatal("load config failed: %v", err)
		}
		checker(_config)
		bytes, _ := json.Marshal(_config)
		GetLogger().Sugar().Infof("load config successfully: %s", bytes)
	})
}

func reload(conf interface{}) {
	c, ok := conf.(*config)
	if !ok {
		GetLogger().Sugar().Infof("config type mismatch %T", conf)
		return
	}
	checker(c)
	bytes, _ := json.Marshal(conf)
	GetLogger().Sugar().Infof("reload config: %s", bytes)
}

func checker(c *config) {
	if c.Port == 0 {
		c.Port = 8080
	}
}
