# Evidence schema notes

A schema should define:

- field name;
- type;
- unit or controlled vocabulary when applicable;
- whether multiple values are allowed;
- required support kind: passage, table, figure, equation, dataset, or citation;
- validation rules;
- export label.

Important tests:

- invalid units fail;
- required fields fail when absent;
- accepted values require source support;
- corrected evidence preserves the previous value in history;
- exports include field, value, units, paper ID, source ID, passage/table ID, status, reviewer, and timestamp.
