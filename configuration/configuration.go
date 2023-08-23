package configuration

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"log"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	FileSource   = "file"
	RemoteSource = "remote"
)

type Configuration struct {
	Schedulers []Scheduler `mapstructure:"schedulers"`
}

type Scheduler struct {
	Job  string `mapstructure:"job"`
	Cron string `mapstructure:"cron"`
	Task struct {
		URL    string `mapstructure:"url"`
		Method string `mapstructure:"method"`
		Body   string `mapstructure:"body"`
		Header string `mapstructure:"header"`
	} `mapstructure:"task"`
}

type RemoteProvider struct {
	Secure        bool
	Provider      string
	Endpoint      string
	Path          string
	SecretKeyring string
}

var Config Configuration

func Load() error {
	viper.SetConfigName("configuration")
	viper.AddConfigPath("configuration")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("[INFO] Config file changed:", e.Name)
	})
	viper.WatchConfig()
	err := viper.Unmarshal(&Config)
	if err == nil {
		fmt.Println("[INFO] Configuration file has been loaded.")
	}
	return err
}

func LoadRemote(remote RemoteProvider) error {
	var err error
	if remote.Secure {
		err = viper.AddSecureRemoteProvider(remote.Provider, remote.Endpoint, remote.Path, remote.SecretKeyring)
	} else {
		err = viper.AddRemoteProvider(remote.Provider, remote.Endpoint, remote.Path)
	}
	viper.SetConfigType(strings.ReplaceAll(filepath.Ext(remote.Path), ".", ""))
	err = viper.ReadRemoteConfig()
	if err == nil {
		err = viper.Unmarshal(&Config)
	}
	if err == nil {
		fmt.Println("[INFO] Configuration remote has been loaded.")
	}
	return err
}
