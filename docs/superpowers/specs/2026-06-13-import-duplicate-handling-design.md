# Design: resilient `import` (duplicate-identifier handling)

Date: 2026-06-13

## Problem

`rforge import <format> <file>` calls `library.Store.Create` once per parsed
record and returns on the first error. `Store.Create` errors when a record has
no identifier (`paper record identifier is required`) or when its identity key
already exists (`paper record already exists`). As a result, a single duplicate
or identifier-less record aborts the whole import and leaves the library in a
partial state. This was surfaced by the full-pipeline e2e test.

## Goal

Make `import` resilient: import every new record, skip records that cannot be
stored cleanly, never partially fail on duplicates, and report what was skipped.

## Behavior

For each record in the parsed batch:

- **No identifier** (`recordKey == ""`): skip; increment a skipped-no-identifier
  count.
- **Duplicate**: identity key already present in the store *or* already seen
  earlier in the same batch: skip; record the identifier.
- **Otherwise**: import it.

Identity key is the existing `recordKey` priority order (DOI → OpenAlex → arXiv →
PMID → Crossref → Semantic Scholar). Existing library records are never mutated;
merging duplicates stays a deliberate step via the existing `duplicate merge`
command. A real storage failure (read/write) still returns an error.

## API

Resilience lives at two layers, each owning one concern, because no-identifier
records are rejected by `NewPaperRecord` inside the parsers — before any record
reaches the store.

**Parser layer — skip unstorable records.** The four format parsers
(`ImportJSON`, `ImportBibTeX`, `ImportCSV`, `ImportRIS`) skip records that cannot
be normalized into a storable `PaperRecord` (missing an identifier — or, rarely,
a title) instead of aborting the whole parse, and report how many they skipped.
Structural failures (unreadable file, malformed JSON) still return an error.

```go
// ([]PaperRecord, skippedNoIdentifier int, error)
func ImportJSON(path string) ([]PaperRecord, int, error)
func ImportBibTeX(path string) ([]PaperRecord, int, error)
func ImportCSV(path string) ([]PaperRecord, int, error)
func ImportRIS(path string) ([]PaperRecord, int, error)
```

**Store layer — skip duplicates.** A new method on `library.Store` so
dedup-against-store and the write happen in one place (also replaces the current
O(n²) per-record `Create`/`List`):

```go
// ImportSummary reports the outcome of a resilient batch import.
type ImportSummary struct {
    Imported            int
    SkippedDuplicate    []string // identifiers skipped as duplicates, e.g. "10.1000/ap-1"
    SkippedNoIdentifier int
}

// ImportRecords adds records to the store, skipping records that have no
// identifier or whose identifier already exists in the store or earlier in the
// same batch. It returns an error only on a storage failure.
func (s Store) ImportRecords(records []PaperRecord) (ImportSummary, error)
```

Implementation: `List` existing records once, seed a `seen` set from their keys,
iterate the batch building the merged slice (skipping per the rules above), then
`ReplaceAll` once. `SkippedDuplicate` reports the bare identifier (key with its
`type:` prefix stripped). `ImportRecords` also defensively skips any
no-identifier record, though the parsers normally remove those first.

## CLI

`executeImport` threads the parser's skipped-no-identifier count and replaces
its per-record `Create` loop with one `ImportRecords` call:

```go
records, skippedNoIdentifier, err := library.ImportJSON(path) // or the matching format
summary, err := store.ImportRecords(records)
// imported = summary.Imported
// skipped_duplicate = summary.SkippedDuplicate
// skipped_no_identifier = skippedNoIdentifier
```

- JSON (back-compatible): keeps `"imported": N`; adds `"skipped_duplicate": [...]`
  and `"skipped_no_identifier": M`.
- Plain: `imported N records (skipped X duplicates, Y without identifiers)`.

`duplicate split` also parses replacement records via `ImportJSON`; it ignores
the skipped count (replacement sets are expected to be clean).

## Testing (test-first)

Library store (`internal/library`, `ImportRecords`):

- imports new records and reports `Imported`;
- skips an in-store duplicate and reports its identifier;
- skips an in-batch duplicate (same identifier twice in one call);
- skips a no-identifier record and counts it;
- returns the right summary for a mixed batch.

Library parsers (`internal/library`, `ImportJSON`/`ImportBibTeX`/`ImportCSV`/`ImportRIS`):

- a parser skips a record with no identifier, keeps the valid ones, and returns
  the skipped count;
- a structurally malformed file still returns an error.

CLI (`internal/cli`):

- importing a file containing a duplicate no longer aborts (exit 0) and the JSON
  reports `imported` plus the skip fields;
- a clean import still reports `imported: N` with empty skip fields.

Update any existing test that asserted the old abort/`already exists` behavior.

## Out of scope (YAGNI)

- merge-on-import;
- an `--on-duplicate skip|merge|error` flag;
- new import provenance events.
