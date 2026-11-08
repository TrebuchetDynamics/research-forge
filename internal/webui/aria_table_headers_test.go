package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAriaTableHeadersMatchCells guards WCAG 1.3.1: every local cockpit
// role="table" declares one columnheader per cell rendered in a body row.
func TestAriaTableHeadersMatchCells(t *testing.T) {
	dir := t.TempDir()
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)
	writeLocalPDF(t, dir, "10-1000-ap")
	router := NewRouter(Config{ProjectPath: dir})

	for _, route := range []string{"/papers", "/sources"} {
		t.Run(route, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d", route, rec.Code)
			}
			blocks := ariaTableBlocks(rec.Body.String())
			if len(blocks) == 0 {
				t.Fatalf("GET %s rendered no role=\"table\" block", route)
			}
			for i, block := range blocks {
				headers := strings.Count(block, `role="columnheader"`)
				cells := ariaFirstDataRowCells(block)
				if headers != cells {
					t.Errorf("%s table %d: %d columnheader(s) vs %d cell(s) in first data row", route, i, headers, cells)
				}
			}
		})
	}
}

// ariaTableBlocks returns each role="table" element's markup by matching <div> depth.
func ariaTableBlocks(body string) []string {
	var blocks []string
	for search := 0; search < len(body); {
		start := strings.Index(body[search:], `<div role="table"`)
		if start < 0 {
			break
		}
		start += search
		depth, end := 0, -1
		for i := start; i < len(body); {
			open := strings.Index(body[i:], "<div")
			closeIdx := strings.Index(body[i:], "</div>")
			if closeIdx < 0 {
				break
			}
			if open >= 0 && open < closeIdx {
				depth++
				i += open + len("<div")
				continue
			}
			depth--
			i += closeIdx + len("</div>")
			if depth == 0 {
				end = i
				break
			}
		}
		if end < 0 {
			break
		}
		blocks = append(blocks, body[start:end])
		search = end
	}
	return blocks
}

// ariaFirstDataRowCells counts role="cell" markers in the first role="row" that has any.
func ariaFirstDataRowCells(block string) int {
	for _, row := range strings.Split(block, `<div role="row"`)[1:] {
		if cells := strings.Count(row, `role="cell"`); cells > 0 {
			return cells
		}
	}
	return 0
}
