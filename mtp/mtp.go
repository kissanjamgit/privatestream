// Package mtp ...
package mtp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/kissanjamgit/privatestream/config"
)

var codeHook = auth.CodeAuthenticatorFunc(func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Println(`Enter Code`)
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
})

func MtpRun(ctx context.Context, cfg *config.Config, apiChan chan<- *tg.Client) (err error) {
	opts := telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{
			Path: cfg.SessionStorage,
		},
	}
	flowFn := auth.NewFlow(auth.Constant(cfg.PhoneNo, ``, codeHook), auth.SendCodeOptions{})
	client := telegram.NewClient(cfg.AppID, cfg.AppHash, opts)
	err = client.Run(ctx, func(ctx context.Context) (err error) {
		err = client.Auth().IfNecessary(ctx, flowFn)
		if err != nil {
			return
		}
		api := client.API()
		apiChan <- api
		<-ctx.Done()
		return errors.New(`mtp shutdown`)
	})
	return
}

// func getJoinedChannels(ctx context.Context, api *tg.Client) {
// 	// Fetch the user's dialogs (chats)
// 	dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
// 		OffsetPeer: &tg.InputPeerEmpty{},
// 		Limit:      100, // Adjust your limit as needed
// 	})
// 	if err != nil {
// 		fmt.Fprintf(gin.DefaultWriter, "Failed to get dialogs: %v", err)
// 	}
//
// 	// The response can be one of multiple types; type-switch to extract chats
// 	switch modifiedDialogs := dialogs.(type) {
// 	case *tg.MessagesDialogs:
// 		for _, chat := range modifiedDialogs.Chats {
// 			if channel, ok := chat.(*tg.Channel); ok {
// 				fmt.Fprintf(gin.DefaultWriter, "Channel Name: %s | ID: %d | AccessHash: %d\n",
// 					channel.Title, channel.ID, channel.AccessHash)
// 			}
// 		}
// 	}
// }
