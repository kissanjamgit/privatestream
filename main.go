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
	"github.com/kissanjamgit/privatestream/config"
	"github.com/kissanjamgit/privatestream/keyredirect"
	"github.com/kissanjamgit/privatestream/secretservice"
)

var codeHook = auth.CodeAuthenticatorFunc(func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Print(`Enter Code`)
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
})

// var targetChannelID int64 = 1786343615

func block() (err error) {
	config, err := config.NewConfig()
	if err != nil {
		return
	}
	ctx := context.Background()
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
		api := client.API()
		engin := gin.New()
		err = secretservice.Add(engin, api, ctx, config)
		keyredirect.Add(engin, config)
		// getJoinedChannels(ctx, api)

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

func getJoinedChannels(ctx context.Context, api *tg.Client) {
	// Fetch the user's dialogs (chats)
	dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100, // Adjust your limit as needed
	})
	if err != nil {
		fmt.Fprintf(gin.DefaultWriter, "Failed to get dialogs: %v", err)
	}

	// The response can be one of multiple types; type-switch to extract chats
	switch modifiedDialogs := dialogs.(type) {
	case *tg.MessagesDialogs:
		for _, chat := range modifiedDialogs.Chats {
			if channel, ok := chat.(*tg.Channel); ok {
				fmt.Fprintf(gin.DefaultWriter, "Channel Name: %s | ID: %d | AccessHash: %d\n",
					channel.Title, channel.ID, channel.AccessHash)
			}
		}
	}
}
