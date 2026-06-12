.PHONY: fmt test vet vuln check decisions decisions-markdown decision-issues todo-audit build-release checksums sbom install-smoke fyne-smoke external-e2e-artificial-photosynthesis

fmt:
	gofmt -w cmd internal

test:
	go test ./...

vet:
	go vet ./...

vuln:
	govulncheck ./...

check: test vet todo-audit
	git diff --check

decisions:
	go run ./cmd/rforge --json decisions

decisions-markdown:
	go run ./cmd/rforge decisions --markdown

decision-issues:
	go run ./cmd/rforge decisions --issue-body project_license
	go run ./cmd/rforge decisions --issue-body fyne_desktop_build_scope

todo-audit: decisions
	go run ./cmd/rforge decisions --check TODO.md
	grep -n "\\[ \\]" TODO.md

build-release:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/rforge-linux-amd64 ./cmd/rforge
	GOOS=darwin GOARCH=amd64 go build -o dist/rforge-darwin-amd64 ./cmd/rforge
	GOOS=windows GOARCH=amd64 go build -o dist/rforge-windows-amd64.exe ./cmd/rforge

checksums:
	sha256sum dist/* > dist/checksums.txt

sbom:
	go list -m -json all > dist/dependencies.json

install-smoke:
	go run ./cmd/rforge --help >/dev/null
	go run ./cmd/rforge version >/dev/null

fyne-smoke:
	@echo "Fyne packaging deferred until Fyne build decision lands"

external-e2e-artificial-photosynthesis:
	RFORGE_EXTERNAL_E2E_DIR=/home/xel/git/artificial-photosynthesis go test ./internal/cli -run TestExternalE2EArtificialPhotosynthesisWorkspace -count=1 -v
