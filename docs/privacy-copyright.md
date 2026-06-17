# Privacy and copyright

ResearchForge separates metadata, provenance, and document assets. The local dashboard is a review surface only: it must not publish, upload, or make copyrighted/private material shareable without an explicit gate.

| Asset class | Default permission | Export rule | Review gate | Dashboard behavior |
| --- | --- | --- | --- | --- |
| Local-only paths | Project-local only | Redact absolute paths from shareable outputs | Privacy/licensing review | Show basename plus redaction warning |
| Copyrighted PDFs | Local browser viewing only | Exclude unless license/shareability approved | Legal acquisition approval | Embed only from local project routes |
| Reviewer notes | Private by default | Exclude or redact unless marked shareable | Reviewer approval | Show private-note badge |
| Credentials | Never display secret values | Never export | Connector credential review | Show presence/requirements only |
| Embeddings | Local payload policy required | Export only redacted checksums or approved vectors | Embedding egress/privacy approval | Show provider, dimensions, and payload policy |
| Cache files | Private local state | Exclude from packages and reports | Package redaction audit | Show excluded count/status only |
| Shareable report fields | Allowed after trace/redaction gates | Export only supported claims and approved metadata | Claim traceability panel | Block final export on weak/unresolved claims |

Users remain responsible for source/API terms, publisher licenses, institutional policy, and reviewer consent.
