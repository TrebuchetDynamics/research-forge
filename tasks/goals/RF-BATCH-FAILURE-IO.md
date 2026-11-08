+++
schema = 1
id = "RF-BATCH-FAILURE-IO"
title = "Stop search sweeps when failure-ledger persistence fails"
priority = 81
source = "docs/cli.md#search-budgets-and-stop-reasons"
status = "unmet"
focus = false
depends_on = []
+++
- Scope: Bounded offline outcome: both incremental failure-ledger append errors stop the batch through existing rollback, with discriminating CLI tests and unchanged resume accounting. No transaction redesign, refresh-path changes, live qualification or successor planning is included.
