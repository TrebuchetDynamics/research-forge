package webui

import (
	"encoding/xml"
	"html"
	"io"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestSourcePlanningHandlerRendersCockpit(t *testing.T) {
	req := httptest.NewRequest("GET", "/sources?question=Do+catalysts+improve+hydrogen+evolution%3F&type=pico&population=hydrogen+evolution&intervention=catalysts&outcome=efficiency", nil)
	rec := httptest.NewRecorder()
	NewSourcePlanningHandler().ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{"Source planning cockpit", "OpenAlex", "Semantic Scholar", "NASA ADS", "DOAJ", "CORE", "Zotero", "JabRef", "Local files", "Reviewer approval required", "rforge protocol plan-sources"} {
		if !strings.Contains(body, want) {
			t.Fatalf("source planning cockpit missing %q:\n%s", want, body)
		}
	}
}

func TestSourcePlanningFormRoundTrip(t *testing.T) {
	components := map[string]string{
		"population": "Population", "intervention": "Intervention", "comparator": "Comparator",
		"outcome": "Outcome", "exposure": "Exposure", "sample": "Sample", "phenomenon": "Phenomenon",
		"design": "Design", "evaluation": "Evaluation", "research-type": "Research type",
	}
	router := NewRouter(Config{})
	preview := func(t *testing.T, values url.Values) string {
		t.Helper()
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest("GET", "/sources?"+values.Encode(), nil))
		if rec.Code != 200 {
			t.Fatalf("GET /sources status = %d: %s", rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		for _, want := range []string{"Reviewer approval required", "Privacy/Auth", "API key must be redacted", "Auth:"} {
			if !strings.Contains(body, want) {
				t.Errorf("preview missing %q", want)
			}
		}
		if strings.Contains(body, "<script>") || strings.Contains(body, `name="injected"`) {
			t.Fatal("user text injected markup")
		}
		return body
	}
	for _, tc := range []struct{ name, framework, selected string }{
		{"default", "", "freeform"}, {"freeform", "freeform", "freeform"},
		{"pico", "pico", "pico"}, {"peco", "peco", "peco"}, {"spider", "spider", "spider"},
		{"unknown", "unsupported", "freeform"}, {"normalized", " PiCo ", "pico"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := url.Values{}
			if tc.name != "default" {
				values.Set("type", tc.framework)
				values.Set("question", `Compare "catalysts" & <materials>?`)
				for name := range components {
					values.Set(name, name+` "><input name="injected"><script> & 'quoted'`)
				}
			}
			for submission := 0; submission < 2; submission++ {
				body := preview(t, values)
				next, labels, selected := sourcePlanningForm(t, body)
				if !reflect.DeepEqual(selected, []string{tc.selected}) {
					t.Errorf("selected options = %v, want [%s]", selected, tc.selected)
				}
				if len(next) != len(components)+2 {
					t.Errorf("successful controls = %v, want question, type and all components", next)
				}
				for name, label := range components {
					if labels[name] != label {
						t.Errorf("%s visible label = %q, want %q", name, labels[name], label)
					}
					if got := next[name]; len(got) != 1 || got[0] != values.Get(name) {
						t.Errorf("%s values = %q, want [%q]", name, got, values.Get(name))
					}
				}
				if next.Get("question") == "" || (values.Get("question") != "" && next.Get("question") != values.Get("question")) {
					t.Errorf("question was lost: %q", next.Get("question"))
				}
				if submission == 1 && tc.selected != "freeform" {
					// Inspect the rendered CLI query, not just values echoed in the form.
					codes := html.UnescapeString(body[strings.Index(body, `<div role="table"`):])
					if !strings.Contains(codes, "rforge search --source openalex --query") {
						t.Error("missing CLI query preview")
					}
					for _, term := range map[string][]string{
						"pico":   {"population", "intervention", "outcome", "comparator"},
						"peco":   {"population", "exposure", "outcome", "comparator"},
						"spider": {"phenomenon", "sample", "design", "evaluation"},
					}[tc.selected] {
						if !strings.Contains(codes, `"`+term+` \"`) {
							t.Errorf("CLI preview lost structured query term %s: %s", term, codes)
						}
					}
				}
				next.Set("question", `Revised "question" & <review>?`)
				values = next
			}
		})
	}
}

// Decode only the native form; permissive XML mode handles HTML void elements and boolean attributes.
func sourcePlanningForm(t *testing.T, body string) (url.Values, map[string]string, []string) {
	t.Helper()
	start, end := strings.Index(body, "<form "), strings.Index(body, "</form>")
	if start < 0 || end < start {
		t.Fatal("missing source-planning form")
	}
	decoder := xml.NewDecoder(strings.NewReader(body[start : end+len("</form>")]))
	decoder.Strict, decoder.AutoClose, decoder.Entity = false, xml.HTMLAutoClose, xml.HTMLEntity
	values, ids, labels := url.Values{}, map[string]string{}, map[string]string{}
	var selected []string
	labelFor, selectName := "", ""
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("parse form: %v", err)
		}
		switch token := token.(type) {
		case xml.StartElement:
			attrs := map[string]string{}
			for _, attr := range token.Attr {
				attrs[attr.Name.Local] = attr.Value
			}
			switch token.Name.Local {
			case "form":
				if attrs["method"] != "get" || attrs["action"] != "/sources" {
					t.Fatal("form must use native GET /sources")
				}
			case "label":
				labelFor = attrs["for"]
			case "input", "select":
				name, id := attrs["name"], attrs["id"]
				if id == "" || ids[id] != "" {
					t.Errorf("missing or duplicate control id %q", id)
				}
				ids[id] = name
				for _, forbidden := range []string{"disabled", "readonly"} {
					if _, exists := attrs[forbidden]; exists {
						t.Errorf("%s is not editable", name)
					}
				}
				if token.Name.Local == "input" {
					if attrs["type"] != "" && attrs["type"] != "text" {
						t.Errorf("%s is not a native text input", name)
					}
					if _, required := attrs["required"]; required && name != "question" {
						t.Errorf("component %s must remain optional", name)
					}
					values.Add(name, attrs["value"])
				} else {
					selectName = name
				}
			case "option":
				if !values.Has(selectName) {
					values.Set(selectName, attrs["value"])
				}
				if _, ok := attrs["selected"]; ok {
					selected = append(selected, attrs["value"])
					values.Set(selectName, attrs["value"])
				}
			}
		case xml.CharData:
			if labelFor != "" {
				labels[labelFor] += string(token)
			}
		case xml.EndElement:
			if token.Name.Local == "label" {
				labelFor = ""
			}
		}
	}
	byName := map[string]string{}
	for id, name := range ids {
		byName[name] = strings.TrimSpace(labels[id])
	}
	return values, byName, selected
}

func TestRouterServesSourcesRoute(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/sources?question=parser+comparison")
	if status != 200 {
		t.Fatalf("GET /sources status = %d", status)
	}
	if !strings.Contains(body, "Source planning cockpit") {
		t.Fatalf("/sources missing cockpit: %s", body)
	}
}
