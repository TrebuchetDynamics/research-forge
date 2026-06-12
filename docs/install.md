# Installation

```sh
git clone https://github.com/TrebuchetDynamics/research-forge
cd research-forge
go test ./...
go build -o bin/rforge ./cmd/rforge
```

Optional services such as GROBID, OpenSearch, Qdrant, and R/metafor are not required for normal tests.
