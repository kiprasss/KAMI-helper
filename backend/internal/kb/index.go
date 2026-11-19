package kb

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

const (
	DocTypeProject     = "project_rules"
	DocTypeLegislation = "legislation"

	KindRule = "rule"
	KindText = "text"
)

type Entry struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	DocURL      string `json:"-"`
	PostURL     string `json:"url"`
	Title       string `json:"document_title"`
	Breadcrumbs string `json:"breadcrumbs"`
	DocType     string `json:"document_type"`
	IsRules     bool   `json:"-"`
	Point       string `json:"point"`
	Text        string `json:"text"`
	Punishment  string `json:"punishment"`
}

func (e *Entry) indexText() string {
	if e.Kind != KindRule {
		return e.Text
	}

	s := "Пункт " + e.Point + ": " + e.Text
	if e.Punishment != "" {
		s += "\nНаказание: " + e.Punishment
	}

	return s
}

type indexedDoc struct {
	entry      *Entry
	tf         map[string]int
	length     int
	titleTerms map[string]bool
}

type Hit struct {
	Entry *Entry
	Score float64
}

const (
	k1                 = 1.4
	b                  = 0.75
	titleWeight        = 0.8
	unnamedEventFactor = 0.6
	archiveFactor      = 0.4
	relativeCutoff     = 0.3
	minScore           = 1.0
	perDocLimit        = 6
)

var (
	generalRulesRe = regexp.MustCompile(`(?i)^Общие правила проекта`)
	eventRulesRe   = regexp.MustCompile(`(?i)^(Правила\s+)?(мероприяти|событи|ГМП)`)
	eventSectionRe = regexp.MustCompile(`(?i)›\s*Общие правила мероприятий\s*$`)
	narrowRulesRe  = regexp.MustCompile(`(?i)допрос`)
	generalTitleRe = regexp.MustCompile(`(?i)^Общие правила`)
	archiveRe      = regexp.MustCompile(`(?i)›\s*Архив([^\p{L}]|$)`)
	conductedRe    = regexp.MustCompile(`(?i)\|\s*Проведен`)
	personalFormRe = regexp.MustCompile(`(?i)Реальный\s+возраст\s*:`)
	leadingPointRe = regexp.MustCompile(`^(?:Пункт\s+)?(\d{1,2}(?:\.\d{1,2}){1,2})\b`)
)

var eventGeneric = func() map[string]bool {
	set := map[string]bool{}
	for _, t := range Tokenize("правила правил мероприятие мероприятия мероприятий событие события ограбление ограбления ограблений и на для в " +
		"проведения нападения защиты находящихся территории государственных структур криминальные организации перевозки перекрытия") {
		set[t] = true
	}
	return set
}()

func isEventRules(e *Entry) bool {
	if eventRulesRe.MatchString(e.Title) || narrowRulesRe.MatchString(e.Title) {
		return true
	}

	return eventSectionRe.MatchString(e.Breadcrumbs) && !generalTitleRe.MatchString(e.Title)
}

func isArchived(e *Entry) bool {
	return !e.IsRules && (archiveRe.MatchString(e.Breadcrumbs) || conductedRe.MatchString(e.Title))
}

func IsPersonalForm(text string) bool {
	return personalFormRe.MatchString(text)
}

func eventNamed(titleTerms, query map[string]bool) bool {
	for t := range titleTerms {
		if !eventGeneric[t] && query[t] {
			return true
		}
	}

	return false
}

type Index struct {
	docs   []indexedDoc
	df     map[string]int
	avgLen float64
	byID   map[string]*Entry
}

func BuildIndex(entries []*Entry) *Index {
	idx := &Index{df: map[string]int{}, byID: map[string]*Entry{}}
	total := 0

	for _, e := range entries {
		titleTokens := Tokenize(e.Title)
		tokens := append(append([]string{}, titleTokens...), Tokenize(e.indexText())...)

		tf := map[string]int{}
		for _, t := range tokens {
			tf[t]++
		}
		for t := range tf {
			idx.df[t]++
		}

		titleTerms := map[string]bool{}
		for _, t := range titleTokens {
			titleTerms[t] = true
		}

		idx.docs = append(idx.docs, indexedDoc{entry: e, tf: tf, length: len(tokens), titleTerms: titleTerms})
		idx.byID[e.ID] = e
		total += len(tokens)
	}

	idx.avgLen = 1
	if len(idx.docs) > 0 {
		idx.avgLen = float64(total) / float64(len(idx.docs))
	}

	return idx
}

func (idx *Index) Size() int {
	return len(idx.docs)
}

func (idx *Index) Get(id string) *Entry {
	return idx.byID[id]
}

func (idx *Index) Search(query string, allowed map[string]bool) []Hit {
	expanded := ExpandSlang(query)

	seen := map[string]bool{}
	var terms []string
	for _, t := range Tokenize(expanded) {
		if !seen[t] {
			seen[t] = true
			terms = append(terms, t)
		}
	}

	if len(terms) == 0 {
		return nil
	}

	n := float64(len(idx.docs))
	idfOf := func(t string) float64 {
		df := float64(idx.df[t])
		return math.Log(1 + (n-df+0.5)/(df+0.5))
	}

	idfTotal := 0.0
	for _, t := range terms {
		idfTotal += idfOf(t)
	}
	if idfTotal == 0 {
		idfTotal = 1
	}

	var hits []Hit

	for _, d := range idx.docs {
		if allowed != nil && !allowed[d.entry.DocURL] {
			continue
		}

		score := 0.0
		matched := 0
		titleIdf := 0.0

		for _, t := range terms {
			f := d.tf[t]
			if f == 0 {
				continue
			}

			matched++
			idf := idfOf(t)
			if d.titleTerms[t] {
				titleIdf += idf
			}

			boost := 1.0
			switch {
			case pointTokenRe.MatchString(t):
				boost = 3
			case strings.HasPrefix(t, prefixMark):
				boost = 0.5
			}

			ff := float64(f)
			score += boost * idf * (ff * (k1 + 1)) / (ff + k1*(1-b+b*float64(d.length)/idx.avgLen))
		}

		if matched == 0 {
			continue
		}

		score *= 1 + float64(matched)/float64(len(terms))

		unnamedEvent := isEventRules(d.entry) && !eventNamed(d.titleTerms, seen)
		if !unnamedEvent {
			score *= 1 + titleWeight*(titleIdf/idfTotal)
		}

		if d.entry.IsRules {
			score *= 1.25
		}

		if generalRulesRe.MatchString(d.entry.Title) {
			score *= 1.3
		}

		switch {
		case unnamedEvent:
			score *= unnamedEventFactor
		case isArchived(d.entry):
			score *= archiveFactor
		}

		hits = append(hits, Hit{Entry: d.entry, Score: score})
	}

	sort.SliceStable(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})

	return capPerDoc(hits)
}

func leadingPoint(text string) string {
	lines := strings.SplitN(text, "\n", 5)
	if len(lines) > 4 {
		lines = lines[:4]
	}

	for _, line := range lines {
		if m := leadingPointRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			return m[1]
		}
	}

	return ""
}

func capPerDoc(hits []Hit) []Hit {
	if len(hits) == 0 {
		return nil
	}

	cutoff := math.Max(minScore, hits[0].Score*relativeCutoff)
	perDoc := map[string]int{}
	seenPoints := map[string]bool{}
	seenText := map[string]bool{}
	var out []Hit

	for _, h := range hits {
		if h.Score < cutoff {
			continue
		}

		text := h.Entry.indexText()

		if point := leadingPoint(text); point != "" {
			key := h.Entry.DocURL + "#" + point
			if seenPoints[key] {
				continue
			}
			seenPoints[key] = true
		}

		norm := normalizedText(text)
		if seenText[norm] {
			continue
		}
		seenText[norm] = true

		if perDoc[h.Entry.DocURL] >= perDocLimit {
			continue
		}
		perDoc[h.Entry.DocURL]++

		out = append(out, h)
	}

	return out
}

var whitespaceRe = regexp.MustCompile(`\s+`)

func normalizedText(text string) string {
	s := strings.TrimSpace(whitespaceRe.ReplaceAllString(strings.ToLower(text), " "))
	if r := []rune(s); len(r) > 300 {
		s = string(r[:300])
	}

	return s
}
