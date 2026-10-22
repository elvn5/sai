package i18n_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/elvn5/sai/services/bot/internal/i18n"
)

// Russian words the bot reads in what users type, not says.
var notSaid = map[string]bool{
	"чел": true, "ч": true, "бесплат": true, "с": true, "акысыз": true,
	// Says the same in every language.
	"🌐 Выберите язык · Тилди тандаңыз · Choose a language · Sprache wählen": true,
}

func hasCyrillic(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	return false
}

// literal is the value of a string literal, or of literals joined with +.
func literal(e ast.Expr) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(e.Value)
		return s, err == nil
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		a, ok1 := literal(e.X)
		b, ok2 := literal(e.Y)
		return a + b, ok1 && ok2
	case *ast.ParenExpr:
		return literal(e.X)
	}
	return "", false
}

// sourceTexts are the Russian texts of the bot's code (outside this
// package and tests), with where each is.
func sourceTexts(t *testing.T) map[string]string {
	t.Helper()
	texts := map[string]string{}
	fset := token.NewFileSet()
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "i18n" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		add := func(s string, pos token.Pos) {
			if hasCyrillic(s) && !notSaid[s] {
				texts[s] = fset.Position(pos).String()
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				// l.N(n, "день", "дня", "дней"): the forms are one text.
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "N" && len(n.Args) == 4 {
					var forms []string
					for _, a := range n.Args[1:] {
						if s, ok := literal(a); ok {
							forms = append(forms, s)
						}
					}
					if len(forms) == 3 {
						add(strings.Join(forms, "|"), n.Pos())
						return false
					}
				}
			case *ast.BinaryExpr:
				if s, ok := literal(n); ok {
					add(s, n.Pos())
					return false
				}
			case *ast.BasicLit:
				if s, ok := literal(n); ok {
					add(s, n.Pos())
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(texts) < 100 {
		t.Fatalf("found only %d texts: is the walk looking in the right place?", len(texts))
	}
	return texts
}

var (
	verbs = regexp.MustCompile(`%[-+# 0]*\d*(?:\.\d+)?[a-zA-Z%]`)
	tags  = regexp.MustCompile(`</?[a-z]+[^>]*>`)
)

func sorted(re *regexp.Regexp, s string) []string {
	found := re.FindAllString(s, -1)
	slices.Sort(found)
	return found
}

func TestCatalogsCoverTheCode(t *testing.T) {
	texts := sourceTexts(t)
	for _, l := range i18n.Translated {
		catalog := i18n.Catalog(l)
		for text, where := range texts {
			translated, ok := catalog[text]
			if !ok || translated == "" {
				t.Errorf("%s: no translation of %q (%s)", l, text, where)
				continue
			}
			if hasCyrillic(translated) && l != i18n.KY {
				t.Errorf("%s: %q is translated with Cyrillic: %q", l, text, translated)
			}
			if got, want := sorted(verbs, translated), sorted(verbs, text); !slices.Equal(got, want) {
				t.Errorf("%s: %q has verbs %v, want %v", l, translated, got, want)
			}
			if got, want := sorted(tags, translated), sorted(tags, text); !slices.Equal(got, want) {
				t.Errorf("%s: %q has tags %v, want %v", l, translated, got, want)
			}
			if forms := strings.Count(text, "|"); forms == 2 {
				n := strings.Count(translated, "|") + 1
				if (l == i18n.KY && n != 1) || (l != i18n.KY && n != 2) {
					t.Errorf("%s: %q has %d plural forms", l, translated, n)
				}
			}
		}
		for key := range catalog {
			if _, used := texts[key]; !used {
				t.Errorf("%s: %q is not in the code any more", l, key)
			}
		}
	}
}
