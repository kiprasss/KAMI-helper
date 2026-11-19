package forum

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

var challengeTitleRe = regexp.MustCompile(`(?i)Проверка браузера|Checking your browser`)

func FindBrowser(explicit string) string {
	if explicit != "" {
		return explicit
	}

	var candidates []string

	switch runtime.GOOS {
	case "windows":
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LocalAppData"} {
			root := os.Getenv(env)
			if root == "" {
				continue
			}
			candidates = append(candidates,
				filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(root, "Chromium", "Application", "chrome.exe"),
				filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"),
			)
		}
	case "darwin":
		candidates = append(candidates,
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		)
	default:
		home, _ := os.UserHomeDir()
		if dirs, err := filepath.Glob(filepath.Join(home, ".cache", "ms-playwright", "chromium_headless_shell-*")); err == nil {
			sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
			for _, d := range dirs {
				candidates = append(candidates, filepath.Join(d, "chrome-linux", "headless_shell"))
			}
		}
		candidates = append(candidates,
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/snap/bin/chromium",
		)
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}

	return ""
}

func ObtainClearance(ctx context.Context, pageURL, userAgent, browserPath string) (string, error) {
	execPath := FindBrowser(browserPath)

	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	opts = append(opts,
		chromedp.UserAgent(userAgent),
		chromedp.Flag("lang", "ru-RU"),
	)
	if execPath != "" {
		opts = append(opts, chromedp.ExecPath(execPath))
	}
	if runtime.GOOS == "linux" {
		opts = append(opts, chromedp.NoSandbox)
	}

	limitCtx, cancelLimit := context.WithTimeout(ctx, 90*time.Second)
	defer cancelLimit()

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(limitCtx, opts...)
	defer cancelAlloc()

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	if err := chromedp.Do(browserCtx, chromedp.Navigate(pageURL)); err != nil {
		return "", fmt.Errorf("не удалось открыть страницу в браузере: %w", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		title, err := chromedp.Run(browserCtx, chromedp.Title())
		if err != nil {
			title = "Проверка браузера"
		}

		if !challengeTitleRe.MatchString(title) {
			break
		}

		if time.Now().After(deadline) {
			return "", errors.New("проверка браузера не закончилась за 30 секунд")
		}

		select {
		case <-browserCtx.Done():
			return "", browserCtx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	result, err := chromedp.Run(browserCtx, func(ctx context.Context, t *chromedp.Target) (*network.GetCookiesResult, error) {
		res, err := cdp.Call(ctx, t, network.GetCookies, network.GetCookiesParams{URLs: []string{pageURL}})
		return &res, err
	})
	if err != nil {
		return "", fmt.Errorf("не удалось получить cookie: %w", err)
	}
	cookies := result.Cookies

	parts := make([]string, 0, len(cookies))
	names := make([]string, 0, len(cookies))
	for _, c := range cookies {
		parts = append(parts, c.Name+"="+c.Value)
		names = append(names, c.Name)
	}

	if len(parts) == 0 {
		return "", errors.New("браузер не получил cookie")
	}

	log.Printf("Проверка браузера пройдена, cookie: %s", strings.Join(names, ", "))

	return strings.Join(parts, "; "), nil
}
