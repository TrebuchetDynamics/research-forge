# Outbound API data flow

Normal tests use fixtures or mock HTTP. Live calls are opt-in.

| Integration | Outbound data | Credentials | Redaction |
| --- | --- | --- | --- |
| OpenAlex | query terms, filters, pagination cursor | optional contact/user-agent | none by default |
| arXiv | query terms and limits | optional contact/user-agent | none by default |
| Crossref | query terms and row limits | optional contact/user-agent | configured email if added |
| Unpaywall | DOI and configured email | email | email |
| GROBID | PDF bytes to configured local endpoint | none by default | local paths |
| R/metafor | generated local input table/script | none by default | local paths |
| OSS git clone | repository URL | git credentials if user config provides them | credentials in URLs |
