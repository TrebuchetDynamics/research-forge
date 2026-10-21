# V-FS2 campaign-recovery review

Disposition: blocked on an owner scope decision; no task-contract or ledger change is justified. The historical filename is retained. The earlier exhausted-budget diagnosis is superseded by the owner's withdrawal of all V-FS2 budgets.

## Sources

- [Current task](../../tasks/items/V-FS2.md): both Acceptance items, Budget, Next action and Resume authorization. The 2026-10-21 owner decision removes all token, cost and time budgets; the additional Majorana resume does not authorize repeating the campaign.
- [Accepted authorization](../benchmark/frontier-science-plan.md#authorized-v-fs2-campaign): one genuine workflow per topic, existing authorized API access only, no repeat without a new owner decision, and unchanged acquisition/privacy/claim/export gates. Its explicit no-budget amendment supersedes the original spend/token/time maxima, not the search caps or required 120-second timeout.
- [Report and reproduction contract](../benchmark/frontier-science-plan.md#v-fs2-report-source-audit-and-reproduction-contract): actual engine artifacts, readable source-backed reports, citation audits, reproduction instructions and real provenance checks remain required.
- [Engine boundary](../verifiable-engine-plan.md#owner-decisions-2026-10-19): host-agent semantic judgment remains separate from engine operations.
- Retained local worker evidence in the common Git directory: `.git/autowork-execution/slot-data/V-FS2/attempt-1/report.json` and `.git/autowork-execution/slots/V-FS2.json`. These are operational evidence references, not public research artifacts. The slot identifies baseline `6eafff63e9aab49340d22211b4b7fc6c78cab54c` and a held state with cause `blocker`.

## Observed blocker

The latest worker report records that the original uncommitted campaign, `research/frontier-science/frontier-20261021-vfs2-live/`, is unavailable. Missing artifacts include Majorana `batch-state.json`, both `results.jsonl` corpora, immutable provenance receipts/events, command records, `authorization.md` and `run.json`. The original fixed binary and child-session artifacts are also unavailable. This checkout has no tracked files under `research/frontier-science/` and no campaign directory there.

The worker reports checking the assigned worktree, temporary storage, repository execution storage and earlier task tips. The archived attempt recorded campaign paths but no commit; metadata/logs and recorded call counts do not recover the underlying research outputs. The historical Majorana 75/100-call checkpoint and nickelates 24/50-call workflow cannot be resumed or audited from their names or counts. The worker did not invent a checkpoint, fabricate receipts, start a replacement search or consume the additional resume against an empty directory.

The report records four passing V-FS1 checker tests, a passing packet-only structural check and a clean diff on the current baseline. Synthetic test fixtures are not research evidence. Actual campaign `check.py --run`, per-topic engine provenance validation and report/evidence/source audits remain unavailable without the genuine outputs; neither topic qualifies as a finished public example.

All budget stops in the previous review are obsolete. Actual usage remains informational evidence, not grounds for another budget request or hold. Withdrawing budget limits does not restore the campaign or authorize a replacement run.

## Maintenance decision

Leave V-FS2's contract, both Acceptance items, Budget, status and queue position unchanged. Leave [V-FS3](../../tasks/items/V-FS3.md) dependent on V-FS2. Do not close either task, promote a partial/synthetic dossier, grant replacement-acquisition authority or add a speculative recovery producer that promises unavailable inputs. No Blocker resolution marker is warranted because the blocker has not been addressed.

The latest V-FS2 report has `commit: null` and `changed_files: []`; it supplies no integrated candidate or saved task-file patch to apply. Task validation reports no pending proposals or ledger errors. No older planning or task closure is undone.

Further execution requires an explicit owner decision authorizing replacement acquisition despite the no-repeat constraint, or explicitly approving another recovery route. This review grants neither. Any authorized continuation must preserve the known campaign history, perform only authorized real operations, and still deliver the existing two-topic report/audit/provenance contract before V-FS3 can promote the examples. No new credentials, spending authority, settings changes or scientific approvals are implied.
