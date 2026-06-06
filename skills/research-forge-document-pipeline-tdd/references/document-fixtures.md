# Document fixtures

Use only fixtures that are legal to commit:

- generated minimal PDFs created for tests;
- public-domain or explicitly licensed tiny samples with license noted;
- mocked GROBID TEI XML;
- artificial section/reference/table data.

Golden assertions:

- checksum remains stable;
- parser maps title/sections/references/passages correctly;
- passage IDs are deterministic across repeated parses;
- retrieval results include paper ID, asset ID, section ID, passage ID, and character offsets where available.
