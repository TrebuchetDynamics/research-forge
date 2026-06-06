# OSS safety rules

ResearchForge may study external repositories, but development must preserve these boundaries:

- `opensource/clones/` is gitignored.
- committed inventory contains metadata, notes, licenses, risk summaries, and integration decisions only;
- external source snippets are not copied into production code without explicit review;
- clone tests use local repositories or mocked command runners;
- reports distinguish observed metadata from human conclusions.

Recommended test cases:

- invalid repo names are rejected;
- clone destination cannot escape `opensource/clones/`;
- duplicate `oss add` merges or rejects deterministically;
- license scanner handles missing license files;
- report output is stable under sorted inventory order.
