#!/usr/bin/env python3
"""Read-only structural checks, not scientific validation or engine qualification."""
import argparse
import csv
from datetime import datetime
import json
import re
from pathlib import Path, PurePosixPath
import sys
from urllib.parse import urlsplit

IDS = ["majorana", "nickelates"]
SOURCES = ["openalex", "crossref", "semantic-scholar", "arxiv"]
OUTPUTS = ["commands.jsonl", "coverage-stats.log", "evidence.csv", "report.md", "provenance.json"]
COLUMNS = {
    "majorana": "claim_signature study_identifier version_status experimental_or_theoretical device_material field temperature measurement_conditions observed_result quote_location topological_interpretation trivial_alternative independent_replication uncertainty source_locator verification_facets".split(),
    "nickelates": "study_identifier version_status family_composition structure_phase film_or_bulk substrate sample_preparation pressure_value pressure_unit pressure_calibration transition_temperature_value transition_temperature_unit transition_definition measurement_method uncertainty sample_dataset_group replication_status mechanism_claim direct_observation quote_location source_locator".split(),
}
COMMON_REQUIREMENTS = set("contradictory_evidence_matrix signature_comparison uncertainty_assessment source_backed_agent_reported_draft corrections_retractions_version_links_or_gaps unknowns_require_reasons mechanical_provenance_not_scientific_approval".split())
REQUIREMENTS = {
    "majorana": COMMON_REQUIREMENTS | set("five_proposed_falsifiable_prediction_slots refuting_observations_and_source_rationale_or_unsupported_slots independent_group_vs_shared_device_dataset_replication demonstrated_vs_model_dependent_vs_open".split()),
    "nickelates": COMMON_REQUIREMENTS | set("no_pooling_incompatible_measurements original_units_and_explicit_conversions transition_definitions_preserved no_inferred_pressure_uncertainty_shielding sample_dataset_group_provenance mechanism_claim_vs_direct_observation".split()),
}


def require(condition, message):
    if not condition:
        raise ValueError(message)


def text(value):
    return isinstance(value, str) and bool(value.strip())


def strings(value, label, nonempty=True):
    require(isinstance(value, list) and (bool(value) or not nonempty)
            and all(text(v) for v in value), f"{label}: expected string array")
    return value


def safe_file(root, name):
    require(text(name), "path: expected nonempty relative path")
    path = PurePosixPath(name)
    require(not path.is_absolute() and ".." not in path.parts and "\\" not in name,
            f"unsafe path: {name}")
    resolved = (root / name).resolve()
    require(resolved.is_relative_to(root.resolve()), f"unsafe path escape: {name}")
    require(resolved.is_file(), f"missing file: {name}")
    return resolved


def read_json(path):
    return json.loads(path.read_text(encoding="utf-8"))


def check_cases(path):
    packet = object_value(read_json(path), "cases")
    non_gold(packet)
    require(type(packet.get("schema_version")) is int and packet["schema_version"] == 1,
            "cases.schema_version must be integer 1")
    require(packet.get("purpose") == "exploratory_not_gold", "cases.purpose must be exploratory_not_gold")
    cases = packet.get("cases")
    require(isinstance(cases, list) and all(isinstance(c, dict) for c in cases) and [c.get("id") for c in cases] == IDS,
            "cases: expected ordered unique IDs majorana, nickelates")
    for case, depth, count, cap in zip(cases, ["comprehensive", "standard"], [20, 6], [100, 50]):
        name = case["id"]
        require(text(case.get("question")), f"{name}.question is required")
        require(case.get("depth") == depth, f"{name}.depth must be {depth}")
        require(case.get("sources") == SOURCES, f"{name}.sources must list the four scholarly-fast sources")
        queries = strings(case.get("queries"), f"{name}.queries")
        require(len(queries) == count and len(set(queries)) == count, f"{name}: expected {count} distinct queries")
        require(case.get("max_search_calls") == cap and count * len(SOURCES) <= cap,
                f"{name}: inconsistent search cap")
        require(case.get("limit") == 20, f"{name}.limit must be 20")
        outputs = OUTPUTS + (["predictions.json"] if name == "majorana" else [])
        require(case.get("required_outputs") == outputs, f"{name}: incorrect required_outputs")
        require(case.get("evidence_columns") == COLUMNS[name], f"{name}: missing evidence columns")
        requirements = strings(case.get("evidence_requirements"), f"{name}.evidence_requirements")
        require(REQUIREMENTS[name] <= set(requirements),
                f"{name}.evidence_requirements: missing {', '.join(sorted(REQUIREMENTS[name] - set(requirements)))}")
    return cases


def object_value(value, label):
    require(isinstance(value, dict), f"{label}: expected object")
    return value


def array(value, label):
    require(isinstance(value, list), f"{label}: expected array")
    return value


def non_gold(value):
    if isinstance(value, dict):
        for key, item in value.items():
            if key in {"gold", "gold_set", "is_gold", "ground_truth"}:
                require(item is False or item is None, f"{key}: gold claims are forbidden")
            non_gold(item)
    elif isinstance(value, list):
        for item in value:
            non_gold(item)


def host_only(value):
    if isinstance(value, dict):
        for key, item in value.items():
            require(not (key == "provenance_level" and item == "engine_verified"),
                    "host synthesis cannot declare engine_verified")
            host_only(item)
    elif isinstance(value, list):
        for item in value:
            host_only(item)


def timestamp(value, label):
    require(text(value), f"{label}: expected UTC timestamp")
    parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    require(parsed.utcoffset() is not None and parsed.utcoffset().total_seconds() == 0,
            f"{label}: expected UTC timestamp")
    return parsed


def digest(value, label):
    require(isinstance(value, str) and re.fullmatch(r"[0-9a-f]{64}", value),
            f"{label}: expected lowercase SHA-256")


def locator(value, root):
    require(text(value), "source_locator: required")
    if value.startswith("artifact:"):
        path, separator, location = value[9:].partition("#")
        require(separator and text(location), "source_locator: artifact needs #location")
        safe_file(root, path)
        return
    url = urlsplit(value)
    valid = (url.scheme in {"https", "http"} and bool(url.hostname)
             and not url.username and not any(c.isspace() for c in value))
    valid = valid or bool(re.fullmatch(r"doi:10\.\d{4,9}/\S+", value))
    valid = valid or bool(re.fullmatch(r"arxiv:(?:\d{4}\.\d{4,5}|[a-z-]+/\d{7})(?:v\d+)?", value))
    require(valid, f"source_locator: malformed {value!r}")


def command_records(root, topic):
    commands = []
    ids = set()
    for line_number, line in enumerate(safe_file(root, topic + "/commands.jsonl").read_text().splitlines(), 1):
        command = object_value(json.loads(line), f"{topic}.commands line {line_number}")
        identity = command.get("id")
        require(text(identity) and re.fullmatch(r"[A-Za-z0-9_-]+", identity) and identity not in ids,
                f"{topic}.commands: missing/duplicate/invalid id")
        ids.add(identity)
        require(command.get("stage") in {"preflight", "search", "coverage", "brief", "paper", "citations", "provenance_validate"},
                f"{topic}.{identity}: invalid stage")
        strings(command.get("argv"), f"{topic}.{identity}.argv")
        start = timestamp(command.get("start"), f"{topic}.{identity}.start")
        end = timestamp(command.get("end"), f"{topic}.{identity}.end")
        require(end >= start, f"{topic}.{identity}: end before start")
        require(type(command.get("exit_code")) is int, f"{topic}.{identity}: exit_code required")
        for stream in ["stdout", "stderr"]:
            safe_file(root, command.get(stream))
        commands.append(command)
    require(commands, f"{topic}: commands.jsonl must record attempted commands")
    validators = [c for c in commands if c["stage"] == "provenance_validate"
                  and any(c["argv"][i:i + 2] == ["provenance", "validate"]
                          for i in range(len(c["argv"]) - 1))]
    require(validators, f"{topic}: record actual provenance validate result (including failure)")
    return commands


def check_provenance(root, case):
    topic = case["id"]
    directory = root / topic
    record = object_value(read_json(safe_file(root, topic + "/provenance.json")), f"{topic}.provenance")
    host_only(record)
    require(record.get("schema_version") == "2", f"{topic}.provenance: schema_version must be string 2")
    require(record.get("provenance_level") in {"agent_reported", "unverified"},
            f"{topic}.provenance: host index must be agent_reported or unverified")
    require(text(record.get("question")) and record.get("depth") == case["depth"],
            f"{topic}.provenance: question/depth required")
    strings(record.get("sources"), f"{topic}.provenance.sources", nonempty=False)
    strings(record.get("errors"), f"{topic}.provenance.errors", nonempty=False)
    timestamp(record.get("timestamp"), f"{topic}.provenance.timestamp")
    version = object_value(record.get("rforge_version"), f"{topic}.provenance.rforge_version")
    require(all(text(version.get(key)) for key in ["version", "commit", "date"]),
            f"{topic}.provenance.rforge_version: version/commit/date required")
    for entry in array(record.get("entries"), f"{topic}.provenance.entries"):
        require(object_value(entry, "entry").get("provenance_level") in {"agent_reported", "unverified"},
                f"{topic}: manual entries must be agent_reported or unverified")
    indexed = set()
    for output in array(record.get("outputs"), f"{topic}.provenance.outputs"):
        output = object_value(output, "output")
        safe_file(directory, output.get("path"))
        digest(output.get("sha256"), f"{topic}.provenance.output.sha256")
        indexed.add(output["path"])
    required = {"report.md", "evidence.csv", "coverage-stats.log"}
    if topic == "majorana":
        required.add("predictions.json")
    required.update(name for name in ["manifest.json", "batch-state.json", "results.jsonl", "queries.txt"]
                    if (directory / name).exists())
    require(required <= indexed, f"{topic}.provenance: unindexed outputs {', '.join(sorted(required - indexed))}")
    runs = strings(record.get("runs"), f"{topic}.provenance.runs", nonempty=False)
    for run_id in runs:
        require(re.fullmatch(r"[A-Za-z0-9_-]+", run_id), f"{topic}: invalid run ID")
        safe_file(root, f"{topic}/provenance/runs/{run_id}.json")
    # Only the real engine validator checks receipt/event linkage and checksums.
    return runs


def check_matrix(root, case):
    name = case["id"] + "/evidence.csv"
    with safe_file(root, name).open(encoding="utf-8", newline="") as stream:
        reader = csv.DictReader(stream, strict=True)
        headers = reader.fieldnames or []
        missing = set(case["evidence_columns"]) - set(headers)
        require(not missing, f"{name}: missing headers {', '.join(sorted(missing))}")
        require(len(set(headers)) == len(headers), f"{name}: duplicate headers")
        for number, row in enumerate(reader, 2):
            require(None not in row and all(text(v) for v in row.values()),
                    f"{name}:{number}: missing/extra cells; use unknown: reason, not blanks")
            locator(row["source_locator"], root)
            if "verification_facets" in row:
                require("engine_verified" not in row["verification_facets"],
                        f"{name}:{number}: host facets cannot declare engine_verified")


def check_predictions(root):
    predictions = read_json(safe_file(root, "majorana/predictions.json"))
    require(isinstance(predictions, list) and len(predictions) == 5
            and all(isinstance(p, dict) for p in predictions)
            and [p.get("slot") for p in predictions] == [1, 2, 3, 4, 5],
            "majorana: expected five prediction slots ordered 1..5")
    host_only(predictions)
    for prediction in predictions:
        if prediction.get("status") == "unsupported":
            require(text(prediction.get("reason")), "prediction: unsupported slot needs reason")
        else:
            require(prediction.get("status") == "proposed", "prediction: status must be proposed or unsupported")
            require(all(text(prediction.get(key)) for key in ["prediction", "refuting_observation", "source_rationale"]),
                    "prediction: proposal needs prediction, refuting_observation and source_rationale")
            for source in strings(prediction.get("source_locators"), "prediction.source_locators"):
                locator(source, root)


def check_findings(root, command_ids):
    findings = array(read_json(safe_file(root, "findings.json")), "findings")
    host_only(findings)
    seen = set()
    for finding in findings:
        finding = object_value(finding, "finding")
        identity = finding.get("id")
        require(text(identity) and re.fullmatch(r"[A-Za-z0-9_-]+", identity) and identity not in seen,
                "finding: missing/duplicate/invalid ID")
        seen.add(identity)
        topics = strings(finding.get("topics"), f"{identity}.topics")
        require(set(topics) <= set(IDS), f"{identity}: unknown topic")
        require(finding.get("category") in {"engine_bug_candidate", "agent_error", "upstream_failure", "usability_gap", "scientific_gap"},
                f"{identity}: invalid category")
        require(finding.get("severity") in {"low", "medium", "high", "critical"}, f"{identity}: invalid severity")
        require(all(text(finding.get(key)) for key in ["observed", "expected", "proposed_next_action"]),
                f"{identity}: observed/expected/proposed_next_action required")
        strings(finding.get("reproduction_steps"), f"{identity}.reproduction_steps")
        require(finding.get("reproducibility") in {"unconfirmed", "offline_reproduced"},
                f"{identity}: reproducibility must be unconfirmed or offline_reproduced, never confirmed")
        for ref in strings(finding.get("command_refs"), f"{identity}.command_refs"):
            require(ref in command_ids and ref.split("/")[0] in topics,
                    f"{identity}: dangling/mismatched command reference {ref}")
        for ref in strings(finding.get("artifact_refs"), f"{identity}.artifact_refs"):
            safe_file(root, ref)
        if finding["category"] == "engine_bug_candidate" and finding["reproducibility"] == "offline_reproduced":
            for ref in strings(finding.get("offline_reproduction_artifacts"), f"{identity}.offline_reproduction_artifacts"):
                safe_file(root, ref)


def check_run(root, cases):
    run = object_value(read_json(safe_file(root, "run.json")), "run")
    non_gold(run)
    host_only(run)
    require(run.get("purpose") == "exploratory_not_gold", "run.purpose must be exploratory_not_gold")
    require(text(run.get("campaign_id")) and re.fullmatch(r"[A-Za-z0-9_-]+", run["campaign_id"]),
            "run.campaign_id: expected safe identifier")
    timestamp(run.get("started_at"), "run.started_at")
    environment = object_value(run.get("environment"), "run.environment")
    require(all(text(environment.get(key)) for key in ["os", "agent", "model"]), "environment: os/agent/model required")
    binary = object_value(run.get("binary"), "run.binary")
    require(text(binary.get("path")), "binary.path required")
    digest(binary.get("sha256"), "binary.sha256")
    version = object_value(binary.get("version"), "binary.version")
    require(all(text(version.get(key)) for key in ["version", "commit", "date"]), "binary.version: version/commit/date required")
    git = object_value(run.get("git"), "run.git")
    require(isinstance(git.get("revision"), str) and re.fullmatch(r"[0-9a-f]{40,64}", git["revision"]), "git.revision required")
    require(type(git.get("dirty")) is bool, "git.dirty must be boolean")
    digest(git.get("diff_sha256"), "git.diff_sha256")
    topics = object_value(run.get("topics"), "run.topics")
    ceilings = object_value(run.get("authorized_ceilings"), "run.authorized_ceilings")
    require(set(topics) == set(IDS) and set(ceilings) == set(IDS), "run: expected both topics and authorized ceilings")
    resume_authorizations = object_value(run.get("owner_resume_authorization", {}),
                                         "run.owner_resume_authorization")
    require(set(resume_authorizations) <= {"majorana"},
            "owner_resume_authorization: only one additional Majorana resume is supported")
    if "majorana" in resume_authorizations:
        authorization = object_value(resume_authorizations["majorana"], "majorana resume authorization")
        require(type(authorization.get("additional_resumes")) is int
                and authorization["additional_resumes"] == 1 and text(authorization.get("authority")),
                "majorana resume authorization: require additional_resumes=1 and recorded owner authority")
    command_ids = set()
    partial = False
    for case in cases:
        topic = case["id"]
        state = object_value(topics[topic], f"{topic}.status")
        status = state.get("status")
        require(status in {"complete", "partial", "blocked", "failed"}, f"{topic}: invalid status")
        partial = partial or status != "complete"
        require(text(state.get("reason")), f"{topic}: status reason required")
        require(type(state.get("search_attempted")) is bool, f"{topic}.search_attempted must be boolean")
        require(state.get("stop_reason") in {"completed", "budget_exhausted", "coverage_saturated", "interrupted", "error_threshold", "not_started"},
                f"{topic}: invalid stop_reason")
        require(text(state.get("coverage")), f"{topic}.coverage required (unknown: reason allowed)")
        strings(state.get("gaps"), f"{topic}.gaps", nonempty=False)
        limits = object_value(ceilings[topic], f"{topic}.authorized_ceilings")
        maximums = {"search_calls": case["max_search_calls"], "citation_seeds": 3, "citation_depth": 1,
                    "citation_max_records": 30, "citation_max_api_calls": 6, "citation_retry_budget": 0,
                    "paper_resolutions": 5, "process_timeout_seconds": 120,
                    "search_resumes": 2 if topic in resume_authorizations else 1}
        for key, maximum in maximums.items():
            require(type(limits.get(key)) is int and 0 <= limits[key] <= maximum,
                    f"{topic}.{key}: authorized ceiling must be 0..{maximum}")
        usage = object_value(state.get("usage"), f"{topic}.usage")
        unknowns = object_value(state.get("usage_unknowns"), f"{topic}.usage_unknowns")
        for key in ["search_calls", "citation_seeds", "paper_resolutions", "search_resumes", "http_attempts"]:
            require(key in usage, f"{topic}.usage.{key} required")
            if usage[key] is None:
                require(text(unknowns.get(key)), f"{topic}.usage.{key}: unknown needs reason")
            else:
                require(type(usage[key]) is int and usage[key] >= 0
                        and (key == "http_attempts" or usage[key] <= limits[key]), f"{topic}.usage.{key}: exceeds ceiling or invalid")
        for output in case["required_outputs"]:
            safe_file(root, topic + "/" + output)
        commands = command_records(root, topic)
        require(state["search_attempted"] == any(c["stage"] == "search" for c in commands),
                f"{topic}.search_attempted must match search command evidence")
        command_ids.update(topic + "/" + c["id"] for c in commands)
        runs = check_provenance(root, case)
        check_matrix(root, case)
        engine_files = ["manifest.json", "batch-state.json", "results.jsonl", "provenance/events.jsonl"]
        missing_engine = [name for name in engine_files if not (root / topic / name).exists()]
        if missing_engine or not runs:
            require(status != "complete" and text(state.get("engine_absence_reason")),
                    f"{topic}: missing engine artifacts require non-complete status and engine_absence_reason")
            require(not any(c["exit_code"] == 0 and c["stage"] == "search" for c in commands)
                    and any(c["exit_code"] != 0 and c["stage"] in {"preflight", "search"} for c in commands),
                    f"{topic}: engine absence needs failed-before-search command evidence")
        for name in engine_files:
            if (root / topic / name).exists():
                safe_file(root, topic + "/" + name)
        if status == "complete":
            require(state["search_attempted"] and state["stop_reason"] not in {"not_started", "interrupted", "error_threshold"},
                    f"{topic}: complete status inconsistent with search/stop_reason")
    check_predictions(root)
    check_findings(root, command_ids)
    summary = safe_file(root, "summary.md").read_text(encoding="utf-8").lower()
    for marker in ["exploratory", "not gold", "verified failures", "unconfirmed opportunities", "gates", "unsupported outputs"]:
        require(marker in summary, f"summary.md: missing {marker}")
    if partial:
        require("status: partial" in summary, "summary.md: non-complete topics require Status: partial")
    print("Campaign status: partial" if partial else "Campaign status: complete")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--cases", type=Path, required=True)
    parser.add_argument("--run", type=Path)
    args = parser.parse_args()
    try:
        cases = check_cases(args.cases)
        if args.run:
            check_run(args.run, cases)
    except (ValueError, OSError, TypeError, KeyError, AttributeError, csv.Error, RuntimeError) as error:
        print(f"invalid: {error}", file=sys.stderr)
        return 1
    print("Structurally valid; scientific support and human approval not assessed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
