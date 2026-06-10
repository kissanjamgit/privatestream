// Package exp is a package for experimenting with the gotd library.
package exp

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/go-faster/errors"
	"github.com/gotd/td/tg"
	"golang.org/x/sync/errgroup"
)

func StreamToPipe(ctx context.Context, api *tg.Client, doc *tg.Document, attrVideo *tg.DocumentAttributeVideo, pw *io.PipeWriter) (err error) {
	defer pw.Close()

	Location := &tg.InputDocumentFileLocation{
		ID:            doc.GetID(),
		AccessHash:    doc.GetAccessHash(),
		FileReference: doc.GetFileReference(),
		ThumbSize:     "",
	}

	const chunkSize = 256 * 1024
	var offset int64 = 0
	for offset < doc.Size {
		limit := chunkSize
		if offset > doc.Size {
			limit = int(doc.Size - offset)
		}
		fileInterface, err := api.UploadGetFile(ctx,
			&tg.UploadGetFileRequest{
				Location: Location,
				Limit:    limit,
				Offset:   offset,
			})
		if err != nil {
			return err
		}
		file, ok := fileInterface.(*tg.UploadFile)
		if !ok {
			err = fmt.Errorf(`fileInterface.(*tg.UploadFile) !ok`)
			return err
		}
		pw.Write(file.GetBytes())
		offset += int64(limit)
	}
	return
}

func Add(api *tg.Client, ctx context.Context, channelID int64) (err error) {
	chats, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: channelID}})
	if err != nil {
		return
	} else if len(chats.GetChats()) == 0 {
		err = fmt.Errorf(`len(chats.GetChats()) ==0`)
		return
	}
	chat := chats.GetChats()[0]
	channel, ok := chat.(*tg.Channel)
	if !ok {
		err = fmt.Errorf(`chat.(*tg.Channel) !ok`)
		return
	}
	msg, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  chat.GetID(),
			AccessHash: channel.AccessHash,
		},
		Limit: 4,
	})
	if err != nil {
		return
	}
	channelMsg, ok := msg.(*tg.MessagesChannelMessages)
	if !ok {
		err = fmt.Errorf(`msg.(*tg.MessagesChannelMessages) !ok`)
		return
	}
	for _, m := range channelMsg.GetMessages() {

		doc, attrVideo := VideoExt(m)
		if attrVideo == nil {
			continue
		}

		pr, pw := io.Pipe()

		var g errgroup.Group
		g.Go(func() error {
			return StreamToPipe(ctx, api, doc, attrVideo, pw)
		})
		_, err = io.Copy(os.Stdout, pr)
		if err != nil {
			return err
		}
		err = g.Wait()
		return err
	}
	return
}

func VideoExt(m tg.MessageClass) (doc *tg.Document, attrVideo *tg.DocumentAttributeVideo) {
	msgInterface, ok := m.(*tg.Message)
	if !ok {
		return
	}
	msgMediaInterface, ok := msgInterface.GetMedia()
	if !ok {
		return
	}
	media, ok := msgMediaInterface.(*tg.MessageMediaDocument)
	if !ok || !media.Video {
		return
	}
	docInterface, ok := media.GetDocument()
	if !ok {
		return
	}
	doc, ok = docInterface.(*tg.Document)
	if !ok {
		return
	}
	for _, atr := range doc.GetAttributes() {
		attrVideo, ok := atr.(*tg.DocumentAttributeVideo)
		if !ok {
			continue
		}
		return doc, attrVideo

	}

	return
}

func GetDocFromMessage(m tg.MessageClass) (doc *tg.Document, ok bool) {
	msg, ok := m.(*tg.Message)
	if !ok {
		return
	}

	media, ok := msg.GetMedia()
	if !ok {
		return
	}
	mediaDoc, ok := media.(*tg.MessageMediaDocument)
	if !ok {
		return
	}
	docInterface, ok := mediaDoc.GetDocument()
	if !ok {
		return
	}
	doc, ok = docInterface.(*tg.Document)
	if !ok {
		return
	}
	return
}

func DocSearch(ctx context.Context, api *tg.Client, channel *tg.Channel, req *tg.ChannelsGetMessagesRequest) (docList []*tg.Document, err error) {
	res, err := api.ChannelsGetMessages(ctx, req)
	if err != nil {
		return
	}
	messagesMessages, ok := res.(*tg.MessagesMessages)
	if !ok {
		return
	}

	if len(messagesMessages.GetMessages()) == 0 {
		return
	}

	for _, item := range messagesMessages.Messages {
		doc, ok := GetDocFromMessage(item)
		if !ok {
			continue
		}
		docList = append(docList, doc)
	}
	// doc, ok :=  GetDocFromMessage(msgClass)
	// // api.sear
	// if err != nil {
	// 	return
	// }
	return
}

func DocGet(ctx context.Context, api *tg.Client, req *tg.MessagesGetHistoryRequest) (docs []*tg.Document, err error) {
	msg, err := api.MessagesGetHistory(ctx, req)
	if err != nil {
		return
	}
	channelMsg, ok := msg.(*tg.MessagesChannelMessages)
	if !ok {
		err = fmt.Errorf(`msg.(*tg.MessagesChannelMessages) !ok`)
		return
	}
	for _, msgInterface := range channelMsg.GetMessages() {
		msg, ok := msgInterface.(*tg.Message)
		if !ok {
			continue
		}
		msgMediaInterface, ok := msg.GetMedia()
		if !ok {
			continue
		}
		msgMediaDoc, ok := msgMediaInterface.(*tg.MessageMediaDocument)
		if !ok {
			continue
		}
		docInterface, ok := msgMediaDoc.GetDocument()
		if !ok {
			continue
		}
		doc, ok := docInterface.(*tg.Document)
		if !ok {
			continue
		}
		docs = append(docs, doc)
		// for _, attrInterface := range doc.GetAttributes() {
		// 	attrFileName, ok := attrInterface.(*tg.DocumentAttributeFilename)
		// 	if !ok {
		// 		continue
		// 	}
		// 	if strings.TrimSpace(attrFileName.FileName) != `output.ts` {
		// 		continue
		// 	}
		// 	return
		//
		// }

	}
	return
}

func GetChannel(ctx context.Context, api *tg.Client, channelID int64) (channel *tg.Channel, err error) {
	chats, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: channelID}})
	if err != nil {
		return
	} else if len(chats.GetChats()) == 0 {
		err = fmt.Errorf(`len(chats.GetChats()) ==0`)
		return
	}
	chat := chats.GetChats()[0]
	channel, ok := chat.(*tg.Channel)
	if !ok {
		err = fmt.Errorf(`chat.(*tg.Channel) !ok`)
		return
	}
	return
}

func GetData(ctx context.Context, pw *io.PipeWriter, offset int64, length int64, fileSize int64, api *tg.Client, location *tg.InputDocumentFileLocation) error {
	const tgBlockAlign = 524288
	const fixedLimit = 524288
	defer pw.Close()

	if offset >= fileSize || length <= 0 {
		return nil
	}

	endOffset := offset + length
	if endOffset > fileSize {
		endOffset = fileSize
	}

	// 🚀 Snap DOWN to the nearest clean 512KB block boundary
	currentOffset := (offset / tgBlockAlign) * tgBlockAlign

	// Calculate how many leading bytes we must discard from our first 512KB chunk
	skip := offset - currentOffset

	for currentOffset < endOffset {
		if currentOffset >= fileSize {
			break
		}

		// 🚀 Always pull the maximum allowed block size for top performance
		limit := fixedLimit

		req := &tg.UploadGetFileRequest{
			Location: location,
			Offset:   currentOffset,
			Limit:    limit,
		}

		res, err := api.UploadGetFile(ctx, req)
		if err != nil {
			return errors.Wrapf(err, "mtproto fetch failed at offset %d, limit %d", currentOffset, limit)
		}

		uploadFile, ok := res.(*tg.UploadFile)
		if !ok {
			return errors.Errorf("unexpected mtproto response type: %T", res)
		}

		payload := uploadFile.GetBytes()
		if len(payload) == 0 {
			break
		}

		// Determine our slicing indices relative to the current 512KB block
		chunkStart := skip
		chunkEnd := int64(len(payload))

		// If the current block extends past what ffplay needs, trim the trailing edge
		bytesAvailableFromCurrent := currentOffset + chunkEnd
		if bytesAvailableFromCurrent > endOffset {
			chunkEnd -= (bytesAvailableFromCurrent - endOffset)
		}

		// Safe in-bounds slice verification
		if chunkStart < chunkEnd && chunkStart < int64(len(payload)) {
			chunk := payload[chunkStart:chunkEnd]
			if _, err := pw.Write(chunk); err != nil {
				return errors.Wrap(err, "failed writing chunk to pipe response")
			}
		}

		// Advance currentOffset by the full block size Telegram provided
		currentOffset += int64(len(payload))

		// Clear skip so it doesn't affect subsequent 512KB blocks
		skip = 0
	}

	return nil
}
