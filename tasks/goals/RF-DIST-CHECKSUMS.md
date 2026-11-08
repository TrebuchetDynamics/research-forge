+++
schema = 1
id = "RF-DIST-CHECKSUMS"
title = "Generate repeatable checksums for local distribution artifacts"
priority = 50
source = "Makefile#checksums"
status = "unmet"
focus = false
depends_on = []
+++
- Admission authority: repository-work discovery request round 8 permits one bounded ordinary offline building task for the reproduced local checksum-generation defect. Priority 50 and Next do not displace existing priorities.
- Boundary: RF-DEV-GATES explicitly covers only fmt-check and has no automatic tooling successor; RF-RELEASE remains unmet and its live/release qualification gates are not reopened. This goal covers only repeatable local checksum generation, not release execution, installation, publication or authenticity qualification.
- Preserve all benchmark pauses, provider-input requirements, live/host/Hermes deferrals and scientific, acquisition, privacy and export human gates. No authorization, ownership, instruction or runtime-state change is granted.
