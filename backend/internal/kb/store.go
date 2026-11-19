package kb

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"backend/internal/forum"
)

type Server struct {
	Code    string   `json:"code"`
	Name    string   `json:"name"`
	Sources []string `json:"-"`
}

type Options struct {
	BaseURL             string
	GeneralSources      []string
	Servers             []Server
	CacheFile           string
	CacheMaxAge         time.Duration
	MaxThreadsPerSource int
	ForumPages          int
	MaxPagesPerThread   int
}

type StoredRule struct {
	PostURL string `json:"post_url"`
	forum.ExtractedRule
}

type Chunk struct {
	PostURL string   `json:"post_url"`
	Text    string   `json:"text"`
	Images  []string `json:"images,omitempty"`
}

type Document struct {
	URL         string       `json:"url"`
	Title       string       `json:"title"`
	Breadcrumbs string       `json:"breadcrumbs"`
	IsRules     bool         `json:"is_rules"`
	LastPostTs  int64        `json:"last_post_ts"`
	FetchedAt   int64        `json:"fetched_at"`
	Rules       []StoredRule `json:"rules"`
	Chunks      []Chunk      `json:"chunks"`
}

type SourceState struct {
	Title     string   `json:"title"`
	Docs      []string `json:"docs"`
	FetchedAt int64    `json:"fetched_at"`
	Error     string   `json:"error,omitempty"`
}

const cacheVersion = 2

type cacheFile struct {
	Version     int                     `json:"version"`
	LastRefresh int64                   `json:"last_refresh"`
	Sources     map[string]*SourceState `json:"sources"`
	Documents   map[string]*Document    `json:"documents"`
}

type Status struct {
	Ready       bool   `json:"ready"`
	Refreshing  bool   `json:"refreshing"`
	Blocked     bool   `json:"blocked"`
	LastRefresh int64  `json:"last_refresh"`
	Documents   int    `json:"documents"`
	Rules       int    `json:"rules"`
	Entries     int    `json:"entries"`
	Progress    string `json:"progress"`
	LastError   string `json:"last_error"`
}

type Store struct {
	opts    Options
	fetcher *forum.Fetcher

	mu          sync.RWMutex
	sources     map[string]*SourceState
	documents   map[string]*Document
	lastRefresh int64
	index       *Index
	scopes      map[string]map[string]bool
	docTypes    map[string]string
	images      map[string]bool
	status      Status

	refreshMu sync.Mutex
}

func NewStore(opts Options, fetcher *forum.Fetcher) *Store {
	s := &Store{
		opts:      opts,
		fetcher:   fetcher,
		sources:   map[string]*SourceState{},
		documents: map[string]*Document{},
		index:     BuildIndex(nil),
	}

	return s
}

func (s *Store) Servers() []Server {
	return s.opts.Servers
}

func (s *Store) HasServer(code string) bool {
	for _, srv := range s.opts.Servers {
		if srv.Code == code {
			return true
		}
	}

	return false
}

func (s *Store) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.status
}

func (s *Store) LoadCache() error {
	data, err := os.ReadFile(s.opts.CacheFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var c cacheFile
	if err := json.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("кэш повреждён: %w", err)
	}

	if c.Version != cacheVersion {
		log.Printf("Кэш старого формата (версия %d) — правила будут скачаны заново", c.Version)
		return nil
	}

	s.mu.Lock()
	if c.Sources != nil {
		s.sources = c.Sources
	}
	if c.Documents != nil {
		s.documents = c.Documents
	}
	s.lastRefresh = c.LastRefresh
	s.mu.Unlock()

	s.rebuild()

	return nil
}

func (s *Store) saveCache() error {
	s.mu.RLock()
	c := cacheFile{
		Version:     cacheVersion,
		LastRefresh: s.lastRefresh,
		Sources:     s.sources,
		Documents:   s.documents,
	}
	data, err := json.Marshal(c)
	s.mu.RUnlock()

	if err != nil {
		return err
	}

	if dir := filepath.Dir(s.opts.CacheFile); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	tmp := s.opts.CacheFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, s.opts.CacheFile)
}

func (s *Store) LastRefresh() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastRefresh == 0 {
		return time.Time{}
	}

	return time.UnixMilli(s.lastRefresh)
}

func entryID(parts ...string) string {
	h := sha1.Sum([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h[:])[:16]
}

func (s *Store) rebuild() {
	s.mu.Lock()
	defer s.mu.Unlock()

	general := map[string]bool{}
	for _, src := range s.opts.GeneralSources {
		if st := s.sources[src]; st != nil {
			for _, d := range st.Docs {
				general[d] = true
			}
		}
	}

	docTypes := map[string]string{}
	for d := range general {
		docTypes[d] = DocTypeProject
	}

	scopes := map[string]map[string]bool{}
	for _, srv := range s.opts.Servers {
		allowed := map[string]bool{}
		for d := range general {
			allowed[d] = true
		}
		for _, src := range srv.Sources {
			if st := s.sources[src]; st != nil {
				for _, d := range st.Docs {
					allowed[d] = true
					if _, ok := docTypes[d]; !ok {
						docTypes[d] = DocTypeLegislation
					}
				}
			}
		}
		scopes[srv.Code] = allowed
	}

	urls := make([]string, 0, len(s.documents))
	for u := range s.documents {
		if _, ok := docTypes[u]; ok {
			urls = append(urls, u)
		}
	}
	sort.Strings(urls)

	var entries []*Entry
	rulesCount := 0

	for _, u := range urls {
		d := s.documents[u]
		base := Entry{
			DocURL:      d.URL,
			Title:       d.Title,
			Breadcrumbs: d.Breadcrumbs,
			DocType:     docTypes[u],
			IsRules:     d.IsRules,
		}

		for i, c := range d.Chunks {
			if IsPersonalForm(c.Text) {
				continue
			}
			e := base
			e.ID = entryID(u, KindText, fmt.Sprint(i), c.Text)
			e.Kind = KindText
			e.PostURL = c.PostURL
			e.Text = c.Text
			e.Images = c.Images
			entries = append(entries, &e)
		}

		if d.IsRules {
			for i, r := range d.Rules {
				e := base
				e.ID = entryID(u, KindRule, fmt.Sprint(i), r.Point, r.Text)
				e.Kind = KindRule
				e.PostURL = r.PostURL
				e.Point = r.Point
				e.Text = r.Text
				e.Punishment = r.Punishment
				e.Images = r.Images
				entries = append(entries, &e)
				rulesCount++
			}
		}
	}

	images := map[string]bool{}
	for _, e := range entries {
		for _, img := range e.Images {
			images[img] = true
		}
	}

	s.index = BuildIndex(entries)
	s.images = images
	s.scopes = scopes
	s.docTypes = docTypes
	s.status.Ready = len(entries) > 0
	s.status.Documents = len(urls)
	s.status.Rules = rulesCount
	s.status.Entries = len(entries)
	s.status.LastRefresh = s.lastRefresh
}

func (s *Store) Search(query, server string) []Hit {
	s.mu.RLock()
	idx := s.index
	allowed := s.scopes[server]
	s.mu.RUnlock()

	if allowed == nil {
		allowed = map[string]bool{}
	}

	return idx.Search(query, allowed)
}

func (s *Store) KnownImage(url string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.images[url]
}

func (s *Store) Get(id string) *Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.index.Get(id)
}

func (s *Store) setProgress(format string, args ...any) {
	s.mu.Lock()
	s.status.Progress = fmt.Sprintf(format, args...)
	s.mu.Unlock()
}

func (s *Store) allSources() []string {
	seen := map[string]bool{}
	var out []string

	add := func(u string) {
		if !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}

	for _, u := range s.opts.GeneralSources {
		add(u)
	}
	for _, srv := range s.opts.Servers {
		for _, u := range srv.Sources {
			add(u)
		}
	}

	return out
}

func (s *Store) Refresh(ctx context.Context) error {
	if !s.refreshMu.TryLock() {
		return nil
	}
	defer s.refreshMu.Unlock()

	started := time.Now()

	s.mu.Lock()
	s.status.Refreshing = true
	s.status.Blocked = false
	s.status.LastError = ""
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.status.Refreshing = false
		s.status.Progress = ""
		s.mu.Unlock()
	}()

	sources := s.allSources()
	log.Printf("Обновление базы правил: источников %d", len(sources))

	var errs []string
	fetched, cached := 0, 0
	blocked := false

	for i, src := range sources {
		s.setProgress("Источник %d из %d", i+1, len(sources))

		norm, ok := forum.NormalizeURL(src, s.opts.BaseURL)
		if !ok {
			errs = append(errs, "неверная ссылка: "+src)
			continue
		}

		var (
			title string
			docs  []string
			err   error
		)

		if norm.Kind == forum.KindThread {
			var d *Document
			d, err = s.indexThread(ctx, norm.URL, 0, &fetched, &cached)
			if d != nil {
				title = d.Title
				docs = []string{d.URL}
			}
		} else {
			title, docs, err = s.indexForum(ctx, norm.URL, &fetched, &cached, &errs)
		}

		if err != nil {
			if errors.Is(err, forum.ErrBlocked) {
				blocked = true
				errs = append(errs, err.Error())
				log.Printf("Форум закрыт защитой от ботов: %v", err)
				break
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}

			errs = append(errs, fmt.Sprintf("%s: %v", src, err))
			log.Printf("Источник %s не обработан: %v", src, err)

			s.mu.Lock()
			if st := s.sources[src]; st != nil {
				st.Error = err.Error()
			}
			s.mu.Unlock()
			continue
		}

		s.mu.Lock()
		s.sources[src] = &SourceState{Title: title, Docs: docs, FetchedAt: time.Now().UnixMilli()}
		s.mu.Unlock()

		s.rebuild()
	}

	s.mu.Lock()
	referenced := map[string]bool{}
	for _, st := range s.sources {
		for _, d := range st.Docs {
			referenced[d] = true
		}
	}
	for u := range s.documents {
		if !referenced[u] {
			delete(s.documents, u)
		}
	}
	if !blocked {
		s.lastRefresh = time.Now().UnixMilli()
	}
	s.status.Blocked = blocked
	if len(errs) > 0 {
		s.status.LastError = errs[0]
	}
	s.mu.Unlock()

	s.rebuild()

	if err := s.saveCache(); err != nil {
		log.Printf("Не удалось сохранить кэш: %v", err)
	}

	log.Printf("Обновление завершено за %s: скачано тем %d, из кэша %d, ошибок %d",
		time.Since(started).Round(time.Second), fetched, cached, len(errs))

	return nil
}

var (
	rulesTitleRe     = regexp.MustCompile(`(?i)правил`)
	complaintTitleRe = regexp.MustCompile(`(?i)жалоб`)
	rulesPrefixRe    = regexp.MustCompile(`(?i)^\s*Правила`)
	sensitiveLabelRe = regexp.MustCompile(`(?i)(ник|nick|никнейм|(^|[^\p{L}])id([^\p{L}]|$)|айди|static|статик|discord|дискорд|(^|[^\p{L}])vk([^\p{L}]|$)|telegram|телеграм|steam|(^|[^\p{L}])ip([^\p{L}]|$))`)
)

func isComplaintThread(title string, fields []forum.ThreadField, breadcrumbs []string) bool {
	if rulesPrefixRe.MatchString(title) {
		return false
	}

	for _, f := range fields {
		if sensitiveLabelRe.MatchString(f.Label) {
			return true
		}
	}

	for _, b := range breadcrumbs {
		if complaintTitleRe.MatchString(b) {
			return true
		}
	}

	return false
}

var errComplaint = errors.New("тема является жалобой и не индексируется")

func (s *Store) indexThread(ctx context.Context, url string, listingTs int64, fetched, cached *int) (*Document, error) {
	s.mu.RLock()
	prev := s.documents[url]
	s.mu.RUnlock()

	if prev != nil && time.Since(time.UnixMilli(prev.FetchedAt)) < s.opts.CacheMaxAge {
		*cached++
		return prev, nil
	}

	body, err := s.fetcher.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	first, err := forum.ParseThread(body, url)
	if err != nil {
		return nil, err
	}

	if isComplaintThread(first.Title, first.Fields, first.Breadcrumbs) {
		return nil, errComplaint
	}

	posts := append([]forum.ParsedPost{}, first.Posts...)
	pages := min(first.TotalPages, s.opts.MaxPagesPerThread)

	for p := 2; p <= pages; p++ {
		pageBody, err := s.fetcher.Get(ctx, forum.PageURL(url, p))
		if err != nil {
			if errors.Is(err, forum.ErrBlocked) || ctx.Err() != nil {
				return nil, err
			}
			log.Printf("%s: %v", forum.PageURL(url, p), err)
			continue
		}

		page, err := forum.ParseThread(pageBody, url)
		if err == nil {
			posts = append(posts, page.Posts...)
		}
	}

	*fetched++

	isRules := rulesTitleRe.MatchString(first.Title)
	for _, b := range first.Breadcrumbs {
		if rulesTitleRe.MatchString(b) {
			isRules = true
		}
	}

	var header []string
	if first.Prefix != "" {
		header = append(header, "Статус/метка темы: "+first.Prefix)
	}
	for _, f := range first.Fields {
		header = append(header, f.Label+": "+f.Value)
	}

	doc := &Document{
		URL:         url,
		Title:       first.Title,
		Breadcrumbs: strings.Join(first.Breadcrumbs, " › "),
		IsRules:     isRules,
		LastPostTs:  listingTs,
		FetchedAt:   time.Now().UnixMilli(),
	}

	seenPoints := map[string]bool{}

	for i, post := range posts {
		text := post.Text
		if i == 0 && len(header) > 0 {
			text = strings.Join(header, "\n") + "\n\n" + text
		}

		var chunks []string
		if isRules {
			chunks = forum.ChunkRulesText(text, 1200, 600)
		} else {
			chunks = forum.ChunkText(text, 900, 120)
		}
		for _, c := range chunks {
			clean, images := forum.ExtractImages(c)
			if strings.TrimSpace(clean) == "" && len(images) > 0 && len(doc.Chunks) > 0 {
				last := &doc.Chunks[len(doc.Chunks)-1]
				last.Images = append(last.Images, images...)
				continue
			}
			doc.Chunks = append(doc.Chunks, Chunk{PostURL: post.URL, Text: clean, Images: images})
		}

		source := post.Text
		if isRules {
			source = forum.DropOldWording(post.Text)
		}

		for _, r := range forum.ExtractRules(source, 6) {
			if seenPoints[r.Point] {
				continue
			}
			seenPoints[r.Point] = true
			doc.Rules = append(doc.Rules, StoredRule{PostURL: post.URL, ExtractedRule: r})
		}
	}

	s.mu.Lock()
	s.documents[url] = doc
	s.mu.Unlock()

	return doc, nil
}

func (s *Store) indexForum(ctx context.Context, url string, fetched, cached *int, errs *[]string) (string, []string, error) {
	var items []forum.ThreadListItem
	title := ""

	var collect func(forumURL string, depth int) error
	collect = func(forumURL string, depth int) error {
		body, err := s.fetcher.Get(ctx, forumURL)
		if err != nil {
			return err
		}

		firstPage, err := forum.ParseForum(body, s.opts.BaseURL)
		if err != nil {
			return err
		}

		if depth == 0 {
			title = firstPage.Title
		}
		items = append(items, firstPage.Threads...)

		pages := min(firstPage.TotalPages, s.opts.ForumPages)
		for p := 2; p <= pages; p++ {
			pageBody, err := s.fetcher.Get(ctx, forum.PageURL(forumURL, p))
			if err != nil {
				if errors.Is(err, forum.ErrBlocked) || ctx.Err() != nil {
					return err
				}
				*errs = append(*errs, fmt.Sprintf("%s: %v", forum.PageURL(forumURL, p), err))
				continue
			}
			if page, err := forum.ParseForum(pageBody, s.opts.BaseURL); err == nil {
				items = append(items, page.Threads...)
			}
		}

		if depth == 0 {
			for _, sub := range firstPage.Subforums {
				if err := collect(sub.URL, depth+1); err != nil {
					if errors.Is(err, forum.ErrBlocked) || ctx.Err() != nil {
						return err
					}
					*errs = append(*errs, fmt.Sprintf("%s: %v", sub.URL, err))
				}
			}
		}

		return nil
	}

	if err := collect(url, 0); err != nil {
		return "", nil, err
	}

	unique := map[string]forum.ThreadListItem{}
	var ordered []forum.ThreadListItem
	for _, it := range items {
		if _, ok := unique[it.URL]; !ok {
			unique[it.URL] = it
			ordered = append(ordered, it)
		}
	}

	boolInt := func(v bool) int {
		if v {
			return 1
		}
		return 0
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if boolInt(a.Sticky) != boolInt(b.Sticky) {
			return a.Sticky
		}
		ar, br := rulesTitleRe.MatchString(a.Title), rulesTitleRe.MatchString(b.Title)
		if ar != br {
			return ar
		}
		return a.LastPostTs > b.LastPostTs
	})

	if len(ordered) > s.opts.MaxThreadsPerSource {
		ordered = ordered[:s.opts.MaxThreadsPerSource]
	}

	var urls []string
	for i, t := range ordered {
		s.setProgress("%s: тема %d из %d", title, i+1, len(ordered))

		d, err := s.indexThread(ctx, t.URL, t.LastPostTs, fetched, cached)
		if err != nil {
			if errors.Is(err, forum.ErrBlocked) || ctx.Err() != nil {
				return "", nil, err
			}
			if !errors.Is(err, errComplaint) {
				*errs = append(*errs, fmt.Sprintf("%s: %v", t.URL, err))
			}
			continue
		}
		urls = append(urls, d.URL)
	}

	return title, urls, nil
}
