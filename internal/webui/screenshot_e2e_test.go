//go:build playwright_e2e

// Opt-in Playwright screenshot regression test for the local research cockpit.
//
// Screenshots are captured in a JavaScript-disabled, fixed-viewport context so
// the rendered output is the deterministic server-rendered HTML/SVG (no htmx or
// graph-JS variance). Goldens are environment-sensitive: they are generated
// against the pinned Chromium that `make web-gui-e2e` installs. Regenerate with:
//
//	make web-gui-screenshots-update
//
// and compare (the default e2e behavior) with:
//
//	make web-gui-e2e
package webui

import (
	"os"
	"path/filepath"
	"testing"

	"net/http/httptest"

	"github.com/playwright-community/playwright-go"
)

// screenshotDiffTolerance is the maximum fraction of pixels allowed to differ
// from the golden before the test fails.
const screenshotDiffTolerance = 0.01

func TestPlaywrightScreenshotRegression(t *testing.T) {
	if os.Getenv("RFORGE_RUN_PLAYWRIGHT") == "" {
		t.Skip("set RFORGE_RUN_PLAYWRIGHT=1 to run the Playwright screenshot regression test")
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

	dir := t.TempDir()
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)
	writeCitationGraph(t, dir, sampleCitationGraph)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	// JS-disabled, fixed-viewport context => deterministic server-rendered output.
	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		JavaScriptEnabled: playwright.Bool(false),
		DeviceScaleFactor: playwright.Float(1),
		Viewport:          &playwright.Size{Width: 1024, Height: 768},
		ReducedMotion:     playwright.ReducedMotionReduce,
	})
	if err != nil {
		t.Fatalf("new context: %v", err)
	}
	defer ctx.Close()

	goldenDir := filepath.Join("testdata", "screenshots")
	update := os.Getenv("RFORGE_UPDATE_SCREENSHOTS") != ""
	if update {
		if err := os.MkdirAll(goldenDir, 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
	}

	views := []struct {
		name     string
		path     string
		settleAt string // selector to wait for before capture
	}{
		{"shell", "/", "h1"},
		{"papers-list", "/papers", `[role="table"]`},
		{"paper-detail", "/papers/10-1000-ap", ".rf-paper-view"},
		{"artifacts", "/artifacts", ".citation-graph-svg"},
	}

	for _, view := range views {
		t.Run(view.name, func(t *testing.T) {
			page, err := ctx.NewPage()
			if err != nil {
				t.Fatalf("new page: %v", err)
			}
			defer page.Close()

			if _, err := page.Goto(ts.URL + view.path); err != nil {
				t.Fatalf("goto %s: %v", view.path, err)
			}
			if err := page.Locator(view.settleAt).First().WaitFor(playwright.LocatorWaitForOptions{
				State:   playwright.WaitForSelectorStateVisible,
				Timeout: playwright.Float(10000),
			}); err != nil {
				t.Fatalf("waiting for %s on %s: %v", view.settleAt, view.path, err)
			}

			shot, err := page.Screenshot(playwright.PageScreenshotOptions{
				FullPage:   playwright.Bool(true),
				Animations: playwright.ScreenshotAnimationsDisabled,
			})
			if err != nil {
				t.Fatalf("screenshot %s: %v", view.path, err)
			}

			goldenPath := filepath.Join(goldenDir, view.name+".png")
			if update {
				if err := os.WriteFile(goldenPath, shot, 0o644); err != nil {
					t.Fatalf("write golden %s: %v", goldenPath, err)
				}
				t.Logf("updated golden %s (%d bytes)", goldenPath, len(shot))
				return
			}

			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s (run `make web-gui-screenshots-update` to create it): %v", goldenPath, err)
			}
			diff, err := comparePNG(golden, shot)
			if err != nil {
				t.Fatalf("compare %s: %v", view.name, err)
			}
			if !diff.SameBounds || diff.DiffRatio > screenshotDiffTolerance {
				actualPath := filepath.Join(goldenDir, view.name+".actual.png")
				_ = os.WriteFile(actualPath, shot, 0o644)
				t.Fatalf("screenshot regression for %s: sameBounds=%v diffRatio=%.4f (tolerance %.4f); wrote %s",
					view.name, diff.SameBounds, diff.DiffRatio, screenshotDiffTolerance, actualPath)
			}
		})
	}
}
