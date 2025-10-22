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
	"net/url"
	"strings"
)

func CmdWeather(args []string) string {
	loc := "your location"
	if len(args) > 0 {
		loc = strings.Join(args, " ")
	}
	loc = strings.TrimSpace(loc)
	if loc == "" {
		loc = "your location"
	}

	cacheKey := "weather:" + strings.ToLower(strings.TrimSpace(loc))
	if data, ok := readCache(cacheKey); ok {
		return string(data) + " (cached)"
	}

	geourl := "https://geocoding-api.open-meteo.com/v1/search?name=" + url.QueryEscape(loc) + "&count=1&language=en"
	body, sc, err := httpGetWithRetries(geourl)
	if err != nil || sc != 200 {
		wttrURL := "https://wttr.in/" + url.PathEscape(loc) + "?format=3"
		if b2, sc2, err2 := httpGetWithRetries(wttrURL); err2 == nil && sc2 == 200 {
			s := strings.TrimSpace(string(b2))
			if s != "" {
				_ = writeCache(cacheKey, []byte(s), 300) // cache 5 minutes
				return s
			}
		}
		if data, ok := readCache(cacheKey); ok {
			return string(data) + " (cached)"
		}
		if err != nil {
			return "weather: geocode failed: " + err.Error()
		}
		return fmt.Sprintf("weather: geocode returned status %d", sc)
	}

	var geoRes struct {
		Results []struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Name      string  `json:"name"`
			Country   string  `json:"country"`
			Timezone  string  `json:"timezone"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &geoRes); err != nil {
		wttrURL := "https://wttr.in/" + url.PathEscape(loc) + "?format=3"
		if b2, sc2, err2 := httpGetWithRetries(wttrURL); err2 == nil && sc2 == 200 {
			s := strings.TrimSpace(string(b2))
			if s != "" {
				_ = writeCache(cacheKey, []byte(s), 300)
				return s
			}
		}
		if data, ok := readCache(cacheKey); ok {
			return string(data) + " (cached)"
		}
		return "weather: geocode parse error: " + err.Error()
	}
	if len(geoRes.Results) == 0 {
		wttrURL := "https://wttr.in/" + url.PathEscape(loc) + "?format=3"
		if b2, sc2, err2 := httpGetWithRetries(wttrURL); err2 == nil && sc2 == 200 {
			s := strings.TrimSpace(string(b2))
			if s != "" {
				_ = writeCache(cacheKey, []byte(s), 300)
				return s
			}
		}
		if data, ok := readCache(cacheKey); ok {
			return string(data) + " (cached)"
		}
		return "weather: location not found"
	}
	g := geoRes.Results[0]

	forecastURL := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current_weather=true&timezone=auto",
		g.Latitude, g.Longitude)
	b2, sc2, err2 := httpGetWithRetries(forecastURL)
	if err2 != nil || sc2 != 200 {
		wttrURL := "https://wttr.in/" + url.PathEscape(loc) + "?format=3"
		if b3, sc3, err3 := httpGetWithRetries(wttrURL); err3 == nil && sc3 == 200 {
			s := strings.TrimSpace(string(b3))
			if s != "" {
				_ = writeCache(cacheKey, []byte(s), 300)
				return s
			}
		}
		if data, ok := readCache(cacheKey); ok {
			return string(data) + " (cached)"
		}
		if err2 != nil {
			return "weather: forecast fetch failed: " + err2.Error()
		}
		return fmt.Sprintf("weather: forecast returned status %d", sc2)
	}

	var fRes struct {
		CurrentWeather struct {
			Temperature   float64 `json:"temperature"`
			Windspeed     float64 `json:"windspeed"`
			Winddirection float64 `json:"winddirection"`
			Weathercode   int     `json:"weathercode"`
			Time          string  `json:"time"`
		} `json:"current_weather"`
	}
	if err := json.Unmarshal(b2, &fRes); err != nil {
		wttrURL := "https://wttr.in/" + url.PathEscape(loc) + "?format=3"
		if b3, sc3, err3 := httpGetWithRetries(wttrURL); err3 == nil && sc3 == 200 {
			s := strings.TrimSpace(string(b3))
			if s != "" {
				_ = writeCache(cacheKey, []byte(s), 300)
				return s
			}
		}
		if data, ok := readCache(cacheKey); ok {
			return string(data) + " (cached)"
		}
		return "weather: forecast parse error: " + err.Error()
	}

	c := fRes.CurrentWeather
	if c.Time == "" {
		if data, ok := readCache(cacheKey); ok {
			return string(data) + " (cached)"
		}
		return "weather: no current weather available"
	}

	wDesc := map[int]string{
		0:  "Clear",
		1:  "Mainly clear",
		2:  "Partly cloudy",
		3:  "Overcast",
		45: "Fog",
		48: "Depositing rime fog",
		51: "Light drizzle",
		53: "Moderate drizzle",
		55: "Dense drizzle",
		56: "Light freezing drizzle",
		57: "Dense freezing drizzle",
		61: "Slight rain",
		63: "Moderate rain",
		65: "Heavy rain",
		66: "Light freezing rain",
		67: "Heavy freezing rain",
		71: "Slight snow",
		73: "Moderate snow",
		75: "Heavy snow",
		80: "Rain showers",
		81: "Moderate showers",
		82: "Violent showers",
		95: "Thunderstorm",
		96: "Thunderstorm with slight hail",
		99: "Thunderstorm with heavy hail",
	}
	desc := "Unknown"
	if v, ok := wDesc[c.Weathercode]; ok {
		desc = v
	}

	name := g.Name
	if g.Country != "" {
		name = name + ", " + g.Country
	}
	out := fmt.Sprintf("%s — %s — %.1f°C — wind %.1f km/h", name, desc, c.Temperature, c.Windspeed)

	_ = writeCache(cacheKey, []byte(out), 300)
	return out
}
