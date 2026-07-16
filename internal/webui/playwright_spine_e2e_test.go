//go:build playwright_e2e

package webui

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

type spineE2EView struct{ name, path, want string }

func metaAnalysisSpineViews() []spineE2EView {
	return []spineE2EView{
		{"forge", "/forge", "Forge home"},
		{"workbenches", "/workbenches", "HTMX workbenches"},
		{"source-planning", "/sources?question=Do+catalysts+improve+hydrogen+evolution", "Source planning cockpit"},
		{"import-dedupe", "/dedupe", "Dedupe/cluster review"},
		{"legal-acquisition", "/acquisition", "legal acquisition Workbench"},
		{"parser-arbitration", "/parsing", "Parser conflict review"},
		{"retrieval-tuning", "/retrieve", "retrieval tuning Workbench"},
		{"screening", "/screening", "Screening cockpit"},
		{"evidence-extraction", "/evidence", "evidence extraction Workbench"},
		{"meta-analysis", "/analysis", "meta-analysis Workbench"},
		{"report-traceability", "/report", "report traceability Workbench"},
		{"research-map", "/map", "Research map cockpit"},
		{"connector-health", "/connectors", "Connector health"},
		{"reproducibility-export", "/package", "reproducibility/export Workbench"},
		{"information-architecture", "/architecture", "Dashboard information architecture"},
		{"privacy", "/privacy", "Dashboard permissions/privacy model"},
	}
}

func TestPlaywrightMetaAnalysisSpineScreensAndNoJSFallbacks(t *testing.T) {
	if os.Getenv("RFORGE_RUN_PLAYWRIGHT") == "" {
		t.Skip("set RFORGE_RUN_PLAYWRIGHT=1 to run the Playwright spine e2e")
	}
	pw, err := playwright.Run()
	if err != nil {
		t.Skipf("Playwright driver unavailable: %v", err)
	}
	defer pw.Stop()
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if err != nil {
		t.Skipf("Chromium unavailable: %v", err)
	}
	defer browser.Close()

	dir := t.TempDir()
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)
	writeCitationGraph(t, dir, sampleCitationGraph)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	checkContext := func(t *testing.T, ctx playwright.BrowserContext, mode string) {
		t.Helper()
		for _, view := range metaAnalysisSpineViews() {
			t.Run(mode+"/"+view.name, func(t *testing.T) {
				page, err := ctx.NewPage()
				if err != nil {
					t.Fatalf("new page: %v", err)
				}
				defer page.Close()
				if _, err := page.Goto(ts.URL + view.path); err != nil {
					t.Fatalf("goto %s: %v", view.path, err)
				}
				if err := page.Locator("body").WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateVisible, Timeout: playwright.Float(10000)}); err != nil {
					t.Fatalf("body visible: %v", err)
				}
				body, err := page.Locator("body").TextContent()
				if err != nil {
					t.Fatalf("body text: %v", err)
				}
				if !strings.Contains(body, view.want) {
					t.Fatalf("%s missing %q in %s body:\n%s", view.path, view.want, mode, body)
				}
				if mode == "no-js" && !strings.Contains(body, "CLI equivalent") && !strings.Contains(body, "No-JS") && view.path != "/forge" && view.path != "/privacy" && view.path != "/architecture" {
					t.Fatalf("%s no-js fallback missing CLI/no-JS cue:\n%s", view.path, body)
				}
			})
		}
	}

	jsCtx, err := browser.NewContext()
	if err != nil {
		t.Fatalf("new js context: %v", err)
	}
	defer jsCtx.Close()
	checkContext(t, jsCtx, "js")
	noJSCtx, err := browser.NewContext(playwright.BrowserNewContextOptions{JavaScriptEnabled: playwright.Bool(false), DeviceScaleFactor: playwright.Float(1), Viewport: &playwright.Size{Width: 1024, Height: 768}, ReducedMotion: playwright.ReducedMotionReduce})
	if err != nil {
		t.Fatalf("new no-js context: %v", err)
	}
	defer noJSCtx.Close()
	checkContext(t, noJSCtx, "no-js")
}
