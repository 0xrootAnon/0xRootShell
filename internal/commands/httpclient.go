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
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

var httpClient = &http.Client{
	Timeout: 8 * time.Second,
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func httpGetWithRetries(url string) ([]byte, int, error) {
	const maxAttempts = 4
	backoff := 250 * time.Millisecond

	var lastErr error
	ua := "0xRootShell/1.0 (+https://github.com/0xRootAnon/0xRootShell)"

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", ua)
		resp, err := httpClient.Do(req)
		if err == nil {
			body, rerr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if rerr == nil {
				return body, resp.StatusCode, nil
			}
			lastErr = rerr
		} else {
			lastErr = err
		}

		if attempt == maxAttempts {
			break
		}

		jitter := time.Duration(rand.Intn(150)) * time.Millisecond
		sleep := backoff + jitter
		time.Sleep(sleep)
		backoff *= 2
	}

	return nil, 0, fmt.Errorf("httpGetWithRetries: last error: %v", lastErr)
}
