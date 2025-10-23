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
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	AppName    = "NetPulse"
	AppVersion = "1.0.0"
	UserAgent  = AppName + "/" + AppVersion
)

const asciiArt = `
 /$$   /$$             /$$     /$$$$$$$            /$$                    
| $$$ | $$            | $$    | $$__  $$          | $$                    
| $$$$| $$  /$$$$$$  /$$$$$$  | $$  \ $$ /$$   /$$| $$  /$$$$$$$  /$$$$$$ 
| $$ $$ $$ /$$__  $$|_  $$_/  | $$$$$$$/| $$  | $$| $$ /$$_____/ /$$__  $$
| $$  $$$$| $$$$$$$$  | $$    | $$____/ | $$  | $$| $$|  $$$$$$ | $$$$$$$$
| $$\  $$$| $$_____/  | $$ /$$| $$      | $$  | $$| $$ \____  $$| $$_____/
| $$ \  $$|  $$$$$$$  |  $$$$/| $$      |  $$$$$$/| $$ /$$$$$$$/|  $$$$$$$
|__/  \__/ \_______/   \___/  |__/       \______/ |__/|_______/  \_______/
			© 2025 0xRootAnon. All rights reserved.
`

func centerPrintTo(w io.Writer, s string, defaultWidth int) {
	width := defaultWidth
	if wenv := os.Getenv("NETPULSE_WIDTH"); wenv != "" {
		var ww int
		if _, err := fmt.Sscanf(wenv, "%d", &ww); err == nil && ww > 0 {
			width = ww
		}
	}
	for _, line := range splitLines(s) {
		padding := (width - len(line)) / 2
		if padding < 0 {
			padding = 0
		}
		fmt.Fprintf(w, "%s%s\n", spaces(padding), line)
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start <= len(s)-1 {
		out = append(out, s[start:])
	}
	return out
}

func spaces(n int) string { return fmt.Sprintf("%*s", n, "") }

type ConfigXML struct {
	XMLName      xml.Name     `xml:"settings"`
	Client       ClientXML    `xml:"client"`
	ServerConfig ServerConfig `xml:"server-config"`
	Download     AttrsXML     `xml:"download"`
	Upload       AttrsXML     `xml:"upload"`
}

type ClientXML struct {
	IP  string `xml:"ip,attr"`
	ISP string `xml:"isp,attr"`
	Lat string `xml:"lat,attr"`
	Lon string `xml:"lon,attr"`
}

type ServerConfig struct {
	ThreadCount int    `xml:"threadcount,attr"`
	IgnoreIDs   string `xml:"ignoreids,attr"`
}

type AttrsXML struct {
	TestLength int `xml:"testlength,attr"`
	Threads    int `xml:"threads,attr"`
	MaxChunk   int `xml:"maxchunkcount,attr"`
	ThreadsPer int `xml:"threadsperurl,attr"`
}

type ClientConfig struct {
	Source      string
	Timeout     time.Duration
	Secure      bool
	PreAllocate bool
}

type Server struct {
	URL      string
	Scheme   string
	Host     string
	Sponsor  string
	Name     string
	Country  string
	ID       int
	Latency  float64
	Distance float64
}

type Results struct {
	Download   float64
	Upload     float64
	Ping       float64
	BytesSent  int64
	BytesRecvd int64
	Server     *Server
	ClientIP   string
	ISP        string
}

type Client struct {
	cfg       ClientConfig
	http      *http.Client
	best      *Server
	servers   []*Server
	results   Results
	clientLat float64
	clientLon float64
}

func NewClient(cfg ClientConfig) (*Client, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	if cfg.Source != "" {
		ip := net.ParseIP(cfg.Source)
		if ip == nil {
			return nil, fmt.Errorf("invalid source IP: %s", cfg.Source)
		}
		local := &net.TCPAddr{IP: ip}
		dialer := &net.Dialer{
			Timeout:   cfg.Timeout,
			LocalAddr: local,
		}
		transport.DialContext = dialer.DialContext
	} else {
		dialer := &net.Dialer{
			Timeout: cfg.Timeout,
		}
		transport.DialContext = dialer.DialContext
	}

	client := &http.Client{
		Transport:     transport,
		Timeout:       cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}

	return &Client{
		cfg:  cfg,
		http: client,
	}, nil
}

func baseFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		raw = strings.TrimSpace(raw)
		if strings.HasPrefix(raw, "http") {
			if idx := strings.Index(raw[7:], "/"); idx >= 0 {
				return raw[:idx+7]
			}
			return raw
		}
		return raw
	}
	return u.Scheme + "://" + u.Host
}

func distance(lat1, lon1, lat2, lon2 float64) float64 {
	radius := 6371.0
	dlat := (lat2 - lat1) * (math.Pi / 180.0)
	dlon := (lon2 - lon1) * (math.Pi / 180.0)
	lat1r := lat1 * (math.Pi / 180.0)
	lat2r := lat2 * (math.Pi / 180.0)
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1r)*math.Cos(lat2r)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1+a))
	return radius * c
}

func (c *Client) GetConfig(ctx context.Context) error {
	scheme := "http"
	if c.cfg.Secure {
		scheme = "https"
	}
	u := scheme + "://www.speedtest.net/speedtest-config.php"
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("config fetch error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("config returned status %d", resp.StatusCode)
	}
	var cfgXML ConfigXML
	if err := xml.NewDecoder(resp.Body).Decode(&cfgXML); err != nil {
		return fmt.Errorf("config xml parse: %w", err)
	}
	lat, _ := strconv.ParseFloat(cfgXML.Client.Lat, 64)
	lon, _ := strconv.ParseFloat(cfgXML.Client.Lon, 64)
	c.clientLat = lat
	c.clientLon = lon
	c.results.ClientIP = cfgXML.Client.IP
	c.results.ISP = cfgXML.Client.ISP
	return nil
}

func (c *Client) GetServers(ctx context.Context) error {
	scheme := "http"
	if c.cfg.Secure {
		scheme = "https"
	}
	candidates := []string{
		scheme + "://www.speedtest.net/speedtest-servers.php",
		"http://c.speedtest.net/speedtest-servers.php",
	}
	var lastErr error
	for _, u := range candidates {
		req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
		req.Header.Set("User-Agent", UserAgent)
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("status %d from %s", resp.StatusCode, u)
			resp.Body.Close()
			continue
		}
		dec := xml.NewDecoder(resp.Body)
		var servers []*Server
		for {
			tok, err := dec.Token()
			if err != nil {
				if err == io.EOF {
					break
				}
				lastErr = err
				break
			}
			switch se := tok.(type) {
			case xml.StartElement:
				if se.Name.Local == "server" {
					var attr struct {
						URL     string `xml:"url,attr"`
						Lat     string `xml:"lat,attr"`
						Lon     string `xml:"lon,attr"`
						Sponsor string `xml:"sponsor,attr"`
						Name    string `xml:"name,attr"`
						Country string `xml:"country,attr"`
						ID      int    `xml:"id,attr"`
					}
					if err := dec.DecodeElement(&attr, &se); err != nil {
						continue
					}
					lat, _ := strconv.ParseFloat(attr.Lat, 64)
					lon, _ := strconv.ParseFloat(attr.Lon, 64)
					d := distance(c.clientLat, c.clientLon, lat, lon)
					base := baseFromURL(attr.URL)
					u2, _ := url.Parse(base)
					s := &Server{
						URL:      attr.URL,
						Scheme:   u2.Scheme,
						Host:     u2.Host,
						Sponsor:  attr.Sponsor,
						Name:     attr.Name,
						Country:  attr.Country,
						ID:       attr.ID,
						Distance: d,
					}
					servers = append(servers, s)
				}
			}
		}
		resp.Body.Close()
		if len(servers) > 0 {
			c.servers = servers
			return nil
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no servers discovered")
	}
	return lastErr
}

func (c *Client) SetMiniServer(ctx context.Context, miniURL string) error {
	u, err := url.Parse(miniURL)
	if err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", miniURL, nil)
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("mini server connect: %w", err)
	}
	resp.Body.Close()
	c.servers = []*Server{{
		URL:      miniURL,
		Scheme:   u.Scheme,
		Host:     u.Host,
		Sponsor:  "Speedtest Mini",
		Name:     u.Host,
		Country:  "",
		ID:       0,
		Distance: 0,
	}}
	return nil
}

func (c *Client) GetClosestServers(n int) error {
	if len(c.servers) == 0 {
		if err := c.GetServers(context.Background()); err != nil {
			return err
		}
	}
	for i := 0; i < len(c.servers)-1; i++ {
		for j := i + 1; j < len(c.servers); j++ {
			if c.servers[j].Distance < c.servers[i].Distance {
				c.servers[j], c.servers[i] = c.servers[i], c.servers[j]
			}
		}
	}
	if n < 1 || n > len(c.servers) {
		n = len(c.servers)
	}
	c.servers = c.servers[:n]
	return nil
}

func (c *Client) pingServer(ctx context.Context, s *Server) float64 {
	trials := 3
	var total float64
	base := s.Scheme + "://" + s.Host
	for i := 0; i < trials; i++ {
		u := base + "/latency.txt?x=" + strconv.FormatInt(time.Now().UnixNano(), 10) + fmt.Sprintf(".%d", i)
		req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
		req.Header.Set("User-Agent", UserAgent)
		start := time.Now()
		resp, err := c.http.Do(req)
		if err != nil {
			total += 3600.0
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		el := time.Since(start).Seconds() * 1000.0
		if resp.StatusCode == 200 {
			total += el
		} else {
			total += 3600.0
		}
	}
	avg := total / float64(trials)
	return avg
}

func (c *Client) GetBestServer(ctx context.Context) (*Server, error) {
	if len(c.servers) == 0 {
		return nil, errors.New("no servers to choose from")
	}
	type res struct {
		s *Server
		l float64
	}
	ch := make(chan res, len(c.servers))
	var wg sync.WaitGroup
	for _, s := range c.servers {
		wg.Add(1)
		go func(sv *Server) {
			defer wg.Done()
			lat := c.pingServer(ctx, sv)
			ch <- res{s: sv, l: lat}
		}(s)
	}
	wg.Wait()
	close(ch)
	bestL := math.MaxFloat64
	var best *Server
	for r := range ch {
		r.s.Latency = r.l
		if r.l < bestL {
			bestL = r.l
			best = r.s
		}
	}
	if best == nil {
		return nil, errors.New("no best server found")
	}
	c.best = best
	c.results.Ping = best.Latency
	c.results.Server = best
	return best, nil
}

func (c *Client) Download(ctx context.Context, single bool, bytes bool, out io.Writer) (float64, error) {
	if c.best == nil {
		return 0, errors.New("best server not selected")
	}
	base := c.best.Scheme + "://" + c.best.Host
	sizes := []int{350, 500, 750, 1000, 1500, 2000, 2500, 3000}
	threads := 4
	if single {
		threads = 1
	}
	urls := make([]string, 0, len(sizes))
	for _, s := range sizes {
		urls = append(urls, fmt.Sprintf("%s/random%dx%d.jpg", base, s, s))
	}
	sem := make(chan struct{}, threads)
	resultCh := make(chan int64, len(urls))
	progressCh := make(chan int64)
	var wg sync.WaitGroup
	start := time.Now()
	for _, u := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
			req.Header.Set("User-Agent", UserAgent)
			resp, err := c.http.Do(req)
			if err != nil || resp == nil {
				return
			}
			n, _ := io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			select {
			case progressCh <- n:
			default:
			}
			resultCh <- n
		}(u)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		var acc int64
		spinner := []rune{'|', '/', '-', '\\'}
		si := 0
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case n, ok := <-progressCh:
				if !ok {
					el := time.Since(start).Seconds()
					if el <= 0 {
						el = 0.000001
					}
					bps := float64(acc*8) / el
					val, unit := formatRate(bps, bytes)
					fmt.Fprintf(out, "\rDownload: %.2f %s    \n", val, unit)
					return
				}
				acc += n
			case <-ticker.C:
				el := time.Since(start).Seconds()
				if el <= 0 {
					el = 0.000001
				}
				bps := float64(acc*8) / el
				val, unit := formatRate(bps, bytes)
				fmt.Fprintf(out, "\rTesting download... %c  downloaded: %.2f MB (%0.2f %s)", spinner[si%len(spinner)], float64(acc)/1e6, val, unit)
				si++
			}
		}
	}()

	wg.Wait()
	close(progressCh)
	close(resultCh)
	<-done

	var total int64
	for v := range resultCh {
		total += v
	}
	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 0.000001
	}
	bps := float64(total*8) / elapsed
	c.results.BytesRecvd = total
	c.results.Download = bps
	return bps, nil
}

func (c *Client) Upload(ctx context.Context, single bool, bytes bool, out io.Writer) (float64, error) {
	var actualSent int64
	if c.best == nil {
		return 0, errors.New("best server not selected")
	}
	uploadURL := ""
	if c.best != nil && c.best.URL != "" {
		if u, err := url.Parse(c.best.URL); err == nil {
			dir := path.Dir(u.Path)
			u.Path = path.Join(dir, "upload.php")
			u.RawQuery = ""
			uploadURL = u.Scheme + "://" + u.Host + u.Path
		}
	}
	if uploadURL == "" {
		uploadURL = c.best.Scheme + "://" + c.best.Host + "/upload.php"
	}
	sizes := []int{32768, 65536, 131072, 262144, 524288}
	threads := 4
	if single {
		threads = 1
	}
	requestCount := 6
	sem := make(chan struct{}, threads)
	resultCh := make(chan int64, requestCount)
	progressCh := make(chan int64)
	var wg sync.WaitGroup
	start := time.Now()
	var expected int64
	selSizes := make([]int, 0, requestCount)
	for i := 0; i < requestCount; i++ {
		sz := sizes[rand.Intn(len(sizes))]
		selSizes = append(selSizes, sz)
		expected += int64(sz)
	}

	for _, sz := range selSizes {
		wg.Add(1)
		go func(sz int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			cr := &countingReader{r: NewPatternReader(sz), total: &actualSent}
			req, _ := http.NewRequestWithContext(ctx, "POST", uploadURL+"?x="+strconv.FormatInt(time.Now().UnixNano(), 10), cr)
			req.Header.Set("User-Agent", UserAgent)
			req.Header.Set("Content-Type", "application/octet-stream")
			req.ContentLength = int64(sz)
			resp, err := c.http.Do(req)
			if err == nil && resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
			select {
			case progressCh <- int64(sz):
			default:
			}
			resultCh <- int64(sz)
		}(sz)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		spinner := []rune{'|', '/', '-', '\\'}
		si := 0
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case _, ok := <-progressCh:
				if !ok {
					total := atomic.LoadInt64(&actualSent)
					el := time.Since(start).Seconds()
					if el <= 0 {
						el = 0.000001
					}
					bps := float64(total*8) / el
					val, unit := formatRate(bps, bytes)
					fmt.Fprintf(out, "\rUpload: %.2f %s    \n", val, unit)
					return
				}
			case <-ticker.C:
				total := atomic.LoadInt64(&actualSent)
				el := time.Since(start).Seconds()
				if el <= 0 {
					el = 0.000001
				}
				bps := float64(total*8) / el
				val, unit := formatRate(bps, bytes)
				fmt.Fprintf(out, "\rTesting upload... %c  uploaded: %.2f/%.2f MB (%0.2f %s)", spinner[si%len(spinner)], float64(total)/1e6, float64(expected)/1e6, val, unit)
				si++
			}
		}
	}()

	wg.Wait()
	close(progressCh)
	close(resultCh)
	<-done

	total := atomic.LoadInt64(&actualSent)
	if total == 0 {
		return 0, fmt.Errorf("zero bytes uploaded (actualSent==0); check upload URL, server response, or increase timeout")
	}
	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 0.000001
	}
	bps := float64(total*8) / elapsed
	c.results.BytesSent = total
	c.results.Upload = bps
	return bps, nil
}

type countingReader struct {
	r     io.Reader
	total *int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		atomic.AddInt64(c.total, int64(n))
	}
	return n, err
}

type PatternReader struct {
	rem     int
	pattern []byte
	pos     int
}

func NewPatternReader(n int) *PatternReader {
	p := []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	return &PatternReader{rem: n, pattern: p}
}

func (r *PatternReader) Read(p []byte) (int, error) {
	if r.rem <= 0 {
		return 0, io.EOF
	}
	n := len(p)
	if n > r.rem {
		n = r.rem
	}
	for i := 0; i < n; i++ {
		p[i] = r.pattern[(r.pos+i)%len(r.pattern)]
	}
	r.pos = (r.pos + n) % len(r.pattern)
	r.rem -= n
	return n, nil
}

func (c *Client) BestServer() *Server { return c.best }
func (c *Client) Results() Results    { return c.results }
func (c *Client) ClientIP() string    { return c.results.ClientIP }
func (c *Client) ClientInfo() map[string]string {
	return map[string]string{
		"ip":  c.results.ClientIP,
		"isp": c.results.ISP,
	}
}

const (
	colorReset  = "\x1b[0m"
	colorRed    = "\x1b[31m"
	colorGreen  = "\x1b[32m"
	colorYellow = "\x1b[33m"
	colorBlue   = "\x1b[34m"
)

func colorize(s, color string, enabled bool) string {
	if !enabled {
		return s
	}
	return color + s + colorReset
}

func colorForPing(ms float64) string {
	if ms < 50 {
		return colorGreen
	} else if ms < 150 {
		return colorYellow
	}
	return colorRed
}

func colorForSpeed(mbps float64) string {
	if mbps >= 100 {
		return colorGreen
	} else if mbps >= 20 {
		return colorYellow
	}
	return colorRed
}

func formatRate(bps float64, bytes bool) (float64, string) {
	if bytes {
		val := (bps / 8.0) / 1e6
		return val, "MB/s"
	}
	val := bps / 1e6
	return val, "Mbps"
}

// CmdSpeedtest runs the full NetPulse speedtest implementation and returns all output as a string.
// args is a slice of arguments like ["run","--simple"] or flags like ["--simple","--timeout","15"].
// If args is nil or empty, defaults are used (which mimic running the binary without flags).
func CmdSpeedtest(args []string) string {
	var buf bytes.Buffer
	if err := runSpeedtest(&buf, args); err != nil {
		out := buf.String()
		if out != "" {
			return out + "\nERROR: " + err.Error()
		}
		return "speedtest error: " + err.Error()
	}
	return buf.String()
}

var _ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func _sanitizeForStream(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = _ansiRe.ReplaceAllString(s, "")
	return s
}

type chanWriter struct {
	ch  chan string
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *chanWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, _ := w.buf.Write(p)
	for {
		b := w.buf.Bytes()
		idx := bytes.IndexByte(b, '\n')
		if idx < 0 {
			break
		}
		line := string(b[:idx])
		select {
		case w.ch <- _sanitizeForStream(line):
		default:
		}
		w.buf.Next(idx + 1)
	}
	return n, nil
}

func CmdSpeedtestStream(args []string, ch chan string) {
	w := &chanWriter{ch: ch}
	err := runSpeedtest(w, args)
	w.mu.Lock()
	rem := w.buf.String()
	w.mu.Unlock()
	if rem != "" {
		select {
		case ch <- _sanitizeForStream(rem):
		default:
		}
	}
	if err != nil {
		select {
		case ch <- "ERROR: " + err.Error():
		default:
		}
	}
}
func runSpeedtest(out io.Writer, args []string) error {
	rand.Seed(time.Now().UnixNano())

	fs := flag.NewFlagSet("netpulse-speedtest", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	showBanner := fs.Bool("banner", true, "Show ASCII banner on startup")
	bannerWidth := fs.Int("banner-width", 80, "Banner width for centering (overridden by NETPULSE_WIDTH env var)")
	noDownload := fs.Bool("no-download", false, "Do not perform download test")
	noUpload := fs.Bool("no-upload", false, "Do not perform upload test")
	single := fs.Bool("single", false, "Use single connection")
	bytesFlag := fs.Bool("bytes", false, "Display values in bytes instead of bits")
	share := fs.Bool("share", false, "Generate URL to share results (stub)")
	simple := fs.Bool("simple", false, "Show only basic information")
	csvFlag := fs.Bool("csv", false, "CSV output")
	jsonFlag := fs.Bool("json", false, "JSON output")
	csvDelimiter := fs.String("csv-delimiter", ",", "CSV delimiter (single character)")
	csvHeader := fs.Bool("csv-header", false, "Print CSV header")
	list := fs.Bool("list", false, "List servers")
	serverIDs := fs.String("server", "", "Comma-separated server IDs to use")
	mini := fs.String("mini", "", "URL of the Speedtest Mini server")
	source := fs.String("source", "", "Source IP address to bind to")
	timeout := fs.Int("timeout", 10, "HTTP timeout in seconds")
	secure := fs.Bool("secure", false, "Use HTTPS to talk to speedtest.net")
	noPreAllocate := fs.Bool("no-pre-allocate", false, "Don't pre-allocate upload buffer")
	version := fs.Bool("version", false, "Show version")
	noColor := fs.Bool("no-color", false, "Disable ANSI colors in output")
	_ = serverIDs
	if args == nil {
		args = []string{}
	}
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag parse error: %w", err)
	}

	if *version {
		fmt.Fprintf(out, "%s %s\n", AppName, AppVersion)
		return nil
	}

	if *noDownload && *noUpload {
		return fmt.Errorf("cannot supply both --no-download and --no-upload")
	}
	if len(*csvDelimiter) != 1 {
		return fmt.Errorf("--csv-delimiter must be a single character")
	}

	cfg := ClientConfig{
		Source:      *source,
		Timeout:     time.Duration(*timeout) * time.Second,
		Secure:      *secure,
		PreAllocate: !*noPreAllocate,
	}

	if cfg.Source != "" {
		ip := net.ParseIP(cfg.Source)
		if ip == nil {
			return fmt.Errorf("invalid source IP")
		}
		found := false
		ifaces, err := net.Interfaces()
		if err == nil {
			for _, ifi := range ifaces {
				addrs, _ := ifi.Addrs()
				for _, a := range addrs {
					var addrIP net.IP
					switch v := a.(type) {
					case *net.IPNet:
						addrIP = v.IP
					case *net.IPAddr:
						addrIP = v.IP
					}
					if addrIP != nil && addrIP.Equal(ip) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
		if !found {
			return fmt.Errorf("invalid source IP")
		}
	}

	client, err := NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if *showBanner {
		fmt.Fprintln(out)
		centerPrintTo(out, asciiArt, *bannerWidth)
		fmt.Fprintln(out)
	}

	if err := client.GetConfig(ctx); err != nil {
		return fmt.Errorf("could not fetch config: %w", err)
	}

	if *list {
		if err := client.GetServers(ctx); err != nil {
			return fmt.Errorf("cannot fetch server list: %w", err)
		}
		if err := client.GetClosestServers(50); err != nil {
			return fmt.Errorf("cannot compute closest servers: %w", err)
		}
		for _, s := range client.servers {
			fmt.Fprintf(out, "%5d) %s (%s, %s) [%.2f km]\n", s.ID, s.Sponsor, s.Name, s.Country, s.Distance)
		}
		return nil
	}

	if *mini != "" {
		if err := client.SetMiniServer(ctx, *mini); err != nil {
			return fmt.Errorf("mini server error: %w", err)
		}
	} else {
		if err := client.GetServers(ctx); err != nil {
			return fmt.Errorf("cannot fetch servers: %w", err)
		}
	}

	if err := client.GetClosestServers(5); err != nil {
		return fmt.Errorf("cannot select closest servers: %w", err)
	}
	if _, err := client.GetBestServer(ctx); err != nil {
		return fmt.Errorf("cannot select best server: %w", err)
	}

	ping := client.Results().Ping
	pingColor := colorForPing(ping)
	fmt.Fprintf(out, "Hosted by %s (%s) [%.2f km]: %s\n",
		client.BestServer().Sponsor, client.BestServer().Name, client.BestServer().Distance,
		colorize(fmt.Sprintf("%.2f ms", ping), pingColor, !*noColor))

	results := client.Results()

	if !*noDownload {
		fmt.Fprintln(out)
		dctx, dcancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer dcancel()
		down, err := client.Download(dctx, *single, *bytesFlag, out)
		if err != nil {
			fmt.Fprintf(out, "Download error: %v\n", err)
		} else {
			results.Download = down
			val, unit := formatRate(down, *bytesFlag)
			mbps := results.Download / 1e6
			col := colorForSpeed(mbps)
			fmt.Fprintf(out, "%s: %s\n", colorize("Download", colorBlue, !*noColor), colorize(fmt.Sprintf("%.2f %s", val, unit), col, !*noColor))
		}
	} else {
		fmt.Fprintln(out, "Skipping download")
	}

	if !*noUpload {
		fmt.Fprintln(out)
		uctx, ucancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer ucancel()
		up, err := client.Upload(uctx, *single, *bytesFlag, out)
		if err != nil {
			fmt.Fprintf(out, "Upload error: %v\n", err)
		} else {
			results.Upload = up
			val, unit := formatRate(up, *bytesFlag)
			mbps := results.Upload / 1e6
			col := colorForSpeed(mbps)
			fmt.Fprintf(out, "%s: %s\n", colorize("Upload", colorBlue, !*noColor), colorize(fmt.Sprintf("%.2f %s", val, unit), col, !*noColor))
		}
	} else {
		fmt.Fprintln(out, "Skipping upload")
	}

	if *simple {
		dval, dunit := formatRate(results.Download, *bytesFlag)
		uval, uunit := formatRate(results.Upload, *bytesFlag)
		fmt.Fprintf(out, "Ping: %s\n", colorize(fmt.Sprintf("%.2f ms", results.Ping), colorForPing(results.Ping), !*noColor))
		fmt.Fprintf(out, "Download: %.2f %s\n", dval, dunit)
		fmt.Fprintf(out, "Upload: %.2f %s\n", uval, uunit)
	} else if *csvFlag {
		if *csvHeader {
			w := csv.NewWriter(out)
			_ = w.Write([]string{"Server ID", "Sponsor", "Server Name", "Distance", "Ping", "Download(bps)", "Upload(bps)", "IP"})
			w.Flush()
		}
		fmt.Fprintf(out, "%d,%s,%s,%.2f,%.2f,%.2f,%.2f,%s\n",
			client.BestServer().ID,
			client.BestServer().Sponsor,
			client.BestServer().Name,
			client.BestServer().Distance,
			results.Ping,
			results.Download,
			results.Upload,
			client.ClientIP())
	} else if *jsonFlag {
		enc := map[string]interface{}{
			"ping":     results.Ping,
			"download": results.Download,
			"upload":   results.Upload,
			"server":   client.BestServer(),
			"client":   client.ClientInfo(),
		}
		b, _ := json.MarshalIndent(enc, "", "  ")
		fmt.Fprintln(out, string(b))
	}

	if *share {
		fmt.Fprintln(out, "Share: not implemented in wrapper (TODO)")
	}
	return nil
}
