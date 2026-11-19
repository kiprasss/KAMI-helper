package forum

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrBlocked = errors.New("форум включил защиту от ботов (проверка браузера), и пройти её не удалось")

type HTTPError struct {
	Status int
	URL    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d для %s", e.Status, e.URL)
}

var challengeBodyRe = regexp.MustCompile(`(?i)Checking your browser|Проверка браузера|Automated protection|cf-challenge|ddos-guard`)

func IsBotChallenge(status int, body string) bool {
	if status == http.StatusUnauthorized && strings.TrimSpace(body) == `{"ok":false}` {
		return true
	}

	return len(body) < 20000 && challengeBodyRe.MatchString(body)
}

type FetcherOptions struct {
	UserAgent        string
	Delay            time.Duration
	Timeout          time.Duration
	Retries          int
	BrowserClearance bool
	BrowserPath      string
	ForumBaseURL     string
}

const clearanceRetry = 5 * time.Minute

type Fetcher struct {
	opts          FetcherOptions
	client        *http.Client
	mu            sync.Mutex
	lastRequestAt time.Time
	cookieMu      sync.RWMutex
	cookie        string
	lastClearance time.Time
}

func NewFetcher(opts FetcherOptions) *Fetcher {
	if opts.Timeout == 0 {
		opts.Timeout = 20 * time.Second
	}

	return &Fetcher{
		opts:   opts,
		client: &http.Client{Timeout: opts.Timeout},
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}

	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (f *Fetcher) getCookie() string {
	f.cookieMu.RLock()
	defer f.cookieMu.RUnlock()

	return f.cookie
}

func (f *Fetcher) setCookie(cookie string) {
	f.cookieMu.Lock()
	f.cookie = cookie
	f.cookieMu.Unlock()
}

func (f *Fetcher) Get(ctx context.Context, url string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	clearanceTried := false

	for attempt := 0; ; attempt++ {
		if err := sleepCtx(ctx, time.Until(f.lastRequestAt.Add(f.opts.Delay))); err != nil {
			return "", err
		}
		f.lastRequestAt = time.Now()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", f.opts.UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")
		if cookie := f.getCookie(); cookie != "" {
			req.Header.Set("Cookie", cookie)
		}

		res, err := f.client.Do(req)
		if err != nil {
			if attempt < f.opts.Retries {
				if err := sleepCtx(ctx, f.opts.Delay*time.Duration(attempt+2)); err != nil {
					return "", err
				}
				continue
			}
			return "", err
		}

		bodyBytes, readErr := io.ReadAll(io.LimitReader(res.Body, 20<<20))
		res.Body.Close()

		if (res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500) && attempt < f.opts.Retries {
			wait := f.opts.Delay * time.Duration(attempt+2) * 2
			if sec, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && sec > 0 {
				wait = time.Duration(sec) * time.Second
			}
			if err := sleepCtx(ctx, wait); err != nil {
				return "", err
			}
			continue
		}

		if readErr != nil {
			return "", readErr
		}

		body := string(bodyBytes)

		if IsBotChallenge(res.StatusCode, body) {
			f.setCookie("")

			if f.opts.BrowserClearance && !clearanceTried && time.Since(f.lastClearance) >= clearanceRetry {
				clearanceTried = true
				f.lastClearance = time.Now()

				cookie, err := ObtainClearance(ctx, url, f.opts.UserAgent, f.opts.BrowserPath)
				if err != nil {
					log.Printf("Не удалось пройти проверку браузера: %v", err)
				} else {
					f.setCookie(cookie)
					f.lastClearance = time.Time{}
					attempt--
					continue
				}
			}

			return "", ErrBlocked
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return "", &HTTPError{Status: res.StatusCode, URL: url}
		}

		return body, nil
	}
}

var ErrNotImage = errors.New("по ссылке не изображение")

const maxImageSize = 15 << 20

func (f *Fetcher) FetchImage(ctx context.Context, imageURL string) ([]byte, string, error) {
	data, contentType, err := f.fetchImageOnce(ctx, imageURL)
	if errors.Is(err, ErrNotImage) && sameHost(imageURL, f.opts.ForumBaseURL) {
		if _, getErr := f.Get(ctx, f.opts.ForumBaseURL+"/"); getErr == nil {
			return f.fetchImageOnce(ctx, imageURL)
		}
	}

	return data, contentType, err
}

func (f *Fetcher) fetchImageOnce(ctx context.Context, imageURL string) ([]byte, string, error) {
	cookie := f.getCookie()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", f.opts.UserAgent)
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,image/*")
	if cookie != "" && sameHost(imageURL, f.opts.ForumBaseURL) {
		req.Header.Set("Cookie", cookie)
	}

	res, err := f.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, "", &HTTPError{Status: res.StatusCode, URL: imageURL}
	}

	contentType := res.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", ErrNotImage
	}

	data, err := io.ReadAll(io.LimitReader(res.Body, maxImageSize+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxImageSize {
		return nil, "", errors.New("изображение слишком большое")
	}

	return data, contentType, nil
}

func sameHost(a, b string) bool {
	ua, errA := url.Parse(a)
	ub, errB := url.Parse(b)

	return errA == nil && errB == nil && strings.EqualFold(ua.Hostname(), ub.Hostname())
}
