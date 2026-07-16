//go:build playwright_e2e

// Opt-in Playwright browser end-to-end tests for the local research cockpit.
//
// Excluded from the default build by the playwright_e2e build tag so the normal
// `go build/vet/test ./...` gate never needs a browser or the playwright-go
// runtime. Run with:
//
//	RFORGE_RUN_PLAYWRIGHT=1 go test -tags playwright_e2e ./internal/webui
//
// (or `make web-gui-e2e`). It skips cleanly when the Playwright driver or
// browsers are not installed, mirroring the project's other opt-in e2e tests.
package webui

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

func TestPlaywrightDashboard(t *testing.T) {
	if os.Getenv("RFORGE_RUN_PLAYWRIGHT") == "" {
		t.Skip("set RFORGE_RUN_PLAYWRIGHT=1 to run the Playwright dashboard e2e")
	}

	pw, err := playwright.Run()
	if err != nil {
		t.Skipf("Playwright driver unavailable (run `make web-gui-e2e` to install): %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if err != nil {
		t.Skipf("Chromium unavailable: %v", err)
	}
	defer browser.Close()

	// Seed a research folder with a parsed paper, a local PDF, and a citation
	// graph, then serve the real dashboard router against it.
	dir := t.TempDir()
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)
	writeLocalPDF(t, dir, "10-1000-ap")
	writeCitationGraph(t, dir, sampleCitationGraph)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	newPage := func(t *testing.T) playwright.Page {
		t.Helper()
		page, err := browser.NewPage()
		if err != nil {
			t.Fatalf("new page: %v", err)
		}
		return page
	}
	waitVisible := func(t *testing.T, loc playwright.Locator, what string) {
		t.Helper()
		if err := loc.WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(10000),
		}); err != nil {
			t.Fatalf("waiting for %s: %v", what, err)
		}
	}
	goto_ := func(t *testing.T, page playwright.Page, path string) {
		t.Helper()
		if _, err := page.Goto(ts.URL + path); err != nil {
			t.Fatalf("goto %s: %v", path, err)
		}
	}

	t.Run("shell_nav", func(t *testing.T) {
		page := newPage(t)
		defer page.Close()
		goto_(t, page, "/")
		waitVisible(t, page.Locator(`nav a[href="/papers"]`), "papers nav link")
	})

	t.Run("papers_reader_with_pdf", func(t *testing.T) {
		page := newPage(t)
		defer page.Close()

		goto_(t, page, "/papers")
		waitVisible(t, page.GetByText("Artificial Photosynthesis Review"), "paper title in list")

		goto_(t, page, "/papers/10-1000-ap")
		waitVisible(t, page.GetByText("Photosynthesis converts sunlight."), "parsed passage text")
		// Native PDF viewer is embedded for the local PDF.
		if err := page.Locator(`embed[src="/papers/10-1000-ap/pdf"]`).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateAttached,
			Timeout: playwright.Float(10000),
		}); err != nil {
			t.Fatalf("waiting for PDF embed: %v", err)
		}
	})

	t.Run("interactive_graph_click_navigates", func(t *testing.T) {
		page := newPage(t)
		defer page.Close()
		goto_(t, page, "/artifacts")
		node := page.Locator(".citation-graph-interactive a").First()
		waitVisible(t, node, "interactive citation graph node")
		if err := node.Click(); err != nil {
			t.Fatalf("click graph node: %v", err)
		}
		if err := page.WaitForURL("**/papers/10-1000-ap", playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(10000),
		}); err != nil {
			t.Fatalf("expected navigation to paper page after node click: %v", err)
		}
	})

	t.Run("interactive_graph_keyboard_zoom", func(t *testing.T) {
		page := newPage(t)
		defer page.Close()
		goto_(t, page, "/artifacts")
		svg := page.Locator(".citation-graph-interactive")
		waitVisible(t, svg, "interactive citation graph")
		if err := svg.Focus(); err != nil {
			t.Fatalf("focus graph: %v", err)
		}
		if err := page.Keyboard().Press("+"); err != nil {
			t.Fatalf("press zoom key: %v", err)
		}
		// The "+" key zooms in: the viewport <g> gains a scale(1.1...) transform.
		if _, err := page.WaitForFunction(
			`() => { const g = document.querySelector('.citation-graph-interactive g'); return g && /scale\(1\.1/.test(g.getAttribute('transform') || ''); }`,
			nil,
			playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
		); err != nil {
			t.Fatalf("expected viewport to zoom in after pressing '+': %v", err)
		}
	})

	t.Run("htmx_project_switcher", func(t *testing.T) {
		// Own server + folders so switching the active project does not leak into
		// other subtests. Exercises the vendored htmx: the active-project fragment
		// is loaded via hx-get on page load, and switching posts via hx-post.
		alpha := seedLibraryFolder(t, "Alpha")
		beta := seedLibraryFolder(t, "Beta")
		ts2 := httptest.NewServer(NewRouter(Config{ProjectPath: alpha}))
		defer ts2.Close()

		page := newPage(t)
		defer page.Close()
		if _, err := page.Goto(ts2.URL + "/"); err != nil {
			t.Fatalf("goto shell: %v", err)
		}
		// htmx loads the active-folder switch form into the page.
		switchInput := page.Locator("#switch-path")
		if err := switchInput.WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(10000),
		}); err != nil {
			t.Fatalf("htmx did not load the switch form (is htmx working?): %v", err)
		}
		if err := switchInput.Fill(beta); err != nil {
			t.Fatalf("fill switch path: %v", err)
		}
		if err := page.Locator(`button:has-text("Open folder")`).Click(); err != nil {
			t.Fatalf("click switch button: %v", err)
		}
		// htmx swaps the active-project path with the new folder.
		if _, err := page.WaitForFunction(
			`(p) => { const e = document.getElementById('active-project-path'); return e && e.textContent.includes(p); }`,
			beta,
			playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
		); err != nil {
			t.Fatalf("active project did not switch to beta: %v", err)
		}
		// The library now reflects the switched folder.
		if _, err := page.Goto(ts2.URL + "/library"); err != nil {
			t.Fatalf("goto library: %v", err)
		}
		waitVisible(t, page.GetByText("Beta"), "switched-folder paper")
		if n, err := page.GetByText("Alpha").Count(); err != nil || n != 0 {
			t.Fatalf("old folder paper still present after switch (count=%d, err=%v)", n, err)
		}
	})

	t.Run("nojs_static_graph_fallback", func(t *testing.T) {
		ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
			JavaScriptEnabled: playwright.Bool(false),
		})
		if err != nil {
			t.Fatalf("new no-js context: %v", err)
		}
		defer ctx.Close()
		page, err := ctx.NewPage()
		if err != nil {
			t.Fatalf("new no-js page: %v", err)
		}
		if _, err := page.Goto(ts.URL + "/artifacts"); err != nil {
			t.Fatalf("goto artifacts (no-js): %v", err)
		}
		// The server-rendered static SVG fallback is present with clickable nodes.
		if err := page.Locator(`.citation-graph-svg a[href="/papers/10-1000-ap"]`).WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateAttached,
			Timeout: playwright.Float(10000),
		}); err != nil {
			t.Fatalf("no-js static graph fallback missing clickable node: %v", err)
		}
		// The JS-only interactive enhancement must NOT appear without JavaScript.
		if n, err := page.Locator(".citation-graph-interactive").Count(); err != nil || n != 0 {
			t.Fatalf("interactive graph should not render without JS (count=%d, err=%v)", n, err)
		}
	})
}
