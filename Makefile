.PHONY: fmt fmt-check test vet vuln ci check inventory-check decisions decisions-markdown decision-issues todo-audit todo-completion-audit license-decision-live-audit license-decision-approval-gate build-release checksums sbom install install-smoke web-gui-smoke source-live-smoke biomedical-live-smoke semantic-scholar-live-smoke external-e2e-artificial-photosynthesis

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

fmt:
	gofmt -w cmd internal

fmt-check:
	test -z "$(shell gofmt -l cmd internal)"

test:
	go test ./...

vet:
	go vet ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

ci: fmt-check test vet todo-completion-audit vuln
	go list -m all >/dev/null
	go list -m -json all >/dev/null
	git diff --check

check: test vet todo-completion-audit inventory-check
	git diff --check

inventory-check:
	go run ./cmd/rforge oss inventory-check opensource/inventory/manifest.json

decisions:
	go run ./cmd/rforge --json decisions

decisions-markdown:
	go run ./cmd/rforge decisions --markdown

decision-issues:
	go run ./cmd/rforge decisions --issue-body project_license

todo-audit: decisions
	go run ./cmd/rforge decisions --check TODO.md
	@grep -n "\\[ \\]" TODO.md || echo "no unchecked TODO items remain"

todo-completion-audit: decisions
	go run ./cmd/rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md

license-decision-live-audit:
	gh issue view 1 --json title,state,body,comments,labels,milestone --jq 'def text: ([.body] + [.comments[].body]) | join("\n---\n"); def has_spdx: (text | test("License SPDX identifier: (MIT|Apache-2\\.0|GPL-3\\.0-(only|or-later)|AGPL-3\\.0-(only|or-later)|NOASSERTION)")); def has_holder: (text | test("Copyright holder: [^<\\n][^\\n]+")); def has_approver: (text | test("Approved by: [^<\\n][^\\n]+")); def has_date: (text | test("Approval date: [0-9]{4}-[0-9]{2}-[0-9]{2}")); {title, state, labels: [.labels[].name], milestone: (.milestone.title // null), has_spdx: has_spdx, has_holder: has_holder, has_approver: has_approver, has_date: has_date, approved: (has_spdx and has_holder and has_approver and has_date)}'

license-decision-approval-gate:
	@$(MAKE) -s license-decision-live-audit | grep -q '"approved":true' || (echo "license decision approval missing: issue #1 must report approved:true" >&2; exit 1)

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/rforge

build-release:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/rforge-linux-amd64       ./cmd/rforge
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/rforge-darwin-amd64      ./cmd/rforge
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/rforge-windows-amd64.exe ./cmd/rforge

checksums:
	sha256sum dist/* > dist/checksums.txt

sbom:
	go list -m -json all > dist/dependencies.json

install-smoke:
	go run ./cmd/rforge --help >/dev/null
	go run ./cmd/rforge version >/dev/null

web-gui-smoke:
	go test ./internal/webui

web-gui-e2e:
	go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium
	RFORGE_RUN_PLAYWRIGHT=1 go test -tags playwright_e2e ./internal/webui -run TestPlaywright -count=1 -v

web-gui-screenshots-update:
	go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install chromium
	RFORGE_RUN_PLAYWRIGHT=1 RFORGE_UPDATE_SCREENSHOTS=1 go test -tags playwright_e2e ./internal/webui -run TestPlaywrightScreenshotRegression -count=1 -v

source-live-smoke:
	RFORGE_RUN_LIVE_SOURCE_SMOKE=1 go test ./internal/sources -run TestOptInLiveSourceConnectorSmoke -count=1 -v

biomedical-live-smoke:
	RFORGE_RUN_LIVE_SOURCE_SMOKE=1 go test ./internal/sources -run 'TestOptInLiveSourceConnectorSmoke/(pubmed|europepmc)' -count=1 -v

semantic-scholar-live-smoke:
	RFORGE_RUN_LIVE_SOURCE_SMOKE=1 go test ./internal/sources -run 'TestOptInLiveSourceConnectorSmoke/semantic-scholar' -count=1 -v

external-e2e-artificial-photosynthesis:
	RFORGE_EXTERNAL_E2E_DIR=/home/xel/git/artificial-photosynthesis go test ./internal/cli -run TestExternalE2EArtificialPhotosynthesisWorkspace -count=1 -v
