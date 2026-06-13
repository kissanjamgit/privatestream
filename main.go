// Package main does something
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/kissanjamgit/privatestream/cache"
	"github.com/kissanjamgit/privatestream/cli"
	"github.com/kissanjamgit/privatestream/config"
	"github.com/kissanjamgit/privatestream/mtp"
)

// var targetChannelID int64 = 1786343615

func block() (err error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return
	}
	cache.Add(cfg)
	ctx := context.Background()
	signalContext, _ := signal.NotifyContext(ctx, os.Interrupt)
	cbrCmd := cli.New(cfg, mtp.MtpRun, signalContext)
	err = cbrCmd.Execute()
	if err != nil {
		return
	}
	return
}

func main() {
	_ = block()
}
