// Package cli ...
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/gotd/td/tg"
	"github.com/kissanjamgit/privatestream/config"
	"github.com/kissanjamgit/privatestream/crpt"
	"github.com/kissanjamgit/privatestream/keyredirect"
	"github.com/kissanjamgit/privatestream/secretservice"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	cfg           *config.Config
	mtp           MTP
	baseURL       string
	signalContext context.Context
)

func set(c *config.Config, m MTP, s context.Context) {
	cfg = c
	mtp = m
	signalContext = s
}

var RunEShow = func(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(`len(args) < 1 `)
	}
	index := 0
	uri := fmt.Sprintf(`%s/list/%s/index/%d/list.m3u8`, baseURL, args[0], index)
	res, err := http.Get(uri)
	if err != nil {
		return err
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

var RunEPlay = func(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(`len(args) < 1 `)
	}
	index := 0
	// list/preview/index/0/list.m3u
	uri := fmt.Sprintf(`%s/list/%s/index/%d/list.m3u8`, baseURL, args[0], index)
	res, err := http.Get(uri)
	if err != nil {
		return err
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	execCmd := exec.Command(cfg.Player, cfg.PlayerArgs...)
	stdin, err := execCmd.StdinPipe()
	if err != nil {
		return err
	}

	var g errgroup.Group
	g.Go(func() error {
		_, err := stdin.Write(b)
		if err != nil {
			stdin.Close()
			return err
		}
		if err := stdin.Close(); err != nil {
			return err
		}
		return err
	})

	err = execCmd.Start()
	if err != nil {
		return err
	}
	err = g.Wait()
	if err != nil {
		return err
	}
	err = execCmd.Wait()

	return err
}

var RunEhostkey = func(cmd *cobra.Command, args []string) error {
	return nil
}

var RunEtoolEncryptAESBase64URLSafe = func(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf(`len(args) != 1`)
	}

	input := []byte(args[0])
	value, err := crpt.EncryptAESBase64URLSafe(cfg, input)
	if err != nil {
		return err
	}

	fmt.Println(value)
	return nil
}

var RunEtoolDecryptAESBase64URLSafe = func(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf(`len(args) != 1`)
	}
	value, err := crpt.DecryptAESBase64URLSafe(cfg, args[0])
	if err != nil {
		return err
	}
	fmt.Println(value)
	return nil
}

var RunERun = func(cmd *cobra.Command, args []string) (err error) {
	g, ctx := errgroup.WithContext(signalContext)
	apiChan := make(chan *tg.Client, 1)
	g.Go(func() error {
		return mtp(ctx, cfg, apiChan)
	})
	var api *tg.Client
	select {
	case api = <-apiChan:
	case <-ctx.Done():
		return g.Wait()
	}
	engin := gin.New()
	err = secretservice.Add(engin, api, ctx, cfg)
	if err != nil {
		return
	}
	keyredirect.Add(engin, cfg)

	g.Go(func() error {
		e := errors.New(`gin engin down`)
		err = e
		srv := http.Server{
			Addr:    cfg.Addr + `:` + cfg.Port,
			Handler: engin,
		}

		go func() {
			<-ctx.Done()
			srv.Shutdown(context.Background())
		}()

		err = srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return e
		}
		return err
	})

	err = g.Wait()
	return
}

type MTP func(ctx context.Context, Config *config.Config, apiChan chan<- *tg.Client) error

func New(Config *config.Config, mtp MTP, signalContext context.Context) (cbrCmd *cobra.Command) {
	set(Config, mtp, signalContext)
	baseURL = `http://` + cfg.Addr + `:` + cfg.Port
	cbrCmd = &cobra.Command{
		Use: "privatestream",
	}

	cbrCmd.AddCommand(&cobra.Command{
		Use:  `run`,
		RunE: RunERun,
	})

	cbrCmd.AddCommand(&cobra.Command{
		Use:  `show`,
		RunE: RunEShow,
	})
	cbrCmd.AddCommand(&cobra.Command{
		Use:  `play`,
		RunE: RunEPlay,
	})
	cbrCmd.AddCommand(&cobra.Command{
		Use:  `hostkey`,
		RunE: RunEhostkey,
	})
	tool := cobra.Command{Use: `tool`}
	tool.AddCommand(&cobra.Command{Use: `encryptAESBase64URLSafe`, RunE: RunEtoolEncryptAESBase64URLSafe})
	tool.AddCommand(
		&cobra.Command{Use: `decryptAESBase64URLSafe`, RunE: RunEtoolDecryptAESBase64URLSafe},
	)
	cbrCmd.AddCommand(&tool)
	return
}
