// 0xRootShell — A minimalist, aesthetic terminal for creators
// Copyright (c) 2025 Khwahish Sharma (aka 0xRootAnon)
//
// Licensed under the GNU General Public License v3.0 or later (GPLv3+).
// You may obtain a copy of the License at
// https://www.gnu.org/licenses/gpl-3.0.html
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
package commands

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type cacheEntry struct {
	Timestamp int64  `json:"ts"`
	TTL       int64  `json:"ttl"` // seconds
	Data      []byte `json:"data"`
}

func cacheFilePath(key string) string {
	_ = os.MkdirAll("data/cache", 0755)
	fn := url.QueryEscape(key) + ".json"
	return filepath.Join("data/cache", fn)
}

func writeCache(key string, data []byte, ttlSeconds int) error {
	entry := cacheEntry{
		Timestamp: time.Now().Unix(),
		TTL:       int64(ttlSeconds),
		Data:      data,
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFilePath(key), b, 0644)
}

func readCache(key string) ([]byte, bool) {
	b, err := os.ReadFile(cacheFilePath(key))
	if err != nil {
		return nil, false
	}
	var e cacheEntry
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, false
	}
	if e.TTL > 0 && time.Now().Unix()-e.Timestamp > e.TTL {
		return nil, false
	}
	return e.Data, true
}
