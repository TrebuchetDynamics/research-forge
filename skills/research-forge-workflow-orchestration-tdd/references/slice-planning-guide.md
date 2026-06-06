# Slice planning guide

Good slices are:

- one observable behavior;
- testable without live APIs by default;
- small enough to finish in one session;
- tied to a milestone exit criterion;
- safe to validate with a command;
- explicit about CLI/UI parity expectations.

Bad slices:

- "build ingestion";
- "add UI";
- "improve architecture";
- "support all reports".

Better slices:

- `rforge project create` writes manifest and event log;
- OpenAlex fixture response normalizes one work into `PaperRecord`;
- `screen decide` rejects unknown exclusion reason;
- Markdown report golden output includes search query audit section.
