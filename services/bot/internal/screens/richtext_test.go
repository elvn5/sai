package screens_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/elvn5/sai/services/bot/internal/screens"
)

func TestParagraphs(t *testing.T) {
	got := screens.Paragraphs(`<h2>Что  взять</h2>
		<p>Рюкзак <strong>на 30 л</strong>, <em>палки</em> и <a href="https://x.kg/a?b=1&c=2">термос</a>.</p>
		<ul><li><p>Куртка</p></li><li>Шапка<ul><li>флис</li></ul></li></ul>
		<ol><li>Раз</li><li>Два</li></ol>
		<p><img src="/u.jpg" alt="фото"></p>
		<figure><img src="/a.jpg"><figcaption>Вид на озеро</figcaption></figure>
		<blockquote><p>Горы зовут</p><p>— кто-то</p></blockquote>
		<p>A &lt; B<br>C <a href="/tours/x">свой</a> <a href="javascript:alert(1)">злой</a></p>
		<script>alert(1)</script><pre>  код
  тут</pre>`)
	want := []string{
		"<b>Что взять</b>",
		`Рюкзак <b>на 30 л</b>, <i>палки</i> и <a href="https://x.kg/a?b=1&amp;c=2">термос</a>.`,
		"• Куртка\n• Шапка\n   • флис",
		"1. Раз\n2. Два",
		"<i>Вид на озеро</i>",
		"<blockquote>Горы зовут\n— кто-то</blockquote>",
		"A &lt; B\nC свой злой",
		"<pre>  код\n  тут</pre>",
	}
	if strings.Join(got, "\n¶\n") != strings.Join(want, "\n¶\n") {
		t.Errorf("got:\n%s\n--- want:\n%s", strings.Join(got, "\n¶\n"), strings.Join(want, "\n¶\n"))
	}
	if p := screens.Paragraphs(""); len(p) != 0 {
		t.Errorf("empty: %q", p)
	}
	if p := screens.PlainParagraphs("Раз\n\n\n Два & три \r\n\r\nчетыре"); strings.Join(p, "|") != "Раз|Два &amp; три|четыре" {
		t.Errorf("plain: %q", p)
	}
}

func TestPages(t *testing.T) {
	short := []string{"<b>Один</b>", strings.Repeat("а", 40), strings.Repeat("б", 40)}
	if p := screens.Pages(short, 100); len(p) != 1 || p[0] != "<b>Один</b>\n\n"+short[1]+"\n\n"+short[2] {
		t.Errorf("one page: %q", p)
	}
	// Tags don't count toward the limit; a paragraph that doesn't fit
	// starts the next page.
	if p := screens.Pages(short, 85); len(p) != 2 || p[1] != short[2] {
		t.Errorf("two pages: %q", p)
	}

	long := strings.Repeat("Первое предложение тут. ", 30) // 720 characters
	pages := screens.Pages([]string{"<b>Заголовок</b>", "Текст &amp; ещё", long, "Хвост"}, 300)
	for i, p := range pages {
		if n := utf8.RuneCountInString(p); n > 300+len("<b></b>") {
			t.Errorf("page %d has %d characters", i, n)
		}
		if i > 0 && !strings.HasPrefix(p, "Первое") && p != "Хвост" {
			t.Errorf("page %d starts mid-sentence: %q", i, p[:40])
		}
	}
	if !strings.HasPrefix(pages[0], "<b>Заголовок</b>\n\nТекст &amp; ещё\n\nПервое") || !strings.HasSuffix(pages[len(pages)-1], "тут.\n\nХвост") {
		t.Errorf("pages: %q", pages)
	}
	if got := strings.Count(strings.Join(pages, " "), "Первое предложение тут."); got != 30 {
		t.Errorf("lost text: %d sentences", got)
	}
	if p := screens.Pages(nil, 100); len(p) != 0 {
		t.Errorf("no paragraphs: %q", p)
	}
}
