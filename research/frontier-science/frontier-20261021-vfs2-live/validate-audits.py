#!/usr/bin/env python3
"""Offline reference consistency only; not source-semantic or scientific approval."""
import csv
import hashlib
import json
from pathlib import Path
import re
import sys


def validate(root):
    for topic in ("majorana", "nickelates"):
        directory = root / topic
        report = (directory / "report.md").read_text()
        anchors = re.findall(r'<a id="([^"]+)"', report)
        assert len(anchors) == len(set(anchors)), f"{topic}: duplicate report anchor"
        with (directory / "evidence.csv").open(newline="") as stream:
            rows = list(csv.DictReader(stream))
        assert rows, f"{topic}: no source evidence"
        audit = json.loads((directory / "citation-audit.json").read_text())
        index = json.loads((directory / "provenance.json").read_text())
        primary = {x["source_id"]: x for x in json.loads(
            (directory / "primary-checks.json").read_text())["records"]}
        assert audit["topic"] == topic
        assert audit["assessed_by"]["kind"] == "host_agent"
        assert audit["assessed_by"]["identity"]
        assert audit["engine_run_ids"] == index["runs"]
        for identity in audit["engine_run_ids"]:
            receipt = json.loads((directory / "provenance/runs" / (identity + ".json")).read_text())
            assert receipt["run_id"] == identity
        seen = set()
        used_rows = set()
        for claim in audit["claims"]:
            identity = claim["id"]
            assert identity not in seen, f"{topic}: duplicate claim {identity}"
            seen.add(identity)
            assert claim["report_anchor"] in anchors, f"{topic}/{identity}: missing anchor"
            for key in ("claim", "source_id", "source_locator", "source_location", "rationale"):
                assert isinstance(claim[key], str) and claim[key].strip(), f"{identity}: empty {key}"
            assert claim["support"] in ("supported", "disputed", "insufficient")
            assert claim["source_status"] in ("published", "preprint", "corrected", "retracted", "unknown")
            assert isinstance(claim["access_limitations"], list) and claim["access_limitations"]
            assert claim["evidence_rows"], f"{identity}: no evidence rows"
            for number in claim["evidence_rows"]:
                assert type(number) is int and 1 <= number <= len(rows), f"{identity}: invalid row"
                row = rows[number - 1]
                assert row["source_id"] == claim["source_id"], f"{identity}: wrong row identity"
                assert row["source_locator"] == claim["source_locator"], f"{identity}: wrong locator"
                used_rows.add(number)
            if claim["source_id"] != "workflow":
                source = primary[claim["source_id"]]
                assert source["url"] == claim["source_locator"]
                assert source["http_status"] == 200
                assert source["title"] and source["authors"] and source["date"]
                assert source["doi"] or source["arxiv_id"]
                assert source["short_excerpt"] and source["excerpt_location"]
        assert used_rows == set(range(1, len(rows) + 1)), f"{topic}: unaudited evidence row"
        for output in index["outputs"]:
            path = (directory / output["path"]).resolve()
            assert path.is_relative_to(directory.resolve()) and path.is_file()
            assert hashlib.sha256(path.read_bytes()).hexdigest() == output["sha256"], output["path"]
        # Numbered references must actually resolve to primary identities.
        for source_id, record in primary.items():
            assert f"**{source_id}**" in report and record["url"] in report
        print(f"{topic}: {len(audit['claims'])} claims; {len(rows)} rows; anchors/source identities/locators/hashes consistent")
    predictions = json.loads((root / "majorana/predictions.json").read_text())
    assert [p["slot"] for p in predictions] == [1, 2, 3, 4, 5]
    assert all(p["status"] in ("proposed", "unsupported") for p in predictions)
    print("Offline consistency passed; source semantics and scientific approval NOT assessed.")


if __name__ == "__main__":
    validate(Path(sys.argv[1]).resolve())
