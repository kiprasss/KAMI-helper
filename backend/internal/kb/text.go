package kb

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var stopwords = func() map[string]bool {
	words := strings.Fields(
		"и в во не что он на я с со как а то все она так его но да ты к у же вы за бы по только ее мне было вот от меня еще нет о из ему " +
			"теперь когда даже ну вдруг ли если уже или ни быть был него до вас нибудь опять уж вам ведь там потом себя ничего ей может они " +
			"тут где есть надо ней для мы тебя их чем была сам чтоб без будто чего раз тоже себе под будет ж тогда кто этот того потому " +
			"этого какой совсем ним здесь этом один почти мой тем чтобы нее сейчас были куда зачем всех никогда можно при наконец два об " +
			"другой хоть после над больше тот через эти нас про всего них какая много разве три эту моя впрочем хорошо свою этой перед " +
			"иногда лучше чуть том нельзя такой им более всегда конечно всю между это как-то скажи подскажи пожалуйста привет здравствуйте " +
			"бот можно ли что-то какие какое каков какова сколько можна мона низя нельзи ваще вообще че чё щас " +
			"такое такой значит означает обозначает расшифровывается расшифровка это",
	)

	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[w] = true
	}

	return set
}()

var endings = func() []string {
	list := []string{
		"ления", "лению", "лением", "лении", "ление", "лять", "ляет", "лить",
		"иями", "ями", "ами", "ией", "иям", "ием", "иях", "ость", "ости", "остью", "ение", "ения", "ению", "ением", "ении",
		"ание", "ания", "анию", "анием", "ании", "ться", "тся", "ешь", "ете", "ите", "ишь", "ает", "яет", "ует", "ют", "ут",
		"ать", "ять", "ить", "еть", "ого", "его", "ому", "ему", "ыми", "ими", "ый", "ий", "ой", "ая", "яя", "ое", "ее", "ые",
		"ие", "ых", "их", "ым", "им", "ую", "юю", "ов", "ев", "ей", "ам", "ям", "ах", "ях", "ом", "ем", "а", "я", "о", "е",
		"ы", "и", "у", "ю", "ь", "й",
	}

	sort.SliceStable(list, func(i, j int) bool {
		return utf8.RuneCountInString(list[i]) > utf8.RuneCountInString(list[j])
	})

	return list
}()

var (
	cyrillicRe   = regexp.MustCompile(`[а-я]`)
	vowelEndRe   = regexp.MustCompile(`[аяоеыиуюьй]$`)
	pointTokenRe = regexp.MustCompile(`^\d{1,2}(?:\.\d{1,2}){1,2}$`)
	tokenRe      = regexp.MustCompile(`\d{1,2}(?:\.\d{1,2}){1,2}|[\p{L}\p{N}]+`)
	yoReplacer   = strings.NewReplacer("ё", "е")
)

const prefixMark = "~"

func Stem(word string) string {
	n := utf8.RuneCountInString(word)
	if !cyrillicRe.MatchString(word) || n <= 3 {
		return word
	}

	if n == 4 {
		if vowelEndRe.MatchString(word) {
			r := []rune(word)
			return string(r[:len(r)-1])
		}
		return word
	}

	for _, end := range endings {
		if strings.HasSuffix(word, end) && n-utf8.RuneCountInString(end) >= 3 {
			return strings.TrimSuffix(word, end)
		}
	}

	return word
}

func normalizeWord(word string) string {
	return yoReplacer.Replace(strings.ToLower(word))
}

func tokenTerms(word string) []string {
	t := normalizeWord(word)

	if pointTokenRe.MatchString(t) {
		return []string{t}
	}

	if utf8.RuneCountInString(t) < 2 || stopwords[t] {
		return nil
	}

	s := Stem(t)
	terms := []string{s}

	if r := []rune(s); len(r) > 6 && cyrillicRe.MatchString(s) {
		terms = append(terms, prefixMark+string(r[:6]))
	}

	return terms
}

func Tokenize(text string) []string {
	var out []string

	for _, raw := range tokenRe.FindAllString(yoReplacer.Replace(strings.ToLower(text)), -1) {
		out = append(out, tokenTerms(raw)...)
	}

	return out
}

type slangRule struct {
	all  []string
	none []string
	add  string
}

type compiledSlang struct {
	all  []*regexp.Regexp
	none []*regexp.Regexp
	add  string
}

var slang = func() []compiledSlang {
	out := make([]compiledSlang, 0, len(slangRules))

	for _, r := range slangRules {
		c := compiledSlang{add: r.add}
		for _, p := range r.all {
			c.all = append(c.all, regexp.MustCompile(p))
		}
		for _, p := range r.none {
			c.none = append(c.none, regexp.MustCompile(p))
		}
		out = append(out, c)
	}

	return out
}()

func ExpandSlang(text string) string {
	var extra []string

	for _, rule := range slang {
		matched := true

		for _, re := range rule.all {
			if !re.MatchString(text) {
				matched = false
				break
			}
		}

		for _, re := range rule.none {
			if matched && re.MatchString(text) {
				matched = false
			}
		}

		if matched {
			extra = append(extra, rule.add)
		}
	}

	if len(extra) == 0 {
		return text
	}

	return text + " " + strings.Join(extra, " ")
}
