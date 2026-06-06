# Statistical reproducibility

Each analysis run should record:

- evidence IDs used as inputs;
- generated analysis input file checksum;
- model type and parameters;
- generated script/notebook checksum;
- engine name and version;
- package versions when available;
- output artifacts and checksums;
- warnings and errors.

Testing guidance:

- unit-test generated input tables;
- golden-test generated R scripts;
- parse small fake metafor outputs;
- make real R/metafor integration tests opt-in with an environment variable.
