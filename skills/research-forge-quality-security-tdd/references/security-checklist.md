# Security checklist

For each risky change, test:

- malformed paths: `../`, absolute paths, symlinks, reserved names;
- malformed import files and huge inputs;
- command arguments with spaces, quotes, semicolons, and newlines;
- missing/invalid API keys do not leak configured values;
- HTTP clients have timeouts and bounded response sizes where needed;
- external command failures preserve stderr safely;
- archive extraction cannot write outside destination;
- fuzz tests exist for parsers once formats stabilize.

Useful commands:

```sh
go test ./...
go test -race ./...
go vet ./...
govulncheck ./...
```
