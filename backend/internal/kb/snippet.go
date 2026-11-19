package kb

import (
	"strings"
	"unicode/utf8"
)

const (
	snippetBefore = 60
	snippetLength = 260
)

func QueryTerms(query string) map[string]bool {
	set := map[string]bool{}
	for _, t := range Tokenize(ExpandSlang(query)) {
		set[t] = true
	}

	return set
}

func tokenMatches(token string, terms map[string]bool) bool {
	for _, t := range tokenTerms(token) {
		if terms[t] {
			return true
		}
	}

	return false
}

func Snippet(text string, terms map[string]bool) string {
	flat := strings.Join(strings.Fields(text), " ")

	locs := tokenRe.FindAllStringIndex(flat, -1)
	var marks [][2]int
	for _, loc := range locs {
		if tokenMatches(flat[loc[0]:loc[1]], terms) {
			marks = append(marks, [2]int{loc[0], loc[1]})
		}
	}

	start := 0
	if len(marks) > 0 {
		firstRune := utf8.RuneCountInString(flat[:marks[0][0]])
		if firstRune > snippetBefore {
			start = byteOffset(flat, firstRune-snippetBefore)
			if sp := strings.IndexByte(flat[start:], ' '); sp >= 0 && sp < 30 {
				start += sp + 1
			}
		}
	}

	end := len(flat)
	if utf8.RuneCountInString(flat[start:]) > snippetLength {
		end = start + byteOffset(flat[start:], snippetLength)
		if sp := strings.LastIndexByte(flat[start:end], ' '); sp > 0 {
			end = start + sp
		}
	}

	var sb strings.Builder
	if start > 0 {
		sb.WriteString("… ")
	}

	pos := start
	for _, m := range marks {
		if m[0] < start || m[1] > end {
			continue
		}
		sb.WriteString(flat[pos:m[0]])
		sb.WriteString("<mark>")
		sb.WriteString(flat[m[0]:m[1]])
		sb.WriteString("</mark>")
		pos = m[1]
	}
	sb.WriteString(flat[pos:end])

	if end < len(flat) {
		sb.WriteString(" …")
	}

	return sb.String()
}

func byteOffset(s string, runes int) int {
	i := 0
	for pos := range s {
		if i == runes {
			return pos
		}
		i++
	}

	return len(s)
}
