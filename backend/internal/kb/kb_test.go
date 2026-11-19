package kb

import (
	"strings"
	"testing"
)

func TestTokenize(t *testing.T) {
	if Stem("наказания") != Stem("наказание") {
		t.Fatal("основы «наказания» и «наказание» разные")
	}
	if got := strings.Join(Tokenize("Что будет за пункт 5.4?"), ","); got != "пункт,5.4" {
		t.Fatalf("%q", got)
	}
	for _, w := range []string{"провоцироват", "провоцирование"} {
		found := false
		for _, tok := range Tokenize(w) {
			if tok == "~провоц" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: нет ~провоц в %v", w, Tokenize(w))
		}
	}
	if len(Tokenize("что такое")) != 0 {
		t.Fatal("стоп-слова не отброшены")
	}
}

func TestExpandSlang(t *testing.T) {
	cases := map[string]string{
		"дм в зз":               "DM DeathMatch",
		"рейд на семью":         "рейдов государственных структур на криминальные организации",
		"оскорбил мать":         "оскорбление родных и близких",
		"сколько держать в кпз": "превысила 60 минут",
	}
	for in, want := range cases {
		if out := ExpandSlang(in); !strings.Contains(out, want) {
			t.Fatalf("%q -> %q, нет %q", in, out, want)
		}
	}
	if out := ExpandSlang("рейд на черный рынок"); strings.Contains(out, "рейдов государственных структур") {
		t.Fatalf("лишнее расширение: %q", out)
	}
	if out := ExpandSlang("банкомат"); strings.Contains(out, "BAN") {
		t.Fatalf("«банкомат» принят за бан: %q", out)
	}
}

func TestSearchAndSnippet(t *testing.T) {
	entries := []*Entry{
		{ID: "a", Kind: KindRule, DocURL: "d1", Title: "Общие правила проекта", IsRules: true, Point: "4.1", Text: "DeathMatch (DM) — убийство или попытка убийства без весомой на то IC причины.", Punishment: "JAIL до 120 минут"},
		{ID: "b", Kind: KindRule, DocURL: "d1", Title: "Общие правила проекта", IsRules: true, Point: "3.2", Text: "Запрещено оскорбление личности игрока в ООС разговоре."},
		{ID: "c", Kind: KindText, DocURL: "d2", Title: "Чужой сервер", Text: "DM разрешён"},
	}
	idx := BuildIndex(entries)
	hits := idx.Search("дм", map[string]bool{"d1": true})
	if len(hits) == 0 || hits[0].Entry.ID != "a" {
		t.Fatalf("%+v", hits)
	}
	for _, h := range hits {
		if h.Entry.DocURL == "d2" {
			t.Fatal("найден документ другого сервера")
		}
	}
	snip := Snippet(entries[0].Text, QueryTerms("дм"))
	if !strings.Contains(snip, "<mark>DM</mark>") {
		t.Fatalf("%q", snip)
	}
}
