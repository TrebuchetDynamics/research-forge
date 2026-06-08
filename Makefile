.PHONY: fmt test vet vuln check external-e2e-artificial-photosynthesis

fmt:
	gofmt -w cmd internal

test:
	go test ./...

vet:
	go vet ./...

vuln:
	govulncheck ./...

check: test vet
	git diff --check

external-e2e-artificial-photosynthesis:
	RFORGE_EXTERNAL_E2E_DIR=/home/xel/git/artificial-photosynthesis go test ./internal/cli -run TestExternalE2EArtificialPhotosynthesisWorkspace -count=1 -v
