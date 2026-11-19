package forum

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	roleSectionRe   = regexp.MustCompile(`(?i)^(Правила|Положение|Обязанности)\s+(\p{L}+\s+)?для\s+[^|]+$`)
	imageLineRe     = regexp.MustCompile(`(?i)\.(png|jpe?g|gif|webp)$`)
	noteLineRe      = regexp.MustCompile(`(?i)^(Примечание|Уточнение|Исключение|Пояснение|Спойлер|Наказание|Было|Стало)`)
	trailingPunctRe = regexp.MustCompile(`[.;,]$`)
	chapterRe       = regexp.MustCompile(`(?i)^(Раздел|Глава)\s+\d+`)
	wasLineRe       = regexp.MustCompile(`^\s*Было:?\s*$`)
	becameLineRe    = regexp.MustCompile(`^\s*Стало:?\s*$`)
)

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

func splitSentences(line string) []string {
	var out []string
	runes := []rune(line)
	start := 0

	for i := 0; i < len(runes); i++ {
		if !strings.ContainsRune(".!?…", runes[i]) {
			continue
		}

		j := i + 1
		for j < len(runes) && unicode.IsSpace(runes[j]) {
			j++
		}

		if j > i+1 {
			out = append(out, string(runes[start:i+1]))
			start = j
			i = j - 1
		}
	}

	if start < len(runes) {
		out = append(out, string(runes[start:]))
	}

	return out
}

func ChunkText(text string, maxLen, overlap int) []string {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return nil
	}

	if runeLen(clean) <= maxLen {
		return []string{clean}
	}

	var atoms []string
	for _, line := range strings.Split(clean, "\n") {
		if runeLen(line) <= maxLen {
			atoms = append(atoms, line)
			continue
		}

		buf := ""
		for _, sentence := range splitSentences(line) {
			if runeLen(strings.TrimSpace(buf+" "+sentence)) > maxLen && buf != "" {
				atoms = append(atoms, strings.TrimSpace(buf))
				buf = ""
			}

			if runeLen(sentence) > maxLen {
				r := []rune(sentence)
				for i := 0; i < len(r); i += maxLen {
					end := min(i+maxLen, len(r))
					atoms = append(atoms, string(r[i:end]))
				}
			} else {
				buf += " " + sentence
			}
		}

		if strings.TrimSpace(buf) != "" {
			atoms = append(atoms, strings.TrimSpace(buf))
		}
	}

	var chunks []string
	cur := ""

	for _, atom := range atoms {
		candidate := atom
		if cur != "" {
			candidate = cur + "\n" + atom
		}

		if runeLen(candidate) > maxLen && cur != "" {
			chunks = append(chunks, strings.TrimSpace(cur))

			carry := ""
			if overlap > 0 {
				r := []rune(cur)
				tail := string(r[max(0, len(r)-overlap):])
				if cut := strings.Index(tail, " "); cut >= 0 {
					carry = tail[cut+1:]
				}
			}

			if carry != "" && runeLen(carry)+runeLen(atom)+1 <= maxLen {
				cur = carry + "\n" + atom
			} else {
				cur = atom
			}
		} else {
			cur = candidate
		}
	}

	if strings.TrimSpace(cur) != "" {
		chunks = append(chunks, strings.TrimSpace(cur))
	}

	out := chunks[:0]
	for _, c := range chunks {
		if strings.TrimSpace(c) != "" {
			out = append(out, c)
		}
	}

	return out
}

func headingCandidate(line string) bool {
	return line != "" &&
		!IsImageLine(line) &&
		runeLen(line) <= 80 &&
		!strings.Contains(line, "|") &&
		!imageLineRe.MatchString(line) &&
		!noteLineRe.MatchString(line) &&
		!trailingPunctRe.MatchString(line)
}

func DropOldWording(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))

	for i := 0; i < len(lines); i++ {
		if wasLineRe.MatchString(lines[i]) {
			end := -1
			for j := i + 1; j < len(lines); j++ {
				if becameLineRe.MatchString(lines[j]) {
					end = j
					break
				}
			}

			if end >= 0 {
				i = end
				continue
			}
		}

		out = append(out, lines[i])
	}

	return strings.Join(out, "\n")
}

func ChunkRulesText(text string, maxLen, packLen int) []string {
	clean := strings.TrimSpace(DropOldWording(text))
	if clean == "" {
		return nil
	}

	lines := strings.Split(clean, "\n")

	hasPoints := false
	for _, l := range lines {
		if PointRe.MatchString(strings.TrimSpace(l)) {
			hasPoints = true
			break
		}
	}

	if !hasPoints {
		return ChunkText(clean, 900, 120)
	}

	type unit struct {
		text    string
		point   string
		section string
	}

	var units []unit
	section := ""
	lastMajor := ""
	var heads []string
	var cur *unit

	push := func() {
		if cur != nil && strings.TrimSpace(cur.text) != "" {
			u := *cur
			u.text = strings.TrimSpace(u.text)
			units = append(units, u)
		}
		cur = nil
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)

		if m := PointRe.FindStringSubmatch(line); m != nil {
			parts := strings.Split(m[1], ".")
			major, minor := parts[0], parts[1]

			if minor == "1" && major != lastMajor && len(heads) > 0 {
				section = strings.Join(heads[max(0, len(heads)-2):], " ")
			}

			lastMajor = major
			heads = nil
			push()
			cur = &unit{text: line, point: m[1], section: section}
			continue
		}

		if (SectionRe.MatchString(line) || chapterRe.MatchString(line) || roleSectionRe.MatchString(line)) && runeLen(line) <= 120 {
			push()
			section = line
			heads = nil
			cur = &unit{text: line}
			continue
		}

		if cur != nil {
			switch {
			case headingCandidate(line):
				heads = append(heads, line)
			case line != "":
				heads = nil
			}
			cur.text += "\n" + line
			continue
		}

		cur = &unit{text: line, section: section}
	}

	push()

	var pieces []unit
	for _, u := range units {
		if runeLen(u.text) <= maxLen {
			pieces = append(pieces, u)
			continue
		}

		parts := ChunkText(u.text, maxLen-150, 0)
		punishment := ""
		if u.point != "" && len(parts) > 0 {
			_, punishment = SplitPunishment(strings.SplitN(parts[0], "\n", 2)[0])
		}

		for i, p := range parts {
			label := ""
			if i > 0 && u.point != "" {
				if punishment != "" {
					label = "Пункт " + u.point + " (продолжение; наказание по пункту: " + punishment + "):\n"
				} else {
					label = "Пункт " + u.point + " (продолжение):\n"
				}
			}
			pieces = append(pieces, unit{text: label + p, point: u.point, section: u.section})
		}
	}

	var chunks []string
	buf := ""

	for _, p := range pieces {
		candidate := p.text
		if buf != "" {
			candidate = buf + "\n\n" + p.text
		}

		if buf != "" && runeLen(candidate) > packLen {
			chunks = append(chunks, buf)
			if p.section != "" && p.point != "" {
				buf = p.section + "\n\n" + p.text
			} else {
				buf = p.text
			}
		} else {
			buf = candidate
		}
	}

	if strings.TrimSpace(buf) != "" {
		chunks = append(chunks, buf)
	}

	return chunks
}
