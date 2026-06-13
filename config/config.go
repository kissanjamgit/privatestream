// Package config ...
package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

const (
	appName = `privatestream`
	addr    = `127.0.0.1`
	// port = `30443`
	port = `54821`

	secretKeyURI    = `https://raw.githubusercontent.com/kissanjamgit/privatestream/master/key/enc.key`
	secretChannelID = 3937047128

	// sessionStorage = `session.json`
)

var config *Config

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
	Player     string
	PlayerArgs []string
}
type Config struct {
	Addr string
	Port string
	TelegramConfig
	PlayerConfig
	SecretChannelID int64
	SecretKeyURI    string
	SecretKey       []byte
	CachePath       string
}

type TelegramConfig struct {
	AppID          int
	AppHash        string
	PhoneNo        string
	SessionStorage string
}

func NewConfig() (c *Config, err error) {
	var g errgroup.Group
	var key []byte
	g.Go(func() error {
		var err error
		key, err = os.ReadFile(`key/enc.key`)
		if err != nil {
			return err
		}
		return nil
	})
	m, err := godotenv.Read(`.env`)
	if err != nil {
		return
	}
	appID, err := strconv.Atoi(m[`appID`])
	if err != nil {
		return
	}
	// res, err := http.Get(secretKeyURI)
	// if err != nil {
	// 	return
	// }
	// if res.StatusCode != http.StatusOK {
	// 	err = fmt.Errorf(`res.StatusCode != http.StatusOK`)
	// 	return
	// }
	// key, err := io.ReadAll(res.Body)
	// if err != nil {
	// 	return
	// }
	sessionPath, err := sessionPathGet()
	if err != nil {
		return
	}
	pathCache, err := PathCacheGet()
	if err != nil {
		return
	}
	config = &Config{
		Addr:            addr,
		Port:            port,
		SecretChannelID: secretChannelID,
		SecretKeyURI:    secretKeyURI,
		SecretKey:       key,
		CachePath:       pathCache,

		TelegramConfig: TelegramConfig{
			appID,
			m[`appHash`],
			m[`phoneNo`],
			sessionPath,
		},

		PlayerConfig: PlayerConfig{
			Player:     `mpv.exe`,
			PlayerArgs: []string{`-`},
		},
	}
	c = config
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
