package awstagprocessor

import (
    "encoding/json"
    "os"
    "sync"
    "time"
)

type TagCache struct {
    data     map[string]cachedTags
    mu       sync.RWMutex
    filePath string
    ttl      time.Duration
}

type cachedTags struct {
    Tags      map[string]string
    ExpiresAt time.Time
}

func NewTagCache(file string, ttlSec int) (*TagCache, error) {
    c := &TagCache{
        data:     make(map[string]cachedTags),
        filePath: file,
        ttl:      time.Duration(ttlSec) * time.Second,
    }
    _ = c.load()
    return c, nil
}

func (c *TagCache) load() error {
    f, err := os.Open(c.filePath)
    if err != nil {
        return nil
    }
    defer f.Close()
    return json.NewDecoder(f).Decode(&c.data)
}

func (c *TagCache) Save() error {
    c.mu.RLock()
    defer c.mu.RUnlock()
    f, err := os.Create(c.filePath)
    if err != nil {
        return err
    }
    defer f.Close()
    return json.NewEncoder(f).Encode(c.data)
}

func (c *TagCache) Get(arn string) (map[string]string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    entry, ok := c.data[arn]
    if !ok || time.Now().After(entry.ExpiresAt) {
        return nil, false
    }
    return entry.Tags, true
}

func (c *TagCache) Set(arn string, tags map[string]string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[arn] = cachedTags{
        Tags:      tags,
        ExpiresAt: time.Now().Add(c.ttl),
    }
}
