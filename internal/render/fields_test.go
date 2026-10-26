package render

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rossedman/gh-fleet/internal/ui"
)

func TestTableForPipes(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	p := Printer{Out: &b, Styles: ui.Color()}
	p.Table([]string{"REPO", "NOTE"}, [][]string{{"acme/api", "line one\nline\ttwo", "extra"}, {"a/b", ""}}, func(_, _ int, s string) string { return "[" + s + "]" })
	if want := "acme/api\tline one line two\na/b\t\n"; b.String() != want {
		t.Fatalf("table = %q, want %q", b.String(), want)
	}
}

type inner struct {
	Deep string `json:"deep"`
}

type record struct {
	inner
	Repo    string `json:"repo"`
	Number  int    `json:"number,omitempty"`
	Skip    string `json:"-"`
	Plain   string
	private string
}

func TestFieldsOf(t *testing.T) {
	t.Parallel()
	got := FieldsOf([]record{}, &record{}, map[string]int{}, struct {
		Repo string `json:"repo"`
		Tags []string
	}{})
	if want := []string{"Plain", "Tags", "deep", "number", "repo"}; !slices.Equal(got, want) {
		t.Fatalf("fields = %v, want %v", got, want)
	}
	_ = record{}.private
}

func TestJSONFieldsAndTemplate(t *testing.T) {
	t.Parallel()
	recs := []record{{Repo: "acme/api", Number: 3, Plain: "x"}, {Repo: "acme/web"}}
	var b bytes.Buffer
	p := Printer{Out: &b, JSON: true, Fields: []string{"repo", "number"}}
	if err := p.PrintJSON(recs); err != nil {
		t.Fatal(err)
	}
	if want := "[\n  {\n    \"number\": 3,\n    \"repo\": \"acme/api\"\n  },\n  {\n    \"repo\": \"acme/web\"\n  }\n]\n"; b.String() != want {
		t.Fatalf("fields = %q", b.String())
	}
	b.Reset()
	p = Printer{Out: &b, Fields: []string{"repo"}, JQ: ".repo"}
	if err := p.PrintJSON(recs[0]); err != nil || b.String() != "acme/api\n" {
		t.Fatalf("object fields with jq = %q, %v", b.String(), err)
	}

	b.Reset()
	p = Printer{Out: &b, Template: `{{range .}}{{.repo}} {{truncate 4 .Plain}}{{"\n"}}{{end}}{{join "," (pluck "repo" .)}} {{json (index . 0)}}`}
	if !p.Machine() {
		t.Fatal("template not machine")
	}
	if err := p.PrintJSON([]map[string]any{{"repo": "a/b", "Plain": "abcdefg"}, {"repo": "c/d", "Plain": "ab"}}); err != nil {
		t.Fatal(err)
	}
	if want := "a/b a...\nc/d ab\na/b,c/d {\"Plain\":\"abcdefg\",\"repo\":\"a/b\"}"; b.String() != want {
		t.Fatalf("template = %q", b.String())
	}
	b.Reset()
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	p = Printer{Out: &b, Template: `{{timefmt "2006-01-02" .at}}`}
	if err := p.PrintJSON(map[string]time.Time{"at": when}); err != nil || b.String() != "2026-01-02" {
		t.Fatalf("timefmt = %q, %v", b.String(), err)
	}
	for _, bad := range []string{"{{", "{{.x.y.z}}", `{{timefmt "x" .at}}`} {
		p = Printer{Out: &b, Template: bad}
		if err := p.PrintJSON(map[string]any{"x": 1, "at": "nope"}); err == nil || !strings.Contains(err.Error(), "--template") {
			t.Errorf("%q: err = %v", bad, err)
		}
	}
	if _, err := SelectFields([]byte("{"), []string{"a"}); err == nil {
		t.Error("bad JSON accepted")
	}
	if got, _ := SelectFields([]byte(`[1,{"a":1,"b":2}]`), []string{"a"}); string(got) != `[1,{"a":1}]` {
		t.Errorf("mixed array = %s", got)
	}
}
