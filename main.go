// Package main does something
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/kissanjamgit/private_stream/config"
	"github.com/kissanjamgit/private_stream/keyredirect"
	"github.com/kissanjamgit/private_stream/secretservice"
)

var (
	ctx      context.Context
	codeHook = auth.CodeAuthenticatorFunc(func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
		fmt.Print(`Enter Code`)
		code, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(code), nil
	})
)

// var targetChannelID int64 = 1786343615

func block() (err error) {
	config, err := config.NewConfig()
	if err != nil {
		return
	}
	ctx = context.Background()
	opts := telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{
			Path: config.SessionStorage,
		},
	}
	flowFn := auth.NewFlow(auth.Constant(config.PhoneNo, ``, codeHook), auth.SendCodeOptions{})
	client := telegram.NewClient(config.AppID, config.AppHash, opts)
	err = client.Run(ctx, func(ctx context.Context) (err error) {
		err = client.Auth().IfNecessary(ctx, flowFn)
		if err != nil {
			return
		}
		engin := gin.New()
		err = secretservice.Add(engin, client.API(), ctx, config)
		keyredirect.Add(engin, config)

		engin.Run(config.Addr + `:` + config.Port)
		return
	})
	return
}

func main() {
	err := block()
	if err != nil {
		panic(err)
	}
}
