package forum

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

type ThreadField struct {
	Label string
	Value string
}

type ParsedPost struct {
	ID     string
	Author string
	URL    string
	Text   string
}

type ParsedThread struct {
	Title       string
	Prefix      string
	Breadcrumbs []string
	Fields      []ThreadField
	Posts       []ParsedPost
	TotalPages  int
}

type ThreadListItem struct {
	URL        string
	Title      string
	LastPostTs int64
	Sticky     bool
}

type Subforum struct {
	URL   string
	Title string
}

type ParsedForum struct {
	Title      string
	Threads    []ThreadListItem
	Subforums  []Subforum
	TotalPages int
}

const blockTags = "p, div, li, ul, ol, h1, h2, h3, h4, h5, h6, tr, table, blockquote, pre, dl, dt, dd"

var (
	invisibleReplacer = strings.NewReplacer(string(rune(0x200b)), "", string(rune(0x200c)), "", string(rune(0x200d)), "", string(rune(0xfeff)), "", string(rune(0xa0)), " ")
	spacesRe          = regexp.MustCompile(`[ \t]+`)
	manyNewlinesRe    = regexp.MustCompile(`\n{3,}`)
	postIDRe          = regexp.MustCompile(`post-(\d+)`)
	smileyAltRe       = regexp.MustCompile(`^[:;]`)
	digitsRe          = regexp.MustCompile(`^\d+$`)
)

func cleanText(text string) string {
	text = invisibleReplacer.Replace(text)

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(spacesRe.ReplaceAllString(line, " "))
	}

	text = strings.Join(lines, "\n")
	text = manyNewlinesRe.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

func htmlToText(sel *goquery.Selection) string {
	node := sel.Clone()

	node.Find("script, style, noscript, .bbCodeBlock-expandLink, .bbCodeBlock-title, .js-unfurl-favicon").Remove()

	node.Find("img").Each(func(_ int, img *goquery.Selection) {
		alt, _ := img.Attr("alt")
		replacement := " "
		if alt != "" && !smileyAltRe.MatchString(alt) {
			replacement = " " + alt + " "
		}
		img.ReplaceWithNodes(&html.Node{Type: html.TextNode, Data: replacement})
	})

	node.Find("br").Each(func(_ int, br *goquery.Selection) {
		br.ReplaceWithNodes(&html.Node{Type: html.TextNode, Data: "\n"})
	})

	node.Find(blockTags).Each(func(_ int, block *goquery.Selection) {
		block.PrependNodes(&html.Node{Type: html.TextNode, Data: "\n"})
		block.AppendNodes(&html.Node{Type: html.TextNode, Data: "\n"})
	})

	return cleanText(node.Text())
}

func pageCount(doc *goquery.Document) int {
	maxPage := 1

	doc.Find(".pageNav-page a, .pageNav-page").Each(func(_ int, s *goquery.Selection) {
		n, err := strconv.Atoi(strings.TrimSpace(s.Text()))
		if err == nil && n > maxPage {
			maxPage = n
		}
	})

	return maxPage
}

func titleOf(doc *goquery.Document) (string, string) {
	h1 := doc.Find("h1.p-title-value").First().Clone()
	prefix := strings.TrimSpace(h1.Find(".label").First().Text())
	h1.Find(".label, .label-append").Remove()

	title := cleanText(h1.Text())
	if title == "" {
		title = cleanText(doc.Find("title").First().Text())
	}

	return title, prefix
}

func ParseThread(body, threadURL string) (ParsedThread, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return ParsedThread{}, err
	}

	title, prefix := titleOf(doc)

	var breadcrumbs []string
	doc.Find(".p-breadcrumbs li a").Each(func(_ int, a *goquery.Selection) {
		t := cleanText(a.Text())
		if t == "" {
			return
		}
		for _, existing := range breadcrumbs {
			if existing == t {
				return
			}
		}
		breadcrumbs = append(breadcrumbs, t)
	})

	var fields []ThreadField
	seen := map[string]bool{}
	doc.Find("dl.pairs--customField").Each(func(_ int, dl *goquery.Selection) {
		label := cleanText(dl.Find("dt").First().Text())
		value := htmlToText(dl.Find("dd").First())
		key := label + "\x00" + value
		if label != "" && value != "" && !seen[key] {
			seen[key] = true
			fields = append(fields, ThreadField{Label: label, Value: value})
		}
	})

	var posts []ParsedPost
	doc.Find("article.message--post").Each(func(_ int, art *goquery.Selection) {
		content, ok := art.Attr("data-content")
		if !ok {
			content, _ = art.Attr("id")
		}

		id := strconv.Itoa(len(posts) + 1)
		if m := postIDRe.FindStringSubmatch(content); m != nil {
			id = m[1]
		}

		bodySel := art.Find(".message-body .bbWrapper").First()
		if bodySel.Length() == 0 {
			return
		}

		author, _ := art.Attr("data-author")

		posts = append(posts, ParsedPost{
			ID:     id,
			Author: author,
			URL:    threadURL + "#post-" + id,
			Text:   htmlToText(bodySel),
		})
	})

	return ParsedThread{
		Title:       title,
		Prefix:      prefix,
		Breadcrumbs: breadcrumbs,
		Fields:      fields,
		Posts:       posts,
		TotalPages:  pageCount(doc),
	}, nil
}

func ParseForum(body, baseURL string) (ParsedForum, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return ParsedForum{}, err
	}

	title, _ := titleOf(doc)

	var threads []ThreadListItem
	seen := map[string]bool{}

	doc.Find(".structItem--thread").Each(func(_ int, item *goquery.Selection) {
		link := item.Find(".structItem-title a[data-tp-primary]").First()
		if link.Length() == 0 {
			link = item.Find(`.structItem-title a[href*="/threads/"]`).Last()
		}

		href, _ := link.Attr("href")
		norm, ok := NormalizeURL(href, baseURL)
		if !ok || norm.Kind != KindThread || seen[norm.URL] {
			return
		}
		seen[norm.URL] = true

		var ts int64
		timeSel := item.Find("time.structItem-latestDate").First()
		tsAttr, ok := timeSel.Attr("data-timestamp")
		if !ok {
			tsAttr, _ = timeSel.Attr("data-time")
		}
		if digitsRe.MatchString(tsAttr) {
			n, _ := strconv.ParseInt(tsAttr, 10, 64)
			ts = n * 1000
		} else if dt, ok := timeSel.Attr("datetime"); ok {
			if parsed, err := time.Parse(time.RFC3339, dt); err == nil {
				ts = parsed.UnixMilli()
			}
		}

		sticky := item.Closest(".structItemContainer-group--sticky").Length() > 0 ||
			item.Find(".structItem-status--sticky").Length() > 0

		threads = append(threads, ThreadListItem{
			URL:        norm.URL,
			Title:      cleanText(link.Text()),
			LastPostTs: ts,
			Sticky:     sticky,
		})
	})

	var subforums []Subforum
	doc.Find(".node-title a").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		norm, ok := NormalizeURL(href, baseURL)
		if !ok || norm.Kind != KindForum {
			return
		}
		for _, s := range subforums {
			if s.URL == norm.URL {
				return
			}
		}
		subforums = append(subforums, Subforum{URL: norm.URL, Title: cleanText(a.Text())})
	})

	return ParsedForum{
		Title:      title,
		Threads:    threads,
		Subforums:  subforums,
		TotalPages: pageCount(doc),
	}, nil
}
