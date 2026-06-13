// Package cache ...
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gotd/td/tg"
	"github.com/kissanjamgit/privatestream/config"
)

type timeFieldValue interface {
	TimeFieldconcrete()
}

const timeEntryExpiry = time.Hour * 8

var (
	cacheMap  map[string]timeFieldEntry[*DocumentCache]
	cachePath = ""
)

type timeFieldEntry[T timeFieldValue] struct {
	Key   time.Time
	Value T
}

type DocumentCacheInterface interface {
	GetID() int64
	GetAccessHash() int64
	GetFileReference() []byte
}

type DocumentCache struct {
	ID            int64  `json:"id"`
	AccessHash    int64  `json:"access_hash"`
	Date          int    `json:"date"`
	MimeType      string `json:"mime_type"`
	Size          int64  `json:"size"`
	DCID          int    `json:"dc_id"`
	FileReference []byte `json:"file_reference"`
}

func (c *DocumentCache) GetID() int64 {
	return c.ID
}

func (c *DocumentCache) GetAccessHash() int64 {
	return c.AccessHash
}

func (c *DocumentCache) GetFileReference() []byte {
	return c.FileReference
}

func (c *DocumentCache) FromDocTG(docTG *tg.Document) {
	c.ID = docTG.ID
	c.AccessHash = docTG.AccessHash
	c.Date = docTG.Date
	c.MimeType = docTG.MimeType
	c.Size = docTG.Size
	c.DCID = docTG.DCID
	c.FileReference = docTG.FileReference
}

func (*DocumentCache) TimeFieldconcrete() {}

func Add(Config *config.Config) error {
	// Initialize the pointer map
	cacheMap = map[string]timeFieldEntry[*DocumentCache]{}
	cachePath = Config.CachePath

	f, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(f) > 0 {
		if err := json.Unmarshal(f, &cacheMap); err != nil {
			return fmt.Errorf("failed to unmarshal cache file: %w", err)
		}
	}
	return nil
}

func DocGet(filename string) (*DocumentCache, bool) {
	res, ok := cacheMap[filename]
	if time.Now().After(res.Key.Add(timeEntryExpiry)) {
		ok = false
	}
	// fmt.Fprintf(gin.DefaultWriter, "(cache) get filename: %s, ok: %t\n", filename, ok)
	return res.Value, ok // Directly returns the exact pointer cleanly
}

func DocSet(filename string, docTG *tg.Document) error {
	c := &DocumentCache{}
	c.FromDocTG(docTG)
	cacheMap[filename] = timeFieldEntry[*DocumentCache]{Key: time.Now(), Value: c}
	// fmt.Fprintf(gin.DefaultWriter, "(cache) set filename: %s, ok: %t\n", filename, c != nil)

	updatedJSON, err := json.MarshalIndent(cacheMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	return os.WriteFile(cachePath, updatedJSON, 0o644)
}
