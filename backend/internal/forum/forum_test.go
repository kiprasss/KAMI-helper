package forum

import (
	"strconv"
	"strings"
	"testing"
)

const testBase = "https://forum.nova-rp.com"

const threadHTML = `
<html><head><title>Тема | Форум</title></head><body>
<div class="p-breadcrumbs"><ul>
  <li><a href="/"><span>Форумы</span></a></li>
  <li><a href="/forums/1007/"><span>Жалобы</span></a></li>
  <li><a href="/forums/1010/"><span>Жалобы на игроков</span></a></li>
</ul></div>
<h1 class="p-title-value"><span class="label label--orange">Ожидает рассмотрения</span><span class="label-append">&nbsp;</span>5.4 ОП I 54321</h1>
<div class="js-threadStatusField">
  <dl class="pairs pairs--columns pairs--customField" data-field="PlayerReportID2"><dt>Ник игрока</dt><dd>John Doe</dd></dl>
  <dl class="pairs pairs--columns pairs--customField" data-field="PlayerReportSuspect2"><dt>ID нарушителя</dt><dd>54321</dd></dl>
  <dl class="pairs pairs--columns pairs--customField" data-field="PlayerReportDate2"><dt>Дата нарушения</dt><dd>13.10.2026</dd></dl>
  <dl class="pairs pairs--columns pairs--customField" data-field="PlayerReportTime2"><dt>Время нарушения</dt><dd>0,05</dd></dl>
  <dl class="pairs pairs--columns pairs--customField" data-field="PlayerReportProof2"><dt>Доказательства</dt><dd><a href="https://youtu.be/x">https://youtu.be/x</a></dd></dl>
</div>
<article class="message message--post js-post" data-author="Test Author" data-content="post-1819563" id="js-post-1819563">
  <article class="message-body js-selectToQuote"><div>
    <div class="bbWrapper"><b>5.4</b> Запрещено любое намеренное провоцирование сотрудников государственных структур (в том числе и EMS) на погони и задержания. |<b> JAIL от 120 до 180 минут | WARN при повторном нарушении.</b></div>
  </div></article>
</article>
<article class="message message--post js-post" data-author="Admin" data-content="post-1819600" id="js-post-1819600">
  <article class="message-body"><div><div class="bbWrapper">Жалоба одобрена.<br />Игрок будет наказан.</div></div></article>
</article>
<ul class="pageNav-main"><li class="pageNav-page"><a href="/threads/410706/">1</a></li><li class="pageNav-page"><a href="/threads/410706/page-2">2</a></li></ul>
</body></html>`

const rulesText = `1. Общее положение
1.1 При регистрации аккаунта игрок автоматически соглашается с соблюдением всех правил проекта.

2. Игровые аккаунты и никнеймы
2.1. Запрещено иметь более 1 дополнительного аккаунта. | BAN от 30 дней основного аккаунта | BAN навсегда всех дополнительных аккаунтов.

2.2 Ответственность за безопасность игрового аккаунта несет его владелец.
2.3 Запрещена продажа / обмен игровых аккаунтов. | BAN навсегда.
5.4 Запрещено намеренное провоцирование сотрудников государственных структур на погони. | JAIL от 120 до 180 минут | WARN при повторном нарушении.
Дата публикации: 13.10.2026`

const forumHTML = `
<html><body>
<h1 class="p-title-value">Жалобы на игроков</h1>
<div class="node-title"><a href="/forums/1011/">Проверенные жалобы на игроков</a></div>
<div class="structItemContainer-group structItemContainer-group--sticky">
  <div class="structItem structItem--thread" data-author="Forum Admin">
    <div class="structItem-title">
      <a href="/forums/1010/?prefix_id=4" class="labelLink"><span class="label">ВАЖНО</span></a>
      <a href="/threads/409510/" data-tp-primary="on">Правила подачи жалоб и репорт-тикетов на игроков</a>
    </div>
    <time class="structItem-latestDate u-dt" datetime="2026-05-15T17:18:07+0300" data-timestamp="1778854687" data-time="17:18">15.05.2026</time>
  </div>
</div>
<div class="structItemContainer-group">
  <div class="structItem structItem--thread" data-author="Test Author">
    <div class="structItem-title"><a href="/threads/5-4-op.410706/" data-tp-primary="on">5.4 ОП I 54321</a></div>
    <time class="structItem-latestDate u-dt" data-timestamp="1791900000">13.10.2026</time>
  </div>
</div>
<ul><li class="pageNav-page"><a href="/forums/1010/">1</a></li><li class="pageNav-page"><a href="/forums/1010/page-7">7</a></li></ul>
</body></html>`

func TestParseThread(t *testing.T) {
	th, err := ParseThread(threadHTML, testBase+"/threads/410706/")
	if err != nil {
		t.Fatal(err)
	}
	if th.Title != "5.4 ОП I 54321" || th.Prefix != "Ожидает рассмотрения" {
		t.Fatalf("заголовок %q, префикс %q", th.Title, th.Prefix)
	}
	if len(th.Fields) != 5 || th.Fields[1].Value != "54321" {
		t.Fatalf("поля: %+v", th.Fields)
	}
	if len(th.Posts) != 2 || th.Posts[0].Author != "Test Author" || th.Posts[0].URL != testBase+"/threads/410706/#post-1819563" {
		t.Fatalf("сообщения: %+v", th.Posts)
	}
	if !strings.HasPrefix(th.Posts[0].Text, "5.4 Запрещено") || !strings.Contains(th.Posts[0].Text, "JAIL от 120 до 180 минут") {
		t.Fatalf("текст: %q", th.Posts[0].Text)
	}
	if th.Posts[1].Text != "Жалоба одобрена.\nИгрок будет наказан." {
		t.Fatalf("текст второго сообщения: %q", th.Posts[1].Text)
	}
	if th.TotalPages != 2 || strings.Join(th.Breadcrumbs, "|") != "Форумы|Жалобы|Жалобы на игроков" {
		t.Fatalf("страницы %d, крошки %v", th.TotalPages, th.Breadcrumbs)
	}
}

func TestParseForum(t *testing.T) {
	f, err := ParseForum(forumHTML, testBase)
	if err != nil {
		t.Fatal(err)
	}
	if f.Title != "Жалобы на игроков" || len(f.Threads) != 2 {
		t.Fatalf("раздел: %+v", f)
	}
	if f.Threads[0].URL != testBase+"/threads/409510/" || !f.Threads[0].Sticky || f.Threads[0].LastPostTs != 1778854687000 {
		t.Fatalf("первая тема: %+v", f.Threads[0])
	}
	if f.Threads[1].URL != testBase+"/threads/410706/" || f.Threads[1].Sticky {
		t.Fatalf("вторая тема: %+v", f.Threads[1])
	}
	if len(f.Subforums) != 1 || f.Subforums[0].URL != testBase+"/forums/1011/" || f.Subforums[0].Title != "Проверенные жалобы на игроков" || f.TotalPages != 7 {
		t.Fatalf("подразделы %+v, страниц %d", f.Subforums, f.TotalPages)
	}
}

func TestExtractRules(t *testing.T) {
	rules := ExtractRules(rulesText, 6)
	var points []string
	byPoint := map[string]ExtractedRule{}
	for _, r := range rules {
		points = append(points, r.Point)
		byPoint[r.Point] = r
	}
	if strings.Join(points, ",") != "1.1,2.1,2.2,2.3,5.4" {
		t.Fatalf("пункты: %v", points)
	}
	if byPoint["5.4"].Punishment != "JAIL от 120 до 180 минут | WARN при повторном нарушении" || !byPoint["5.4"].RepeatWarn {
		t.Fatalf("5.4: %+v", byPoint["5.4"])
	}
	if byPoint["2.3"].Punishment != "BAN навсегда" || byPoint["2.3"].RepeatWarn {
		t.Fatalf("2.3: %+v", byPoint["2.3"])
	}
	if byPoint["2.2"].Punishment != "" || !strings.HasPrefix(byPoint["2.1"].Text, "Запрещено иметь более 1") {
		t.Fatalf("2.1/2.2: %+v %+v", byPoint["2.1"], byPoint["2.2"])
	}
	if text, p := SplitPunishment("Запрещён DM. Наказание: JAIL 60 минут."); text != "Запрещён DM." || p != "JAIL 60 минут" {
		t.Fatalf("Наказание: %q %q", text, p)
	}
	if _, p := SplitPunishment("Текст | ещё текст"); p != "" {
		t.Fatalf("лишнее наказание: %q", p)
	}
}

func TestChunkText(t *testing.T) {
	if c := ChunkText("Привет", 900, 120); len(c) != 1 || c[0] != "Привет" {
		t.Fatalf("короткий текст: %v", c)
	}
	var lines []string
	for i := 1; i <= 60; i++ {
		lines = append(lines, strconv.Itoa(i)+".1 Правило номер "+strconv.Itoa(i)+", которое что-то запрещает.")
	}
	chunks := ChunkText(strings.Join(lines, "\n"), 300, 50)
	if len(chunks) <= 5 {
		t.Fatalf("фрагментов %d", len(chunks))
	}
	for _, c := range chunks {
		if runeLen(c) > 300 {
			t.Fatalf("длина %d", runeLen(c))
		}
	}
	if !strings.Contains(strings.Join(chunks, "\n"), "60.1 Правило номер 60") {
		t.Fatal("потерян последний пункт")
	}
	if c := ChunkText(strings.Repeat("а", 2000), 500, 0); len(c) != 4 {
		t.Fatalf("длинная строка: %d", len(c))
	}
}

func TestChunkRulesTextSections(t *testing.T) {
	filler := func(n, tail string) string {
		return n + " Запрещено " + strings.Repeat("что-то очень длинное ", 20) + "| " + tail
	}
	text := strings.Join([]string{
		"Общее положение",
		filler("1.1", "JAIL до 120 минут."),
		filler("1.2", "JAIL до 120 минут."),
		"Правила для государственных структур",
		"2.1 В мероприятии могут принять участие только сотрудники государственных структур. | JAIL до 180 минут.",
		filler("2.2", "JAIL до 120 минут."),
		"Правила для криминальных организаций",
		"3.1 В мероприятии могут принять участие только члены криминальных фракций. | JAIL до 180 минут.",
		filler("3.2", "JAIL до 120 минут."),
	}, "\n")
	chunks := ChunkRulesText(text, 1200, 600)
	find := func(s string) string {
		for _, c := range chunks {
			if strings.Contains(c, s) {
				return c
			}
		}
		return ""
	}
	gov, crime := find("2.1 В мероприятии"), find("3.1 В мероприятии")
	if !strings.Contains(gov, "Правила для государственных структур") || !strings.Contains(crime, "Правила для криминальных организаций") || strings.Contains(crime, "Правила для государственных структур") {
		t.Fatalf("разделы: %q / %q", gov, crime)
	}
	if !strings.HasPrefix(find("2.2 Запрещено"), "Правила для государственных структур") {
		t.Fatalf("2.2 без заголовка: %q", find("2.2 Запрещено"))
	}

	text2 := strings.Join([]string{
		filler("6.11", "Устный выговор лидеру."),
		"6.12 После стрелы проигравшая сторона получает иммунитет на 24 часа.",
		"Примечание: иммунитет может быть снят.",
		"1793244540193.png",
		"Активности в опасном районе:",
		`"Война в гетто"`,
		`7.1 Правила активности "Война в гетто" действуют с 22:00 до 07:00 только на территории опасного района.`,
		filler("7.2", "Устный выговор лидеру."),
		"7.7 Разрешено NLR, PG, RK, TK, DM, уход от RolePlay-ситуации.",
	}, "\n")
	chunks = ChunkRulesText(text2, 1200, 600)
	if c := find("7.7 Разрешено"); !strings.HasPrefix(c, `Активности в опасном районе: "Война в гетто"`) {
		t.Fatalf("7.7: %q", c)
	}
	if strings.HasPrefix(find("6.12 После"), "Активности") {
		t.Fatal("раздел 6 получил чужой заголовок")
	}
}

func TestDropOldWording(t *testing.T) {
	in := "Изменения:\nБыло:\n2.5 Старый текст\nСтало:\n2.5 Новый текст"
	if out := DropOldWording(in); out != "Изменения:\n2.5 Новый текст" {
		t.Fatalf("%q", out)
	}
}

func TestNormalizeURL(t *testing.T) {
	u, ok := NormalizeURL("https://forum.grand-rp.su/threads/nazvanie.410706/page-2#post-1", testBase)
	if !ok || u.URL != testBase+"/threads/410706/" || u.Kind != KindThread {
		t.Fatalf("%+v %v", u, ok)
	}
	if _, ok := NormalizeURL("https://example.com/threads/1/", testBase); ok {
		t.Fatal("чужой сайт принят")
	}
	if IsBotChallenge(200, "<title>Проверка браузера</title>") != true || IsBotChallenge(401, `{"ok":false}`) != true || IsBotChallenge(200, "<html>форум</html>") {
		t.Fatal("распознавание проверки браузера")
	}
}
