package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type ServerSources struct {
	Code    string
	Name    string
	Sources []string
}

type Config struct {
	Port                string
	CORSOrigins         []string
	ForumBaseURL        string
	UserAgent           string
	ScrapeDelay         time.Duration
	RefreshInterval     time.Duration
	CacheMaxAge         time.Duration
	CacheFile           string
	MaxThreadsPerSource int
	ForumPages          int
	MaxPagesPerThread   int
	BrowserClearance    bool
	BrowserPath         string
	GeneralSources      []string
	Servers             []ServerSources
}

var defaultCORSOrigins = []string{
	"http://localhost:1420",
	"http://tauri.localhost",
	"https://tauri.localhost",
	"tauri://localhost",
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	base := strings.TrimRight(getEnv("FORUM_BASE_URL", "https://forum.nova-rp.com"), "/")

	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		CORSOrigins:         splitList(os.Getenv("CORS_ORIGINS"), ","),
		ForumBaseURL:        base,
		UserAgent:           getEnv("SCRAPER_USER_AGENT", "KAMI-Helper/1.0 (NOVA RP rules search)"),
		ScrapeDelay:         time.Duration(getInt("SCRAPE_DELAY_MS", 1000)) * time.Millisecond,
		RefreshInterval:     time.Duration(getInt("REFRESH_MINUTES", 120)) * time.Minute,
		CacheMaxAge:         time.Duration(getInt("SCRAPE_CACHE_MINUTES", 30)) * time.Minute,
		CacheFile:           getEnv("CACHE_FILE", "data/forum-cache.json"),
		MaxThreadsPerSource: max(1, getInt("SCRAPE_MAX_THREADS_PER_SOURCE", 60)),
		ForumPages:          max(1, getInt("SCRAPE_FORUM_PAGES", 2)),
		MaxPagesPerThread:   max(1, getInt("SCRAPE_MAX_PAGES_PER_THREAD", 3)),
		BrowserClearance:    getBool("FORUM_BROWSER_CLEARANCE", true),
		BrowserPath:         os.Getenv("FORUM_BROWSER_PATH"),
	}

	if len(cfg.CORSOrigins) == 0 {
		cfg.CORSOrigins = defaultCORSOrigins
	}

	cfg.GeneralSources = splitList(os.Getenv("FORUM_GENERAL_SOURCES"), ",")
	if len(cfg.GeneralSources) == 0 {
		cfg.GeneralSources = []string{
			base + "/forums/893/",
			base + "/threads/403338/",
		}
	}

	servers, err := parseServers(os.Getenv("FORUM_SERVERS"))
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		servers = []ServerSources{
			{Code: "NEW_ERA", Name: "NEW ERA", Sources: []string{base + "/forums/1320/"}},
			{Code: "MIRAGE", Name: "MIRAGE", Sources: []string{base + "/forums/929/"}},
			{Code: "HORIZON", Name: "HORIZON", Sources: []string{base + "/forums/1478/"}},
		}
	}
	cfg.Servers = servers

	return cfg, nil
}

func parseServers(value string) ([]ServerSources, error) {
	var out []ServerSources

	for _, item := range splitList(value, ";") {
		parts := strings.SplitN(item, "|", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("FORUM_SERVERS: ожидается «КОД|Название|ссылка,ссылка», получено %q", item)
		}

		out = append(out, ServerSources{
			Code:    strings.TrimSpace(parts[0]),
			Name:    strings.TrimSpace(parts[1]),
			Sources: splitList(parts[2], ","),
		})
	}

	return out, nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func getInt(key string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}

	return n
}

func getBool(key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on", "да":
		return true
	case "0", "false", "no", "off", "нет":
		return false
	default:
		return fallback
	}
}

func splitList(value, sep string) []string {
	var items []string

	for _, item := range strings.Split(value, sep) {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}

	return items
}
