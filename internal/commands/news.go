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
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func CmdNews(args []string) string {
	query := ""
	if len(args) > 0 {
		query = strings.Join(args, " ")
	}
	query = strings.TrimSpace(query)

	client := &http.Client{Timeout: 5 * time.Second}
	var endpoint string
	if query == "" || strings.EqualFold(query, "today") {
		endpoint = "https://news.google.com/rss?hl=en-IN&gl=IN&ceid=IN:en"
	} else {
		endpoint = "https://news.google.com/rss/search?q=" + url.QueryEscape(query) + "&hl=en-IN&gl=IN&ceid=IN:en"
	}

	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; 0xRootShell/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "news: failed to fetch headlines: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("news: remote returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "news: read error: " + err.Error()
	}

	type Item struct {
		Title string `xml:"title"`
	}
	type Channel struct {
		Items []Item `xml:"item"`
	}
	type Rss struct {
		Channel Channel `xml:"channel"`
	}

	var r Rss
	if err := xml.Unmarshal(body, &r); err != nil {
		return "news: xml parse error: " + err.Error()
	}

	if len(r.Channel.Items) == 0 {
		return "news: no headlines found"
	}

	limit := 5
	if len(r.Channel.Items) < limit {
		limit = len(r.Channel.Items)
	}

	out := strings.Builder{}
	for i := 0; i < limit; i++ {
		title := strings.TrimSpace(r.Channel.Items[i].Title)
		title = strings.Join(strings.Fields(title), " ")
		if len(title) > 120 {
			title = title[:117] + "..."
		}
		out.WriteString(fmt.Sprintf("%d) %s\n", i+1, title))
	}
	return strings.TrimSpace(out.String())
}
