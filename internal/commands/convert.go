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
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func parseAmountAndToken(tok string) (float64, string, bool) {
	tok = strings.TrimSpace(tok)
	re1 := regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)([A-Za-z]{3,})$`) // 100usd
	re2 := regexp.MustCompile(`^([A-Za-z]{3,})([0-9]+(?:\.[0-9]+)?)$`) // usd100
	if m := re1.FindStringSubmatch(tok); len(m) == 3 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			return v, strings.ToUpper(m[2]), true
		}
	}
	if m := re2.FindStringSubmatch(tok); len(m) == 3 {
		if v, err := strconv.ParseFloat(m[2], 64); err == nil {
			return v, strings.ToUpper(m[1]), true
		}
	}
	return 0, "", false
}

func CmdConvert(args []string) string {
	if len(args) == 0 {
		return "convert: usage: convert <amount> <from> to <to>  e.g. `convert 100 usd to inr`"
	}

	joined := strings.ToLower(strings.Join(args, " "))
	toks := strings.Fields(joined)
	if len(toks) == 0 {
		return "convert: empty input"
	}

	amount := 1.0
	start := 0

	if v, cur, ok := parseAmountAndToken(toks[0]); ok {
		amount = v
		start = 1
		to := ""
		for i := start; i < len(toks); i++ {
			if toks[i] == "to" && i+1 < len(toks) {
				to = toks[i+1]
				break
			}
		}
		if to == "" && start < len(toks) {
			to = toks[start]
		}
		if to == "" {
			return "convert: could not determine target currency"
		}
		return convertWithMultiFallback(amount, cur, strings.ToUpper(strings.Trim(to, " ,.")))
	}

	if v, err := strconv.ParseFloat(strings.Trim(toks[0], ","), 64); err == nil {
		amount = v
		start = 1
	}

	toIdx := -1
	for i := start; i < len(toks); i++ {
		if toks[i] == "to" {
			toIdx = i
			break
		}
	}

	var from, to string
	if toIdx != -1 && toIdx > start {
		from = toks[start]
		if toIdx+1 < len(toks) {
			to = toks[toIdx+1]
		}
	} else {
		if start+1 < len(toks) {
			from = toks[start]
			to = toks[start+1]
		} else {
			return "convert: could not parse currencies. Usage: convert 100 usd to inr"
		}
	}

	if amt2, cur2, ok := parseAmountAndToken(from); ok {
		amount = amt2
		from = cur2
	}

	from = strings.ToUpper(strings.Trim(from, " ,."))
	to = strings.ToUpper(strings.Trim(to, " ,."))

	if len(from) < 3 || len(to) < 3 {
		return "convert: currency codes must be at least 3 letters (e.g. USD, INR)"
	}

	return convertWithMultiFallback(amount, from, to)
}

func convertWithMultiFallback(amount float64, from, to string) string {
	cacheKey := fmt.Sprintf("rate:%s:%s", from, to)

	if data, ok := readCache(cacheKey); ok {
		var cached struct {
			Rate float64 `json:"rate"`
		}
		if err := json.Unmarshal(data, &cached); err == nil && cached.Rate > 0 {
			result := amount * cached.Rate
			return fmt.Sprintf("%.6g %s = %.6g %s  (rate = %.6g) (cached)", amount, from, result, to, cached.Rate)
		}
	}

	type diag struct {
		Provider string
		Status   int
		Err      string
	}
	diags := []diag{}

	convURL := fmt.Sprintf("https://api.exchangerate.host/convert?from=%s&to=%s&amount=%v", from, to, amount)
	if b, sc, err := httpGetWithRetries(convURL); err == nil && sc == 200 {
		var res struct {
			Success bool `json:"success"`
			Query   struct {
				From   string  `json:"from"`
				To     string  `json:"to"`
				Amount float64 `json:"amount"`
			} `json:"query"`
			Info struct {
				Rate float64 `json:"rate"`
			} `json:"info"`
			Result float64 `json:"result"`
		}
		if err := json.Unmarshal(b, &res); err == nil && res.Success && res.Info.Rate > 0 {
			cached := struct {
				Rate float64 `json:"rate"`
			}{Rate: res.Info.Rate}
			if jb, err := json.Marshal(cached); err == nil {
				_ = writeCache(cacheKey, jb, 3600)
			}
			return fmt.Sprintf("%.6g %s = %.6g %s  (rate = %.6g)", res.Query.Amount, strings.ToUpper(res.Query.From), res.Result, strings.ToUpper(res.Query.To), res.Info.Rate)
		}
		diags = append(diags, diag{Provider: "exchangerate.convert", Status: sc, Err: "parse/falsy-success"})
	} else {
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		diags = append(diags, diag{Provider: "exchangerate.convert", Status: sc, Err: errStr})
	}

	latestURL := fmt.Sprintf("https://api.exchangerate.host/latest?base=%s&symbols=%s", from, to)
	if b2, sc2, err2 := httpGetWithRetries(latestURL); err2 == nil && sc2 == 200 {
		var latest struct {
			Success bool               `json:"success"`
			Rates   map[string]float64 `json:"rates"`
			Date    string             `json:"date"`
			Base    string             `json:"base"`
		}
		if err := json.Unmarshal(b2, &latest); err == nil && latest.Success {
			if rate, ok := latest.Rates[to]; ok && rate > 0 {
				cached := struct {
					Rate float64 `json:"rate"`
				}{Rate: rate}
				if jb, err := json.Marshal(cached); err == nil {
					_ = writeCache(cacheKey, jb, 3600)
				}
				result := amount * rate
				return fmt.Sprintf("%.6g %s = %.6g %s  (rate = %.6g)", amount, from, result, to, rate)
			}
		}
		diags = append(diags, diag{Provider: "exchangerate.latest", Status: sc2, Err: "parse/no-rate"})
	} else {
		errStr := ""
		if err2 != nil {
			errStr = err2.Error()
		}
		diags = append(diags, diag{Provider: "exchangerate.latest", Status: sc2, Err: errStr})
	}

	frankURL := fmt.Sprintf("https://api.frankfurter.app/latest?from=%s&to=%s", from, to)
	if b3, sc3, err3 := httpGetWithRetries(frankURL); err3 == nil && (sc3 == 200 || sc3 == 201) {
		var f struct {
			Base  string             `json:"base"`
			Rates map[string]float64 `json:"rates"`
			Date  string             `json:"date"`
		}
		if err := json.Unmarshal(b3, &f); err == nil {
			if rate, ok := f.Rates[to]; ok && rate > 0 {
				cached := struct {
					Rate float64 `json:"rate"`
				}{Rate: rate}
				if jb, err := json.Marshal(cached); err == nil {
					_ = writeCache(cacheKey, jb, 3600)
				}
				result := amount * rate
				return fmt.Sprintf("%.6g %s = %.6g %s  (rate = %.6g)", amount, from, result, to, rate)
			}
		}
		diags = append(diags, diag{Provider: "frankfurter", Status: sc3, Err: "parse/no-rate"})
	} else {
		errStr := ""
		if err3 != nil {
			errStr = err3.Error()
		}
		diags = append(diags, diag{Provider: "frankfurter", Status: sc3, Err: errStr})
	}

	openerURL := fmt.Sprintf("https://open.er-api.com/v6/latest/%s", from)
	if b4, sc4, err4 := httpGetWithRetries(openerURL); err4 == nil && sc4 == 200 {
		var oe struct {
			Result string             `json:"result"`
			Rates  map[string]float64 `json:"rates"`
		}
		if err := json.Unmarshal(b4, &oe); err == nil && (oe.Result == "success" || oe.Result == "") {
			if rate, ok := oe.Rates[to]; ok && rate > 0 {
				cached := struct {
					Rate float64 `json:"rate"`
				}{Rate: rate}
				if jb, err := json.Marshal(cached); err == nil {
					_ = writeCache(cacheKey, jb, 3600)
				}
				result := amount * rate
				return fmt.Sprintf("%.6g %s = %.6g %s  (rate = %.6g)", amount, from, result, to, rate)
			}
		}
		diags = append(diags, diag{Provider: "open.er-api", Status: sc4, Err: "parse/no-rate"})
	} else {
		errStr := ""
		if err4 != nil {
			errStr = err4.Error()
		}
		diags = append(diags, diag{Provider: "open.er-api", Status: sc4, Err: errStr})
	}

	if data, ok := readCache(cacheKey); ok {
		var cached struct {
			Rate float64 `json:"rate"`
		}
		if err := json.Unmarshal(data, &cached); err == nil && cached.Rate > 0 {
			result := amount * cached.Rate
			return fmt.Sprintf("%.6g %s = %.6g %s  (rate = %.6g) (cached)", amount, from, result, to, cached.Rate)
		}
	}

	parts := []string{"convert: remote reported failure. diagnostics:"}
	for _, d := range diags {
		parts = append(parts, fmt.Sprintf("%s(status=%d err=%s)", d.Provider, d.Status, truncate(d.Err, 120)))
	}
	parts = append(parts, "Try again or enable DEBUG to see http logs in data/debug.log")
	return strings.Join(parts, " ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
