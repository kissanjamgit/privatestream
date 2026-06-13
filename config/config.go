// Package config ...
package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"

	"github.com/pelletier/go-toml/v2"
)

const (
	appName = `privatestream`
	// port = `30443`

)

func sessionPathGet() (path string, err error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	appConfigDir := filepath.Join(userConfigDir, appName)

	err = os.MkdirAll(appConfigDir, os.ModePerm)
	if err != nil {
		return
	}
	path = filepath.Join(appConfigDir, `session.json`)
	return
}

type PlayerConfig struct {
	Path       string   `toml:"path"`
	PlayerArgs []string `toml:"player_args"`
}
type Config struct {
	Addr            string `toml:"addr"`
	Port            string `toml:"port"`
	TelegramConfig  `toml:"telegram"`
	PlayerConfig    `toml:"player"`
	SecretChannelID int64  `toml:"secret_channel_id"`
	SecretKeyURI    string `toml:"secret_key_uri"`
	SecretKey       []byte
	CachePath       string
}

type TelegramConfig struct {
	AppID          int    `toml:"app_id"`
	AppHash        string `toml:"app_hash"`
	PhoneNo        string `toml:"phone_no"`
	SessionStorage string `toml:"session_storage"`
}

func NewConfig() (c *Config, err error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	pathConfigFile := filepath.Join(configDir, appName, `config.toml`)
	f, err := os.ReadFile(pathConfigFile)
	if err != nil {
		return
	}

	err = toml.Unmarshal(f, &c)
	if err != nil {
		return
	}

	if c.Addr == `` || c.Port == `` || c.SecretKeyURI == `` {
		return nil, fmt.Errorf("c.Addr == `` || c.Port == `` || c.SecretKeyURI == ``")
	}
	if reflect.DeepEqual(c.TelegramConfig, PlayerConfig{}) {
		return nil, fmt.Errorf(`telegramconfig is empty`)
	}

	if reflect.DeepEqual(c.PlayerConfig, PlayerConfig{}) {
		c.PlayerConfig = PlayerConfig{Path: `mpv.exe`, PlayerArgs: []string{`-`}}
	}

	res, err := http.Get(c.SecretKeyURI)
	if err != nil {
		return
	}

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf(`res.StatusCode != http.StatusOK`)
		return
	}
	key, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	c.SecretKey = key

	pathSession, err := sessionPathGet()
	if err != nil {
		return
	}
	c.SessionStorage = pathSession
	return
}

func PathCacheGet() (path string, err error) {
	tempDir := os.TempDir()
	appTempDir := filepath.Join(tempDir, appName)
	err = os.MkdirAll(appTempDir, os.ModePerm)
	if err != nil {
		return
	}

	path = filepath.Join(appTempDir, `cache`)
	cachefile, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err != nil {
		return
	}
	defer cachefile.Close()
	return
}
