// Package config ...
package config

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	addr = `127.0.0.1`
	// port = `30443`
	port = `54821`

	secretKeyURI    = `https://raw.githubusercontent.com/kissanjamgit/private_stream/main/key/enc.key`
	secretChannelID = 3937047128

	sessionStorage = `session.json`
)

type Config struct {
	Addr string
	Port string
	TelegramConfig
	SecretChannelID int64
	SecretKeyURI    string
	SecretKey       []byte
}

type TelegramConfig struct {
	AppID          int
	AppHash        string
	PhoneNo        string
	SessionStorage string
}

var config *Config

func NewConfig() (c *Config, err error) {
	m, err := godotenv.Read(`.env`)
	if err != nil {
		return
	}
	appID, err := strconv.Atoi(m[`appID`])
	if err != nil {
		return
	}
	res, err := http.Get(secretKeyURI)
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
	config = &Config{
		Addr:            addr,
		Port:            port,
		SecretChannelID: secretChannelID,
		SecretKeyURI:    secretKeyURI,
		SecretKey:       key,

		TelegramConfig: TelegramConfig{
			appID,
			m[`appHash`],
			m[`phoneNo`],
			sessionStorage,
		},
	}
	c = config
	return
}
