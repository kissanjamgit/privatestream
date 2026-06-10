// Package secretservice ...
package secretservice

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gotd/td/tg"
	"github.com/kissanjamgit/private_stream/common"
	"github.com/kissanjamgit/private_stream/config"
	"github.com/kissanjamgit/private_stream/exp"
	"golang.org/x/sync/errgroup"
)

var (
	api *tg.Client
	ctx context.Context
	cfg *config.Config

	baseURL string
)

var regexpFieldRange = regexp.MustCompile(`bytes=(\d+)-(\d*)`)

func MediaStream(c *gin.Context) {
	channel, err := exp.GetChannel(c, api, cfg.SecretChannelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	filename := c.Param(`filename`)
	_, tag, ok := strings.Cut(filename, `_`)
	if !ok {
		err = fmt.Errorf("strings.Cut(filename, `_`)")
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	}
	req := tg.MessagesSearchRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		Q:      `_` + tag + `.ts`,
		Limit:  1,
		Filter: &tg.InputMessagesFilterEmpty{},
	}

	MessageInterface, err := api.MessagesSearch(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	}

	m, ok := MessageInterface.(*tg.MessagesChannelMessages)
	if !ok {
		err = fmt.Errorf(`MessageInterface.(*tg.MessagesChannelMessages) !ok`)
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	}
	msgList := m.GetMessages()
	if len(msgList) != 1 {
		err = fmt.Errorf(`len(m.Messages) != 1`)
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	}
	msg := msgList[0]
	doc, ok := exp.GetDocFromMessage(msg)
	if !ok {
		err = fmt.Errorf(`msg.(*tg.Document) !ok`)
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	}

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
			end = doc.Size - 1
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
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, doc.Size))

	location := &tg.InputDocumentFileLocation{
		ID:            doc.GetID(),
		AccessHash:    doc.GetAccessHash(),
		FileReference: doc.GetFileReference(),
		ThumbSize:     "",
	}
	pr, pw := io.Pipe()
	var g errgroup.Group
	g.Go(func() error {
		return exp.GetData(ctx, pw, start, chunkLength, doc.Size, api, location)
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

func fileNameExt(s string) (str string, err error) {
	before, _, ok := strings.Cut(s, `_`)
	if !ok {
		err = fmt.Errorf(`strings.Cut(s, %q, %q) !ok`, " ", "")
		return
	}
	ciphertext, err := base64.StdEncoding.DecodeString(before)
	if err != nil {
		return
	}
	filename := make([]byte, len(ciphertext))

	block, err := aes.NewCipher(cfg.SecretKey)
	if err != nil {
		return
	}
	blockMode := cipher.NewCBCDecrypter(block, make([]byte, aes.BlockSize))
	blockMode.CryptBlocks(filename, ciphertext)
	str = string(filename)
	return
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
			filename, err := fileNameExt(attrFileName.FileName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{`error`: err.Error()})
				return
			}
			filename = common.CleanString(filename)
			filename = strings.TrimSpace(filename)

			entryFmt := "#EXTINF:-1,%s\n" + baseURL + "/media/%d/" + attrFileName.FileName + "\n"
			fmt.Fprintf(&buff, entryFmt, filename, doc.ID)
			break
		}

	}

	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.Itoa(buff.Len()))
	c.Writer.WriteString(buff.String())
}

func GetMasterPlaylist(c *gin.Context) {
	filaname := c.Param(`filename`)
	_, tag, ok := strings.Cut(filaname, `_`)
	if !ok {
		err := fmt.Errorf(`strings.Cut(filename, %q, %q) !ok`, " ", "")
		c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
		return
	}
	channel, err := exp.GetChannel(c, api, cfg.SecretChannelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	req := tg.MessagesSearchRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		Q:      `_` + tag + `.m3u8`,
		Limit:  1,
		Filter: &tg.InputMessagesFilterEmpty{},
	}

	MessageInterface, err := api.MessagesSearch(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	arr := MessageInterface.(*tg.MessagesChannelMessages)
	if len(arr.Messages) == 0 {
		c.JSON(http.StatusBadRequest, fmt.Errorf(`len(arr.Messages) == 0`))
		return
	}
	doc, ok := exp.GetDocFromMessage(arr.Messages[0])
	if !ok {
		c.JSON(http.StatusBadRequest, fmt.Errorf(`arr.Messages[0].(*tg.Document) !ok`))
		return
	}

	location := &tg.InputDocumentFileLocation{
		ID:            doc.GetID(),
		AccessHash:    doc.GetAccessHash(),
		FileReference: doc.GetFileReference(),
		ThumbSize:     "",
	}
	limitMax := 512 * 1024
	var logicalOffset int64 = 0
	endOffset := 0 + doc.Size
	c.Status(http.StatusOK)
	c.Header("Content-Length", fmt.Sprintf("%d", doc.Size))
	c.Header("Content-Type", "application/octet-stream")
	err = func() error {
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
				return fmt.Errorf(`uploadFileInterface.(*tg.UploadFile) !ok`)
			}
			if len(uploadFile.GetBytes()) == 0 {
				return fmt.Errorf(`len(uploadFile.GetBytes()) == 0`)
			}
			l, err := c.Writer.Write(uploadFile.GetBytes())
			if err != nil {
				return fmt.Errorf(`c.Writer.Write(uploadFile.GetBytes()) %w`, err)
			}
			logicalOffset += int64(l)
		}
		return err
	}()
}

func mediaHandler(c *gin.Context) {
	if strings.HasSuffix(c.Request.URL.Path, `.m3u8`) {
		GetMasterPlaylist(c)
		return
	}

	if strings.HasSuffix(c.Request.URL.Path, `.ts`) {
		MediaStream(c)
		return
	}
	err := fmt.Errorf(`strings.HasSuffix(c.Request.URL.Path, ?) !ok`)
	c.JSON(http.StatusBadRequest, gin.H{`error`: err.Error()})
}

func Add(engin *gin.Engine, _api *tg.Client, ctxT context.Context, _cfg *config.Config) (err error) {
	api = _api
	ctx = ctxT
	cfg = _cfg
	baseURL = `http://` + cfg.Addr + `:` + cfg.Port

	// baseURL = `http://` + cfg.Addr + `:` + cfg.Port

	engin.GET(`/list/:tag/index/:index/list.m3u`, ListPlaylist)
	engin.GET(`/media/:docID/:filename`, mediaHandler)
	return
}
