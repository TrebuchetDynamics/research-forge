# Beta release plan

Beta requires alpha validation plus performance benchmarks, security review, dependency/license scan review, report reproducibility checks, and documented limitations.

Before cutting beta, run `make license-decision-live-audit` and require `make license-decision-approval-gate` to pass with `approved:true`. Issue #1 must include License SPDX identifier, Copyright holder, Approved by, and Approval date.
