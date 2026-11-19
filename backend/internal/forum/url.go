package forum

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type SourceKind string

const (
	KindThread SourceKind = "thread"
	KindForum  SourceKind = "forum"
)

type ForumURL struct {
	URL  string
	Kind SourceKind
	ID   int
}

var forumPathRe = regexp.MustCompile(`/(threads|forums)/(?:[^/]*?\.)?(\d+)(?:/|$)`)

func NormalizeURL(input, baseURL string) (ForumURL, bool) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return ForumURL{}, false
	}

	parsed, err := base.Parse(strings.TrimSpace(input))
	if err != nil {
		return ForumURL{}, false
	}

	host := strings.TrimPrefix(parsed.Hostname(), "www.")
	baseHost := strings.TrimPrefix(base.Hostname(), "www.")
	if host != baseHost && host != "forum.grand-rp.su" {
		return ForumURL{}, false
	}

	m := forumPathRe.FindStringSubmatch(parsed.Path)
	if m == nil {
		return ForumURL{}, false
	}

	id, err := strconv.Atoi(m[2])
	if err != nil {
		return ForumURL{}, false
	}

	kind := KindForum
	if m[1] == "threads" {
		kind = KindThread
	}

	origin := base.Scheme + "://" + base.Host

	return ForumURL{
		URL:  fmt.Sprintf("%s/%s/%d/", origin, m[1], id),
		Kind: kind,
		ID:   id,
	}, true
}

func PageURL(canonicalURL string, page int) string {
	if page > 1 {
		return fmt.Sprintf("%spage-%d", canonicalURL, page)
	}

	return canonicalURL
}
