package forum

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

type ExtractedRule struct {
	Point      string   `json:"point"`
	Text       string   `json:"text"`
	Punishment string   `json:"punishment,omitempty"`
	RepeatWarn bool     `json:"repeat_warn,omitempty"`
	Images     []string `json:"images,omitempty"`
}

var (
	PointRe   = regexp.MustCompile(`(?i)^(?:п\.?\s*|пункт\s+)?(\d{1,2}(?:\.\d{1,2}){1,2})\.?(?:\s+|$)(.*)$`)
	SectionRe = regexp.MustCompile(`^\d{1,2}\.\s+\S`)

	punishWords   = `(?:JAIL|WARN|BAN|MUTE|KICK|DEMORGAN|ДЕМОРГАН|ДЖАЙЛ|ВАРН|БАН|МУТ|КИК|ЧС|Ч\.С\.|Изъятие|Увольнение|Понижение|Предупреждение|Устное предупреждение|Выговор|Снятие|Обнуление|Лишение|Блокировка)`
	punishRe      = regexp.MustCompile(`(?i)(^|[\s|:—–(-])` + punishWords)
	repeatWarnRe  = regexp.MustCompile(`(?i)WARN[^|]*повтор|повтор[^|]*WARN|ВАРН[^|]*повтор`)
	punishLabelRe = regexp.MustCompile(`(?im)(?:Наказание|Наказания|Санкция)\s*[:—–-]\s*(.+)$`)
	newlinesRe    = regexp.MustCompile(`\n+`)
	doubleSpaceRe = regexp.MustCompile(`\s{2,}`)
)

func IsRepeatWarn(punishment string) bool {
	return repeatWarnRe.MatchString(punishment)
}

func SplitPunishment(block string) (string, string) {
	parts := strings.Split(block, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) > 1 {
		for idx := 1; idx < len(parts); idx++ {
			if !punishRe.MatchString(parts[idx]) {
				continue
			}

			var rest []string
			for _, p := range parts[idx:] {
				if p != "" {
					rest = append(rest, p)
				}
			}

			text := strings.TrimSpace(strings.Join(parts[:idx], " | "))
			punishment := strings.TrimSpace(strings.TrimSuffix(strings.Join(rest, " | "), "."))

			return text, punishment
		}
	}

	if loc := punishLabelRe.FindStringSubmatchIndex(block); loc != nil {
		text := strings.TrimSpace(block[:loc[0]])
		punishment := strings.TrimSpace(strings.TrimSuffix(block[loc[2]:loc[3]], "."))

		return text, punishment
	}

	return strings.TrimSpace(block), ""
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}

	return string([]rune(s)[:n])
}

func ExtractRules(text string, maxContinuationLines int) []ExtractedRule {
	type current struct {
		point  string
		lines  []string
		extra  int
		closed bool
	}

	var rules []ExtractedRule
	var cur *current

	hasPunishment := func(line string) bool {
		_, p := SplitPunishment(line)
		return p != ""
	}

	flush := func() {
		if cur == nil {
			return
		}

		block, images := ExtractImages(strings.TrimSpace(strings.Join(cur.lines, "\n")))
		block = doubleSpaceRe.ReplaceAllString(newlinesRe.ReplaceAllString(block, " "), " ")
		body, punishment := SplitPunishment(block)

		if utf8.RuneCountInString(body) >= 8 {
			rules = append(rules, ExtractedRule{
				Point:      cur.point,
				Text:       truncateRunes(body, 1500),
				Punishment: truncateRunes(punishment, 500),
				RepeatWarn: punishment != "" && IsRepeatWarn(punishment),
				Images:     images,
			})
		}

		cur = nil
	}

	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)

		if m := PointRe.FindStringSubmatch(line); m != nil {
			flush()
			cur = &current{point: m[1], lines: []string{m[2]}, closed: hasPunishment(m[2])}
			continue
		}

		if cur == nil {
			continue
		}

		if SectionRe.MatchString(line) {
			flush()
			continue
		}

		if IsImageLine(line) && cur.extra < maxContinuationLines {
			cur.lines = append(cur.lines, line)
			continue
		}

		if line == "" || cur.closed || cur.extra >= maxContinuationLines {
			continue
		}

		cur.lines = append(cur.lines, line)
		cur.extra++
		cur.closed = hasPunishment(line)
	}

	flush()

	return rules
}
