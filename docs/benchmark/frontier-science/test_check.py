"""Synthetic-only offline fixtures; no scientific results or copyrighted source text."""
import csv
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve().parent


class CheckTest(unittest.TestCase):
    def test_case_packet(self):
        self.assertTrue((HERE / "cases.json").is_file(), "missing cases.json packet")
        packet = json.loads((HERE / "cases.json").read_text())
        self.assertEqual(packet["schema_version"], 1)
        self.assertEqual(packet["purpose"], "exploratory_not_gold")
        self.assertEqual([c["id"] for c in packet["cases"]], ["majorana", "nickelates"])
        for case, count, depth, cap in zip(packet["cases"], [20, 6],
                                           ["comprehensive", "standard"], [100, 50]):
            self.assertTrue(case["question"].endswith("?"))
            self.assertEqual(case["depth"], depth)
            self.assertEqual(case["sources"], ["openalex", "crossref", "semantic-scholar", "arxiv"])
            self.assertEqual(len(case["queries"]), count)
            self.assertEqual(case["max_search_calls"], cap)
            self.assertLessEqual(count * 4, cap)
            self.assertEqual(case["limit"], 20)
            self.assertIn("evidence.csv", case["required_outputs"])
            self.assertIn("source_locator", case["evidence_columns"])
            self.assertTrue(case["evidence_requirements"])
        result = subprocess.run([sys.executable, str(HERE / "check.py"), "--cases",
                                 str(HERE / "cases.json")], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        for field, value, message in [
            ("schema_version", True, "schema_version"),
            ("purpose", "gold", "purpose"),
            ("gold_set", True, "gold"),
            ("cases", list(reversed(packet["cases"])), "ordered unique"),
            ("cases", [dict(packet["cases"][0], evidence_requirements=["placeholder"]),
                       packet["cases"][1]], "evidence_requirements"),
        ]:
            with self.subTest(packet=field), tempfile.TemporaryDirectory(dir=os.environ.get("TMPDIR")) as temp:
                path = Path(temp) / "cases.json"
                path.write_text(json.dumps(dict(packet, **{field: value})))
                invalid = subprocess.run([sys.executable, str(HERE / "check.py"), "--cases", str(path)],
                                         capture_output=True, text=True)
                self.assertEqual(invalid.returncode, 1, invalid.stderr)
                self.assertIn(message, invalid.stderr)

    def invoke(self, root, cases=None):
        return subprocess.run([sys.executable, str(HERE / "check.py"), "--cases",
                               str(cases or HERE / "cases.json"), "--run", str(root)],
                              capture_output=True, text=True)

    def fixture(self, root, partial=False):
        """Fabricated records and receipts for shape checks ONLY, never engine evidence."""
        def save(path, value):
            (root / path).write_text(json.dumps(value), encoding="utf-8")

        (root / "summary.md").write_text(
            "# Exploratory, not gold\nStatus: " + ("partial" if partial else "complete") +
            "\n## Verified failures\nNone.\n## Unconfirmed opportunities\nNone.\n"
            "## Gates\nNo approvals.\n## Unsupported outputs\nSynthetic only.\n")
        packet = json.loads((HERE / "cases.json").read_text())
        run = {"campaign_id": "synthetic-only", "purpose": "exploratory_not_gold",
               "started_at": "2026-01-01T00:00:00Z",
               "environment": {"os": "synthetic", "agent": "fixture", "model": "fixture"},
               "binary": {"path": "rforge", "sha256": "0" * 64,
                          "version": {"version": "dev", "commit": "unknown", "date": "unknown"}},
               "git": {"revision": "1" * 40, "dirty": False, "diff_sha256": "0" * 64},
               "authorized_ceilings": {}, "topics": {}}
        for case in packet["cases"]:
            topic = case["id"]
            directory = root / topic
            directory.mkdir()
            (directory / "stdout.log").write_text("Synthetic only; not a real command result.\n")
            (directory / "stderr.log").write_text("")
            command = {"id": "search-1", "stage": "search", "argv": ["rforge", "search", "batch"],
                       "start": "2026-01-01T00:00:00Z", "end": "2026-01-01T00:00:01Z",
                       "exit_code": 1 if partial else 0, "stdout": topic + "/stdout.log",
                       "stderr": topic + "/stderr.log"}
            validation = dict(command, id="validate-1", stage="provenance_validate",
                              argv=["rforge", "provenance", "validate", topic + "/provenance.json"],
                              exit_code=1 if partial else 0)
            (directory / "commands.jsonl").write_text(
                json.dumps(command) + "\n" + json.dumps(validation) + "\n")
            (directory / "coverage-stats.log").write_text("unknown: synthetic fixture\n")
            (directory / "report.md").write_text("agent_reported draft; synthetic only.\n")
            with (directory / "evidence.csv").open("w", newline="") as stream:
                writer = csv.DictWriter(stream, fieldnames=case["evidence_columns"])
                writer.writeheader()
                if not partial:
                    row = {key: "unknown: synthetic fixture" for key in case["evidence_columns"]}
                    row["source_locator"] = "https://example.invalid/synthetic#abstract"
                    row["quote_location"] = "Synthetic quotation; abstract"
                    if topic == "majorana":
                        row["verification_facets"] = "agent_reported; mechanical facets unknown"
                    writer.writerow(row)
            run["authorized_ceilings"][topic] = {
                "search_calls": case["max_search_calls"], "citation_seeds": 3,
                "citation_depth": 1, "citation_max_records": 30, "citation_max_api_calls": 6,
                "citation_retry_budget": 0, "paper_resolutions": 5,
                "process_timeout_seconds": 120, "search_resumes": 1}
            run["topics"][topic] = {
                "status": "partial" if partial else "complete", "reason": "Synthetic only",
                "search_attempted": True, "engine_absence_reason": "Failed before search artifacts" if partial else "",
                "usage": {"search_calls": 0 if partial else 1, "citation_seeds": 0,
                          "paper_resolutions": 0, "search_resumes": 0, "http_attempts": None},
                "usage_unknowns": {"http_attempts": "Not observable in synthetic fixture"},
                "stop_reason": "error_threshold" if partial else "completed",
                "coverage": "unknown: synthetic fixture", "gaps": ["No scientific evidence"]}
            if not partial:
                for name in ["manifest.json", "batch-state.json", "results.jsonl"]:
                    (directory / name).write_text("{}\n")
                (directory / "provenance/runs").mkdir(parents=True)
                (directory / "provenance/runs/synthetic.json").write_text("{}")
                (directory / "provenance/events.jsonl").write_text("{}\n")
            provenance = {"schema_version": "2", "provenance_level": "agent_reported",
                          "question": case["question"], "depth": case["depth"], "sources": case["sources"],
                          "timestamp": "2026-01-01T00:00:00Z", "rforge_version": run["binary"]["version"],
                          "runs": [] if partial else ["synthetic"],
                          "entries": [{"provenance_level": "agent_reported", "description": "Synthetic synthesis"}],
                          "outputs": [{"path": "report.md", "sha256": hashlib.sha256(
                              (directory / "report.md").read_bytes()).hexdigest()}], "errors": []}
            save(topic + "/provenance.json", provenance)
        save("majorana/predictions.json", [
            {"slot": i, "status": "unsupported", "reason": "Synthetic only: no scientific proposal"}
            for i in range(1, 6)])
        save("run.json", run)
        save("findings.json", [] if partial else [
            {"id": "F-1", "topics": ["majorana"], "category": "usability_gap", "severity": "low",
             "observed": "Synthetic observation", "expected": "Synthetic expectation",
             "command_refs": ["majorana/search-1"], "artifact_refs": ["majorana/report.md"],
             "reproduction_steps": ["Synthetic only; no live reproduction"],
             "reproducibility": "unconfirmed", "proposed_next_action": "Review synthetic example"}])
        if not partial:
            predictions = json.loads((root / "majorana/predictions.json").read_text())
            predictions[0] = {"slot": 1, "status": "proposed", "prediction": "Synthetic proposal, not physics",
                              "refuting_observation": "Synthetic refuting observation",
                              "source_rationale": "Synthetic source rationale",
                              "source_locators": ["artifact:majorana/report.md#synthetic"]}
            save("majorana/predictions.json", predictions)
        for case in packet["cases"]:
            directory = root / case["id"]
            provenance = json.loads((directory / "provenance.json").read_text())
            names = ["report.md", "evidence.csv", "coverage-stats.log", "predictions.json",
                     "manifest.json", "batch-state.json", "results.jsonl", "queries.txt"]
            provenance["outputs"] = [
                {"path": name, "sha256": hashlib.sha256((directory / name).read_bytes()).hexdigest()}
                for name in names if (directory / name).exists()]
            save(case["id"] + "/provenance.json", provenance)

    def test_complete_run(self):
        with tempfile.TemporaryDirectory(dir=os.environ.get("TMPDIR")) as temp:
            root = Path(temp)
            self.fixture(root)
            before = {p.relative_to(root): (p.read_bytes(), p.stat().st_mtime_ns)
                      for p in root.rglob("*") if p.is_file()}
            result = self.invoke(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("scientific support", result.stdout)
            self.assertEqual(before, {p.relative_to(root): (p.read_bytes(), p.stat().st_mtime_ns)
                                      for p in root.rglob("*") if p.is_file()})

    def test_partial_run(self):
        with tempfile.TemporaryDirectory(dir=os.environ.get("TMPDIR")) as temp:
            root = Path(temp)
            self.fixture(root, partial=True)
            result = self.invoke(root)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("partial", result.stdout)

    def test_rejects_unsafe_or_overclaimed_run(self):
        # Each mutation breaks one public artifact-contract requirement.
        mutations = [
            ("missing output", "missing file", lambda r: (r / "majorana/stdout.log").unlink()),
            ("traversal", "unsafe path", lambda r: self.replace_json(
                r / "majorana/provenance.json", "outputs", [{"path": "../summary.md", "sha256": "0" * 64}])),
            ("symlink", "unsafe path", lambda r: self.escape_link(r)),
            ("host provenance", "engine_verified", lambda r: self.replace_json(
                r / "majorana/provenance.json", "provenance_level", "engine_verified")),
            ("missing condition", "pressure_calibration", lambda r: (r / "nickelates/evidence.csv").write_text(
                (r / "nickelates/evidence.csv").read_text().replace("pressure_calibration", "omitted"))),
            ("dangling finding", "command reference", lambda r: (r / "findings.json").write_text(json.dumps([
                {"id": "F-1", "topics": ["majorana"], "category": "engine_bug_candidate", "severity": "high",
                 "observed": "Synthetic", "expected": "Synthetic", "command_refs": ["majorana/missing"],
                 "artifact_refs": ["majorana/report.md"], "reproduction_steps": ["Synthetic"],
                 "reproducibility": "unconfirmed", "proposed_next_action": "Offline repro"}]))),
            ("missing prediction", "five prediction slots", lambda r: (r / "majorana/predictions.json").write_text("[]")),
            ("bad locator", "source_locator", lambda r: (r / "nickelates/evidence.csv").write_text(
                (r / "nickelates/evidence.csv").read_text().replace("https://example.invalid/synthetic#abstract", "not-a-locator"))),
            ("bad status", "status", lambda r: self.replace_topic(r, "status", "verified")),
            ("over budget", "search_calls", lambda r: self.replace_topic(
                r, "usage", {"search_calls": 101, "citation_seeds": 0, "paper_resolutions": 0,
                             "search_resumes": 0, "http_attempts": None})),
            ("unexplained absence", "engine_absence_reason", lambda r: self.replace_topic(r, "engine_absence_reason", "")),
            ("waived successful search", "failed-before-search", self.waive_success),
            ("missing validator", "provenance validate", lambda r: (r / "majorana/commands.jsonl").write_text(
                (r / "majorana/commands.jsonl").read_text().splitlines()[0] + "\n")),
            ("missing metadata", "environment", lambda r: self.replace_json(r / "run.json", "environment", {})),
            ("nested overclaim", "engine_verified", lambda r: self.replace_json(
                r / "majorana/provenance.json", "entries", [{"provenance_level": "engine_verified"}])),
            ("malformed JSON", "invalid:", lambda r: (r / "run.json").write_text("{")),
            ("unindexed host outputs", "unindexed outputs", lambda r: self.replace_json(
                r / "majorana/provenance.json", "outputs", [])),
            ("denied search attempt", "search_attempted", self.deny_search),
        ]
        for name, error, mutate in mutations:
            with self.subTest(name=name), tempfile.TemporaryDirectory(dir=os.environ.get("TMPDIR")) as temp:
                root = Path(temp)
                self.fixture(root, partial=name == "unexplained absence")
                mutate(root)
                result = self.invoke(root)
                self.assertEqual(result.returncode, 1, result.stderr)
                self.assertIn(error, result.stderr)
                self.assertNotIn("Traceback", result.stderr)

    def replace_json(self, path, key, value):
        data = json.loads(path.read_text())
        data[key] = value
        path.write_text(json.dumps(data))

    def replace_topic(self, root, key, value):
        path = root / "run.json"
        data = json.loads(path.read_text())
        data["topics"]["majorana"][key] = value
        path.write_text(json.dumps(data))

    def deny_search(self, root):
        self.replace_topic(root, "status", "partial")
        self.replace_topic(root, "search_attempted", False)
        summary = root / "summary.md"
        summary.write_text(summary.read_text().replace("Status: complete", "Status: partial"))

    def waive_success(self, root):
        (root / "majorana/manifest.json").unlink()
        path = root / "majorana/provenance.json"
        provenance = json.loads(path.read_text())
        provenance["outputs"] = [item for item in provenance["outputs"] if item["path"] != "manifest.json"]
        path.write_text(json.dumps(provenance))
        self.replace_topic(root, "status", "partial")
        self.replace_topic(root, "engine_absence_reason", "Not actually failed before search")
        summary = root / "summary.md"
        summary.write_text(summary.read_text().replace("Status: complete", "Status: partial"))
        commands = root / "majorana/commands.jsonl"
        failed = json.loads(commands.read_text().splitlines()[0])
        failed.update(id="preflight-fail", stage="preflight", exit_code=1)
        with commands.open("a") as stream:
            stream.write(json.dumps(failed) + "\n")

    def escape_link(self, root):
        path = root / "majorana/stdout.log"
        path.unlink()
        path.symlink_to(HERE / "cases.json")


if __name__ == "__main__":
    unittest.main()
