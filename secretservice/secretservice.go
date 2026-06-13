// Package secretservice ...
package secretservice

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gotd/td/tg"
	"github.com/kissanjamgit/privatestream/cache"
	"github.com/kissanjamgit/privatestream/config"
	"github.com/kissanjamgit/privatestream/crpt"
	"github.com/kissanjamgit/privatestream/exp"
	"golang.org/x/sync/errgroup"
)

var (
	api *tg.Client
	ctx context.Context
	cfg *config.Config

	baseURL string
)

var regexpFieldRange = regexp.MustCompile(`bytes=(\d+)-(\d*)`)

func DocGet(filename string, c context.Context) (doc *tg.Document, err error) {
	channel, err := exp.GetChannel(c, api, cfg.SecretChannelID)
	if err != nil {
		return
	}

	req := tg.MessagesSearchRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		Q:      filename,
		Limit:  1,
		Filter: &tg.InputMessagesFilterEmpty{},
	}

	MessageInterface, err := api.MessagesSearch(ctx, &req)
	if err != nil {
		return
	}
	arr := MessageInterface.(*tg.MessagesChannelMessages)
	if len(arr.Messages) == 0 {
		err = fmt.Errorf(`len(arr.Messages) == 0`)
		return
	}

	doc, ok := exp.GetDocFromMessage(arr.Messages[0])
	if !ok {
		err = fmt.Errorf(`arr.Messages[0].(*tg.Document) !ok`)
		return
	}
	return
}

func MediaStream(c *gin.Context, filename string, doc cache.DocumentCacheInterface) {
	var err error
	var docSize int64
	switch d := doc.(type) {
	case *cache.DocumentCache:
		docSize = d.Size
	case *tg.Document:
		docSize = d.Size
	default:
		err := fmt.Errorf(`doc.(*tg.Document) !ok`)
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return

	}

	// fmt.Fprintf(gin.DefaultWriter, "uri: %s, filename: %s  docSize: %d\n", c.Request.URL.String(), filename, docSize)

	var start, end int64
	if fieldRange := c.GetHeader(`Range`); fieldRange == `` {
		err = fmt.Errorf("fieldRange == `` ")
		c.JSON(http.StatusRequestedRangeNotSatisfiable, gin.H{`error`: err.Error()})
		return
	} else {
		submatch := regexpFieldRange.FindStringSubmatch(fieldRange)
		if len(submatch) < 3 {
			err = fmt.Errorf("fieldRange == `` ")
			c.JSON(http.StatusRequestedRangeNotSatisfiable, gin.H{`error`: err.Error()})
			return
		}
		startInt, err := strconv.Atoi(submatch[1])
		if err != nil {
			c.JSON(http.StatusRequestedRangeNotSatisfiable, gin.H{`error`: err.Error()})
			return
		}
		start = int64(startInt)

		if submatch[2] == `` {
			end = docSize - 1
		} else {
			endInt, err := strconv.Atoi(submatch[2])
			if err != nil {
				c.JSON(http.StatusRequestedRangeNotSatisfiable, gin.H{`error`: err.Error()})
				return
			}
			end = int64(endInt)
		}
	}
	chunkLength := end - start + 1
	c.Status(206)
	c.Header("Content-Type", "video/mp2t")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", strconv.FormatInt(chunkLength, 10))
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, docSize))

	location := &tg.InputDocumentFileLocation{
		ID:            doc.GetID(),
		AccessHash:    doc.GetAccessHash(),
		FileReference: doc.GetFileReference(),
		ThumbSize:     "",
	}
	pr, pw := io.Pipe()
	var g errgroup.Group
	g.Go(func() error {
		return exp.GetData(ctx, pw, start, chunkLength, docSize, api, location)
	})
	_, err = io.Copy(c.Writer, pr)
	if err != nil {
		fmt.Fprintln(gin.DefaultWriter, gin.H{`error`: err.Error()})
		return
	}
	err = g.Wait()
	if err != nil {
		fmt.Fprintln(gin.DefaultWriter, gin.H{`error`: err.Error()})
		return
	}
}

func ListPlaylist(c *gin.Context) {
	tag := c.Param(`tag`)
	index := 0
	limit := 10

	if i, err := strconv.Atoi(c.Param(`index`)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	} else if i > 0 {
		index = i
		limit *= index
	}
	channel, err := exp.GetChannel(c, api, cfg.SecretChannelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	}
	req := tg.MessagesSearchRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		Q:      `_` + tag + `.m3u8`,
		Limit:  limit,
		Filter: &tg.InputMessagesFilterEmpty{},
	}

	MessageInterface, err := api.MessagesSearch(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	}

	arr := MessageInterface.(*tg.MessagesChannelMessages)
	if len(arr.Messages) == 0 {
		err = fmt.Errorf(`len(arr.Messages) == 0`)
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	}
	messages := func() []tg.MessageClass {
		pageSize := 10
		start := index * pageSize
		end := min(len(arr.Messages), start+pageSize)
		return arr.Messages[start:end]
	}()

	var buff strings.Builder
	buff.WriteString("#EXTM3U\n")
	for _, m := range messages {
		msg, ok := m.(*tg.Message)
		if !ok {
			continue
		}
		media, ok := msg.GetMedia()
		if !ok {
			continue
		}
		mediaDoc, ok := media.(*tg.MessageMediaDocument)
		if !ok {
			continue
		}
		docInterface, ok := mediaDoc.GetDocument()
		if !ok {
			continue
		}
		doc, ok := docInterface.(*tg.Document)
		if !ok {
			continue
		}
		for _, attr := range doc.GetAttributes() {
			attrFileName, ok := attr.(*tg.DocumentAttributeFilename)
			if !ok {
				continue
			}

			var fileID string
			{
				index := strings.LastIndex(attrFileName.FileName, `_`)
				if index == -1 {
					continue
				}
				fileID = attrFileName.FileName[:index]
			}

			if !ok {
				fmt.Fprintf(gin.DefaultWriter, "%v\n", err)
				continue
			}
			filename, err := crpt.DecryptAESBase64URLSafe(cfg, fileID)
			if err != nil {
				fmt.Fprintf(gin.DefaultWriter, "%v\n", err)
				continue
			}
			// filename := strings.TrimSpace(common.CleanString(fileID))
			fmt.Fprintf(&buff, "#EXTINF:-1,%s\n%s/media/%s\n", filename, baseURL, attrFileName.FileName)
		}
	}
	c.Header("Content-Type", "application/x-mpegURL")
	c.Header("Content-Length", strconv.Itoa(buff.Len()))
	c.Writer.WriteString(buff.String())
}

type GetMasterPlaylistPayloadAdaptJSON struct {
	PlaylistDocID string `json:"playlistDocID"`
}

func GetMasterPlaylist(c *gin.Context, filename string, doc cache.DocumentCacheInterface) {
	location := &tg.InputDocumentFileLocation{
		ID:            doc.GetID(),
		AccessHash:    doc.GetAccessHash(),
		FileReference: doc.GetFileReference(),
		ThumbSize:     "",
	}
	var docSize int64
	switch d := doc.(type) {
	case *cache.DocumentCache:
		docSize = d.Size
	case *tg.Document:
		docSize = d.Size

	default:
		err := fmt.Errorf(`doc.(*tg.Document) !ok`)
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	}

	limitMax := 512 * 1024
	var logicalOffset int64 = 0
	endOffset := 0 + docSize
	c.Status(http.StatusOK)
	c.Header("Content-Length", fmt.Sprintf("%d", docSize))
	c.Header("Content-Type", "application/octet-stream")

	err := func() (err error) {
		for logicalOffset < endOffset {
			req := tg.UploadGetFileRequest{
				Offset:   logicalOffset,
				Location: location,
				Limit:    limitMax,
			}
			uploadFileInterface, err := api.UploadGetFile(ctx, &req)
			if err != nil {
				return err
			}
			uploadFile, ok := uploadFileInterface.(*tg.UploadFile)
			if !ok {
				err = fmt.Errorf(`uploadFileInterface.(*tg.UploadFile) !ok`)
				return err
			}
			if len(uploadFile.GetBytes()) == 0 {
				return fmt.Errorf(`len(uploadFile.GetBytes()) == 0`)
			}
			l, err := c.Writer.Write(uploadFile.GetBytes())
			if err != nil {
				return fmt.Errorf(`c.Writer.Write(uploadFile.GetBytes()) %v`, err)
			}
			logicalOffset += int64(l)
		}
		return err
	}()
	if err != nil {
		fmt.Fprintf(gin.DefaultWriter, "GetMasterPlaylist err: %v\n", err)
	}
}

func docHandleCache(filename string) (cache.DocumentCacheInterface, error) {
	d, ok := cache.DocGet(filename)
	if ok {
		return d, nil
	}
	docTG, err := DocGet(filename, ctx)
	if err != nil {
		return nil, err
	}
	cache.DocSet(filename, docTG)
	return docTG, nil
}

func mediaHandler(c *gin.Context) {
	path := c.Request.URL.Path
	var extenstion string
	{
		index := strings.LastIndex(path, `.`)
		if index == -1 {
			c.JSON(http.StatusBadRequest, gin.H{`error`: `index == -1`})
			return
		}
		extenstion = path[index+1:]
	}

	filename := c.Param(`filename`)
	switch extenstion {
	case `m3u8`:
		doc, err := docHandleCache(filename)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
			return
		}
		GetMasterPlaylist(c, filename, doc)

	case `ts`:

		doc, err := docHandleCache(filename)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
			return
		}
		MediaStream(c, filename, doc)
	default:
		err := fmt.Errorf(`case exhaustion extenstion`)
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return

	}
}

func Add(engin *gin.Engine, _api *tg.Client, ctxTG context.Context, _cfg *config.Config) (err error) {
	api = _api
	ctx = ctxTG
	cfg = _cfg
	baseURL = `http://` + cfg.Addr + `:` + cfg.Port

	engin.GET(`/list/:tag/index/:index/list.m3u8`, ListPlaylist)

	engin.GET(`/media/:filename`, mediaHandler)
	return
}
