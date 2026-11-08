package webui

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectFormHTTPFeedback(t *testing.T) {
	alpha := seedLibraryFolder(t, "Alpha")
	router := NewRouter(Config{ProjectPath: alpha})
	request := func(method, path string, form url.Values) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	get := func(path string) string {
		t.Helper()
		rec := request(http.MethodGet, path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", path, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}
	// Snapshot all synthetic fixture bytes, not just the rendered active label.
	snapshot := func() map[string]string {
		t.Helper()
		files := map[string]string{}
		err := filepath.WalkDir(alpha, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			data, err := os.ReadFile(path)
			files[path] = string(data)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return files
	}
	before, _ := json.Marshal(snapshot())
	libraryBefore := get("/library")
	activeBefore := get("/projects/active")
	destination := filepath.Join(t.TempDir(), "absent")
	diagnostics := map[string]string{}
	for _, tc := range []struct{ operation, diagnostic string }{
		{"create", "project title is required"},
		{"open", "rforge.project.toml"},
		{"switch", "research folder does not exist"},
	} {
		rec := request(http.MethodPost, "/projects/"+tc.operation, url.Values{"path": {destination}, "title": {" "}})
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tc.diagnostic) || rec.Header().Get("HX-Trigger") != "" {
			t.Fatalf("rejected %s: %d %s %v", tc.operation, rec.Code, rec.Body.String(), rec.Header())
		}
		diagnostics[tc.operation] = rec.Body.String()
		if get("/projects/active") != activeBefore || get("/library") != libraryBefore {
			t.Fatalf("rejected %s changed active folder/library", tc.operation)
		}
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("rejected create made destination: %v", err)
	}
	after, _ := json.Marshal(snapshot())
	if !bytes.Equal(before, after) {
		t.Fatal("rejections changed fixture bytes")
	}

	// Router controls also supply the actual successful fragments to the event fixture.
	success := map[string]string{}
	for _, operation := range []string{"create", "open", "switch"} {
		rec := request(http.MethodPost, "/projects/"+operation, url.Values{"path": {destination}, "title": {"Feedback control"}})
		trigger, fragment := "project-opened", "Created project"
		if operation == "open" {
			fragment = "Opened project"
		}
		if operation == "switch" {
			trigger, fragment = "refresh-library, refresh-artifacts, refresh-papers", `<p id="active-project-path">`+destination+`</p>`
		}
		if rec.Code != http.StatusOK || rec.Header().Get("HX-Trigger") != trigger || !strings.Contains(rec.Body.String(), fragment) {
			t.Fatalf("valid %s: %d %s %v", operation, rec.Code, rec.Body.String(), rec.Header())
		}
		success[operation] = rec.Body.String()
		if operation != "switch" && get("/projects/active") != activeBefore {
			t.Fatalf("%s unexpectedly selected active folder", operation)
		}
	}
	if !strings.Contains(get("/projects/active"), destination) || strings.Contains(get("/library"), "Alpha") {
		t.Fatal("valid switch did not change active folder/library")
	}

	t.Run("wiring", func(t *testing.T) {
		if !strings.Contains(get("/"), `<script src="/assets/project-forms.js" defer></script>`) {
			t.Error("shell must load local deferred project feedback asset")
		}
		projects := get("/projects")
		for _, operation := range []string{"create", "open"} {
			if !strings.Contains(projects, `hx-post="/projects/`+operation+`" hx-target="#project-status"`) {
				t.Errorf("%s form target missing", operation)
			}
		}
		if !strings.Contains(projects, `id="project-status" role="status"`) ||
			!strings.Contains(activeBefore, `id="project-switch-status" role="status"`) ||
			!strings.Contains(activeBefore, `hx-target="#active-project-path" hx-swap="outerHTML"`) {
			t.Error("separate live status regions and unchanged switch success target required")
		}
	})
	t.Run("events", func(t *testing.T) {
		asset := request(http.MethodGet, "/assets/project-forms.js", nil)
		if asset.Code != http.StatusOK || !strings.Contains(asset.Header().Get("Content-Type"), "javascript") {
			t.Fatalf("feedback asset: %d %v", asset.Code, asset.Header())
		}
		input, err := json.Marshal(map[string]any{"asset": asset.Body.String(), "htmx": get("/assets/htmx.min.js"), "diagnostics": diagnostics, "success": success})
		if err != nil {
			t.Fatal(err)
		}
		// Node >=22 is a core prerequisite; absence is a failure, never a skip.
		cmd := exec.Command("node", "-e", projectFormFeedbackFixture)
		cmd.Stdin = bytes.NewReader(input)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("project feedback event fixture: %v\n%s", err, output)
		}
	})
}

// This deliberately models only events/text sinks, not browser rendering or accessibility.
const projectFormFeedbackFixture = `
const assert = require('node:assert/strict');
const vm = require('node:vm');
const input = JSON.parse(require('node:fs').readFileSync(0, 'utf8'));
assert.ok(Number(process.versions.node.split('.')[0]) >= 22);
const listeners = new Map();
let nodes = new Map();
const document = {
  addEventListener(name, fn) {
    if (!listeners.has(name)) listeners.set(name, []);
    listeners.get(name).push(fn);
  },
  getElementById(id) { return nodes.get(id) || null; }
};
const context = vm.createContext({document});
vm.runInContext(input.asset, context);
const count = () => [...listeners.values()].reduce((n, entries) => n + entries.length, 0);
assert.ok(count() > 0, 'asset must delegate on document');
const initialCount = count();
vm.runInContext(input.asset, context);
assert.equal(count(), initialCount, 'reloading asset must not duplicate listeners');
function node(tagName, attrs = {}) {
  return {
    tagName, attrs, writes: 0, children: [], value: 'input stays intact', text: '',
    getAttribute(name) { return this.attrs[name] ?? null; },
    get textContent() { return this.text; },
    set textContent(value) { this.text = value; this.children = []; this.writes++; },
    set innerHTML(value) { throw Error('diagnostics must never be parsed as HTML'); }
  };
}
function reset() {
  // New forms/statuses after startup and body/fragment replacement.
  document.body = node('BODY');
  nodes = new Map(['project-status', 'project-switch-status', 'active-project-path'].map(id => [id, node('DIV')]));
  nodes.get('active-project-path').text = 'Alpha';
}
reset();
// Execute the real vendored response handler to obtain beforeSwap's actual payload.
// Stop at that event: no layout/swap emulation or full-DOM claim.
const start = input.htmx.indexOf('function Mr(');
const end = input.htmx.indexOf('var Xr=', start);
assert.ok(start >= 0 && end > start);
let observed;
const vendorContext = vm.createContext({
  Q: {config: {ignoreTitle: false}}, O: () => false, Pr: () => ({}),
  le: Object.assign,
  ce(origin, name, detail) {
    detail.elt = origin; // vendored ce sets elt to event origin, not initiating form
    if (name !== 'htmx:beforeSwap') return true;
    const event = {type: name, target: origin, detail};
    const policy = [detail.shouldSwap, detail.isError, detail.serverResponse];
    for (const fn of listeners.get(name) || []) fn(event);
    assert.deepEqual([detail.shouldSwap, detail.isError, detail.serverResponse], policy, 'do not alter HTMX policy');
    observed = detail;
    return false;
  }
});
vm.runInContext(input.htmx.slice(start, end), vendorContext);
function emit(form, status, responseText, verb = 'post', path = form?.getAttribute('hx-post')) {
  const target = nodes.get(path === '/projects/switch' ? 'active-project-path' : 'project-status') || document.body;
  const response = {
    xhr: {status, response: responseText, responseText}, target,
    requestConfig: {elt: form, verb, path, target}, etc: {},
    pathInfo: {requestPath: path, finalRequestPath: path}
  };
  vendorContext.Mr(form || document.body, response);
  assert.equal(observed.elt, target, 'beforeSwap originates on swap target');
}
const malicious = '<img src=x onerror="globalThis.executed=true"><script>globalThis.executed=true</script>';
for (const operation of ['create', 'open', 'switch']) {
  reset();
  const form = node('FORM', {'hx-post': '/projects/' + operation});
  form.elements = [node('INPUT'), node('INPUT')];
  const values = form.elements.map(n => n.value);
  const status = nodes.get(operation === 'switch' ? 'project-switch-status' : 'project-status');
  status.text = 'Previous successful operation';
  for (const [code, diagnostic] of [[400, input.diagnostics[operation]], [500, malicious], [400, 'retry rejected']]) {
    const writes = status.writes;
    emit(form, code, diagnostic);
    assert.equal(status.textContent, diagnostic, operation + ' rejection visible');
    assert.equal(status.writes, writes + 1, 'one feedback update per response');
    assert.equal(status.children.length, 0);
    assert.equal(context.executed, undefined);
    assert.equal(nodes.get('active-project-path').textContent, 'Alpha');
    assert.deepEqual(form.elements.map(n => n.value), values);
  }
  const writes = status.writes;
  // Existing rendered success content must not be overwritten with raw HTML as text.
  if (operation !== 'switch') status.children = [{fragment: input.success[operation]}];
  emit(form, 200, input.success[operation]);
  if (operation === 'switch') {
    assert.equal(status.textContent, '');
    assert.equal(nodes.get('active-project-path').textContent, 'Alpha', 'listener leaves path swap to HTMX');
  } else {
    assert.equal(status.writes, writes);
    assert.equal(status.children[0].fragment, input.success[operation]);
  }
  emit(form, 400, 'another rejection');
  assert.equal(status.textContent, 'another rejection');
}
reset();
const form = node('FORM', {'hx-post': '/projects/create'});
const unchanged = () => JSON.stringify([...nodes.values()].map(n => [n.text, n.writes]));
const before = unchanged();
for (const code of [0, 204]) emit(form, code, 'not an HTTP rejection');
for (const other of [null, node('FORM', {'hx-post': '/unrelated'}), node('BUTTON', {'hx-post': '/projects/create'}), node('FORM')]) emit(other, 400, 'ignore');
emit(form, 400, 'ignore', 'get');
emit(form, 400, 'ignore', 'post', '/unrelated');
assert.equal(unchanged(), before, 'unrelated/malformed events must be no-ops');
// A non-error 204 must not clear a switch rejection either.
const switchForm = node('FORM', {'hx-post': '/projects/switch'});
emit(switchForm, 400, 'keep rejection');
emit(switchForm, 204, '');
assert.equal(nodes.get('project-switch-status').textContent, 'keep rejection');
nodes.clear();
emit(form, 400, 'missing target');
emit(switchForm, 400, 'missing target');
for (const fn of listeners.get('htmx:beforeSwap') || []) fn({target: document.body, detail: {}});
`
