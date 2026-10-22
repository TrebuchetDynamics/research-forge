"""External live-pilot custodian. Private artifacts must be outside the checkout.

No selection, unblinded scoring, SDK installation or global Pi configuration.
The trusted gateway stays outside each observation's bubblewrap filesystem.
"""
import argparse
import base64
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import math
import os
from pathlib import Path
import shutil
import socketserver
import subprocess
import threading
import time
import urllib.error
import urllib.request

from gateway import Gateway, GatewayError

HERE = Path(__file__).resolve().parent
PI_RELEASE = Path(os.environ.get("PI_MANAGED_INSTALL_ROOT", str(Path.home() / ".pi/agent/install"))) / "releases/1.3.0"
NATIVE_SOURCE = Path(os.environ.get("PI_SEARCH_HUB", str(Path.home() / ".pi/agent/git/github.com/TrebuchetDynamics/pi-toolset/extensions/search-hub/index.js")))


def write_json(path, value):
    with path.open("x") as f:
        json.dump(value, f, sort_keys=True, separators=(",", ":"), allow_nan=False)
        f.write("\n")


class UnixServer(socketserver.ThreadingMixIn, socketserver.UnixStreamServer):
    daemon_threads = True


def paper_summary(record):
    abstract = record.get("Abstract") or ""
    return {"title": record.get("Title", ""), "identifiers": record.get("Identifiers", {}), "year": record.get("Year"), "urls": record.get("URLs", []), "abstract": abstract[:400], "abstract_truncated": len(abstract) > 400}


def handler(gateway, condition, private, rforge, policy, gateway_port):
    lock = threading.RLock()
    batch_number = 0

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def respond(self, status, payload):
            body = json.dumps(payload).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def do_GET(self):
            try:
                provider, path = self.path.lstrip("/").split("/", 1)
                with lock:
                    body = gateway.request(provider, "/" + path)
                self.send_response(200)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
            except Exception as error:
                # Never expose a retryable status to connector retry/backoff.
                # The gateway preserved the actual upstream status and header.
                self.respond(400, {"error": str(error) if isinstance(error, GatewayError) else type(error).__name__})

        def do_POST(self):
            nonlocal batch_number
            try:
                n = int(self.headers.get("Content-Length", "0"))
                if not 0 < n <= 65536:
                    raise GatewayError("invalid_request_length")
                params = json.loads(self.rfile.read(n))
                if self.path == "/native" and condition == "without_rforge":
                    with lock:
                        body = gateway.native(params["url"], limit=params.get("limit", 5))
                    result = {"body": base64.b64encode(body).decode()}
                elif self.path == "/rforge" and condition == "with_rforge" and rforge:
                    with lock:
                        gateway.check()
                        gateway.leak(params["query"])
                        batch_number += 1
                        number = batch_number
                        out = private / ("batch-" + str(number))
                    env = {"PATH": os.environ["PATH"], "HOME": str(private), "RFORGE_SOURCE_REQUEST_DELAY": "0s", "RFORGE_SEMANTIC_SCHOLAR_REQUEST_DELAY": "0s", "RFORGE_SEMANTIC_SCHOLAR_MAX_RETRIES": "0"}
                    for source in policy["with_sources"]:
                        env["RFORGE_" + source.replace("-", "_").upper() + "_URL"] = f"http://127.0.0.1:{gateway_port}/{source}"
                    proc = subprocess.run([str(rforge), "search", "batch", "--query", params["query"], "--sources", ",".join(policy["with_sources"]), "--depth", "standard", "--limit", str(max(1, min(50, int(params.get("limit", 25))))), "--continue-on-error", "--out", str(out)], env=env, text=True, capture_output=True, timeout=max(1, gateway.wall_ms / 1000 - (time.monotonic() - gateway.start)))
                    (private / f"batch-{number}.stderr").write_text(proc.stderr)
                    records = [json.loads(line) for line in (out / "results.jsonl").read_text().splitlines()] if (out / "results.jsonl").exists() else []
                    with lock:
                        gateway.leak(json.dumps(records))
                    result = {"records": [paper_summary(record) for record in records], "exit_code": proc.returncode}
                    for name in ("manifest.json", "batch-state.json"):
                        if (out / name).exists():
                            result[name] = json.loads((out / name).read_text())
                    if (out / "failures.jsonl").exists():
                        result["failures"] = [json.loads(line) for line in (out / "failures.jsonl").read_text().splitlines()]
                else:
                    raise GatewayError("condition_tool_not_allowed")
                self.respond(200, result)
            except Exception as error:
                self.respond(503, {"error": str(error) if isinstance(error, GatewayError) else type(error).__name__})
    return Handler


def observe(input_data, condition, private, *, api_key, model_url="https://openrouter.ai/api/v1", gateway_port=None, reviews=None, rforge=None, transport=None):
    started = time.monotonic()
    deadline_ms = round(time.time() * 1000 + input_data["caps"]["wall_clock_ms"])
    private.mkdir(parents=True, exist_ok=True)
    work = private / "sandbox"
    work.mkdir()
    (work / "agent").mkdir()
    (work / "tmp").mkdir()
    socket_dir = private / "socket"
    socket_dir.mkdir()
    traces = private / "http"
    traces.mkdir()
    write_json(work / "input.json", input_data)
    write_json(work / "agent/auth.json", {"openrouter": {"type": "api_key", "key": api_key}})
    write_json(work / "agent/models.json", {"providers": {"openrouter": {"baseUrl": model_url, "api": "openai-completions", "models": [{"id": input_data["model"]["version"], "name": input_data["model"]["version"], "reasoning": False, "input": ["text"], "contextWindow": 200000, "maxTokens": 4096}]}}})
    shutil.copyfile(HERE / "pi-session.mjs", work / "run.mjs")
    caps = input_data["caps"]
    gateway = Gateway(input_data["date_cap"], reviews or [{"title": "Forbidden synthetic review", "identifiers": []}], max_http=caps["http_requests"], max_search=caps["search_calls"], wall_ms=caps["wall_clock_ms"], trace_dir=traces, transport=transport)
    gateway_port = gateway_port or int(os.environ["AUTOWORK_PORT_BASE"]) + 1
    Handler = handler(gateway, condition, private, rforge, input_data["provider_policy"], gateway_port)
    servers = [ThreadingHTTPServer(("127.0.0.1", gateway_port), Handler), UnixServer(str(socket_dir / "gateway.sock"), Handler)]
    threads = [threading.Thread(target=s.serve_forever, daemon=True) for s in servers]
    for thread in threads:
        thread.start()
    node = Path(shutil.which("node")).resolve()
    cmd = ["bwrap", "--die-with-parent", "--new-session", "--unshare-user", "--unshare-pid", "--unshare-ipc", "--unshare-uts", "--clearenv", "--ro-bind", "/usr", "/usr", "--ro-bind", "/lib", "/lib", "--ro-bind", "/lib64", "/lib64", "--ro-bind", "/etc/ssl", "/etc/ssl", "--ro-bind", "/etc/resolv.conf", "/etc/resolv.conf", "--ro-bind", "/etc/hosts", "/etc/hosts", "--proc", "/proc", "--dev", "/dev", "--ro-bind", str(PI_RELEASE), "/pi", "--ro-bind", str(node), "/node", "--ro-bind", str(NATIVE_SOURCE), "/native.mjs", "--bind", str(work), "/work", "--bind", str(socket_dir), "/gateway", "--setenv", "HOME", "/work", "--setenv", "TMPDIR", "/work/tmp", "--setenv", "PATH", "/usr/bin:/bin", "--setenv", "PI_PACKAGE", "/pi/node_modules/@earendil-works/pi-coding-agent", "--setenv", "PI_CODING_AGENT_DIR", "/work/agent", "--setenv", "PILOT_CONDITION", condition, "--setenv", "PILOT_DEADLINE_MS", str(deadline_ms), "--chdir", "/work", "/node", "/work/run.mjs"]
    try:
        proc = subprocess.run(cmd, text=True, capture_output=True, timeout=caps["wall_clock_ms"] / 1000 + 60)
        (private / "host.stderr").write_text(proc.stderr)
        (private / "host.stdout").write_text(proc.stdout)
        if proc.returncode:
            raise RuntimeError(f"Pi host infrastructure exit {proc.returncode}; inspect {private / 'host.stderr'}")
        result = json.loads(proc.stdout)
        if not result.get("complete_accounting"):
            raise RuntimeError("incomplete LM accounting; retain infrastructure attempt, never score estimated tokens")
        result["provider_events"] = gateway.events
        result["search_calls"], result["http_requests"] = gateway.search_calls, gateway.http_requests
        result["date_excluded"] = gateway.date_excluded
        result["reference_review_leak"] = gateway.reference_review_leak
        result["wall_clock_ms"] = round((time.monotonic() - started) * 1000)
        result["exhausted"] = result["exhausted"] or gateway.budget_exhausted or result["wall_clock_ms"] >= caps["wall_clock_ms"]
        if gateway.reference_review_leak:
            raise GatewayError("reference_review_leak")
        write_json(private / "observation.json", result)
        return result
    finally:
        # Credentials never belong in retained research artifacts.
        (work / "agent/auth.json").unlink(missing_ok=True)
        for server in servers:
            server.shutdown()
            server.server_close()
        for thread in threads:
            thread.join()


def openrouter_key():
    value = json.loads((Path.home() / ".pi/agent/auth.json").read_text())["openrouter"]["key"]
    if value.startswith("!"):
        raise RuntimeError("command-backed credentials must be resolved by the provider runtime")
    return value


def billed_cost(key, generation_ids, out, *, usage=None, model=None):
    receipts = []
    for generation in generation_ids:
        req = urllib.request.Request("https://openrouter.ai/api/v1/generation?id=" + generation, headers={"Authorization": "Bearer " + key})
        for attempt in range(20):
            try:
                with urllib.request.urlopen(req, timeout=30) as response:
                    data = json.load(response)["data"]
                break
            except urllib.error.HTTPError as error:
                # Generation billing is eventually indexed. This retries only
                # a read-only control-plane receipt, never a model/research call.
                if error.code != 404 or attempt == 19:
                    raise
                time.sleep(min(2 ** attempt, 15))
        # Store only billing/identity fields, never credentials or unrelated key usage.
        receipts.append({k: data.get(k) for k in ("id", "model", "provider_name", "total_cost", "tokens_prompt", "tokens_completion", "native_tokens_prompt", "native_tokens_completion")})
    write_json(out, receipts)
    if not receipts or any(not isinstance(r["total_cost"], (int, float)) or not math.isfinite(r["total_cost"]) or r["total_cost"] < 0 for r in receipts):
        raise RuntimeError("actual billed generation costs unavailable")
    if any(not isinstance(r[k], int) or isinstance(r[k], bool) or r[k] < 0 for r in receipts for k in ("native_tokens_prompt", "native_tokens_completion")):
        raise RuntimeError("complete billed token accounting unavailable")
    if usage is not None and (sum(r["native_tokens_prompt"] for r in receipts) != usage["input_tokens"] or sum(r["native_tokens_completion"] for r in receipts) != usage["output_tokens"]):
        raise RuntimeError("LM usage does not reconcile with actual billing receipts")
    if model is not None and any(r["model"] != model or r["provider_name"] != "Anthropic" for r in receipts):
        raise RuntimeError("billed model/provider differs from registration")
    return sum(r["total_cost"] for r in receipts)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--private", required=True, type=Path)
    parser.add_argument("--set", required=True, type=Path)
    parser.add_argument("--rforge", required=True, type=Path)
    parser.add_argument("--start", type=int, default=0)
    parser.add_argument("--count", type=int, default=2)
    args = parser.parse_args()
    protocol = json.loads((args.private / "protocol.json").read_text())
    schedule = json.loads((args.private / "schedule.json").read_text())
    metadata = json.loads((args.set / "set.json").read_text())
    reviews = [q["gold_source_review"] for q in metadata["questions"]]
    key = openrouter_key()
    for i in range(args.start, min(len(schedule), args.start + args.count)):
        slot = schedule[i]
        run_dir = args.private / f"run-{i:04d}"
        if run_dir.exists():
            raise RuntimeError("create-only run directory already exists; never selectively retry")
        input_data = json.loads((args.private / "inputs" / f"{i:04d}.json").read_text())
        result = observe(input_data, slot["condition"], run_dir, api_key=key, reviews=reviews, rforge=args.rforge)
        cost = billed_cost(key, result["generation_ids"], run_dir / "billing.json", usage=result, model=protocol["model"]["version"])
        usage = {k: result[k] for k in ("input_tokens", "output_tokens", "search_calls", "tool_calls", "http_requests", "wall_clock_ms")}
        usage["cost_usd"] = cost
        row = {"slot": slot, "model": protocol["model"], "caps": protocol["caps"], "run_id": f"pi-{i:04d}", "transcript": str(run_dir / "observation.json"), "workspace_artifacts": [str(run_dir)], "stop_reason": "budget_exhausted" if result["exhausted"] else ("error_threshold" if result["error"] else "completed"), "usage": usage, "gate": {"reference_review_leak": False, "date_cap_enforced": True, "complete_accounting": True, "budget_checked": True, "date_excluded": result["date_excluded"]}, "answer": result["answer"], "provider_events": result["provider_events"]}
        write_json(run_dir / "raw-run.json", row)
        print(json.dumps({"slot_index": i, "stop_reason": row["stop_reason"], "cost_usd": cost, "http_requests": result["http_requests"]}), flush=True)


if __name__ == "__main__":
    main()
