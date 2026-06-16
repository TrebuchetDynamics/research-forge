//go:build playwright_e2e

// Opt-in Playwright browser end-to-end test for the local research cockpit.
//
// It is excluded from the default build by the playwright_e2e build tag so the
// normal `go build/vet/test ./...` gate never needs a browser or the
// playwright-go runtime. Run it with:
//
//	RFORGE_RUN_PLAYWRIGHT=1 go test -tags playwright_e2e ./internal/webui
//
// It additionally skips cleanly (rather than failing) when the Playwright
// driver or browsers are not installed, mirroring the project's other opt-in
// e2e tests (live source smoke, GROBID, R/metafor).
package webui

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/playwright-community/playwright-go"
)

func TestPlaywrightDashboardReadsPapersAndInteractiveGraph(t *testing.T) {
	if os.Getenv("RFORGE_RUN_PLAYWRIGHT") == "" {
		t.Skip("set RFORGE_RUN_PLAYWRIGHT=1 to run the Playwright dashboard e2e")
	}

	pw, err := playwright.Run()
	if err != nil {
		t.Skipf("Playwright driver unavailable (run `go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps chromium`): %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if err != nil {
		t.Skipf("Chromium unavailable: %v", err)
	}
	defer browser.Close()

	// Seed a research folder with a parsed paper and a citation graph, then serve
	// the real dashboard router against it.
	dir := t.TempDir()
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)
	writeCitationGraph(t, dir, sampleCitationGraph)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("new page: %v", err)
	}

	waitVisible := func(loc playwright.Locator, what string) {
		t.Helper()
		if err := loc.WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateVisible,
			Timeout: playwright.Float(10000),
		}); err != nil {
			t.Fatalf("waiting for %s: %v", what, err)
		}
	}

	// Shell renders with navigation to the papers reader.
	if _, err := page.Goto(ts.URL + "/"); err != nil {
		t.Fatalf("goto shell: %v", err)
	}
	waitVisible(page.Locator(`nav a[href="/papers"]`), "papers nav link")

	// Papers list shows the parsed paper.
	if _, err := page.Goto(ts.URL + "/papers"); err != nil {
		t.Fatalf("goto papers: %v", err)
	}
	waitVisible(page.GetByText("Artificial Photosynthesis Review"), "paper title in list")

	// Paper detail renders parsed full text.
	if _, err := page.Goto(ts.URL + "/papers/10-1000-ap"); err != nil {
		t.Fatalf("goto paper detail: %v", err)
	}
	waitVisible(page.GetByText("Photosynthesis converts sunlight."), "parsed passage text")

	// Artifacts page: the vendored JS enhances the static SVG into the
	// interactive graph (identified by the citation-graph-interactive class).
	if _, err := page.Goto(ts.URL + "/artifacts"); err != nil {
		t.Fatalf("goto artifacts: %v", err)
	}
	interactiveNode := page.Locator(".citation-graph-interactive a").First()
	waitVisible(interactiveNode, "interactive citation graph node")

	// Clicking a graph node navigates to that paper's reading page.
	if err := interactiveNode.Click(); err != nil {
		t.Fatalf("click graph node: %v", err)
	}
	if err := page.WaitForURL("**/papers/10-1000-ap", playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(10000),
	}); err != nil {
		t.Fatalf("expected navigation to paper page after node click: %v", err)
	}
}
