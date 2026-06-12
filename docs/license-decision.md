# License decision brief

ResearchForge does not currently have a project license. Until an owner selects one, the repository should be treated as all-rights-reserved by default.

## Common options

| Option | Typical fit | Trade-offs |
| --- | --- | --- |
| MIT | permissive library/CLI adoption | minimal patent language |
| Apache-2.0 | permissive adoption with patent grant | longer license text |
| GPL-3.0 | strong copyleft research tooling | limits proprietary redistribution |
| AGPL-3.0 | network-copyleft service deployments | strongest adoption constraints |
| No public license yet | owner wants more review | external contributors/users have no reuse grant |

## Owner inputs needed

- intended adoption model: academic, commercial, internal, or mixed;
- patent posture and contributor expectations;
- copyright holder string;
- whether dependencies and example assets are compatible with the chosen license;
- whether dual licensing or a contributor license agreement is desired.

## Implementation after decision

1. Add `LICENSE` with the selected license text.
2. Update `README.md` license section.
3. Update contribution guidance if contributor terms change.
4. Mark `TODO.md` license item complete.
