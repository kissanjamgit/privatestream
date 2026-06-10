// Package keyredirect ...
package keyredirect

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kissanjamgit/private_stream/config"
)

func Add(engin *gin.Engine, cfg *config.Config) {
	engin.GET("/key/enc.key", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, cfg.SecretKeyURI)
	})
}
