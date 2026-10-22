"""Pi adapter public seam using a local SSE provider; no paid requests."""
import json
import os
from pathlib import Path
import tempfile
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer
import unittest

from custodian import observe


class SessionTests(unittest.TestCase):
    def test_sandboxed_session_retains_answer_and_measured_usage(self):
        class Provider(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_POST(self):
                payload = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                if payload["temperature"] != 0 or payload["top_p"] != 1:
                    self.send_error(400)
                    return
                chunks = [
                    {"id": "gen-offline", "choices": [{"index": 0, "delta": {"role": "assistant", "tool_calls": [{"index": 0, "id": "call-1", "type": "function", "function": {"name": "submit_answer", "arguments": '{"bibliography":[{"identifiers":[{"scheme":"doi","value":"10.1234/paper"}]}],"claims":[]}'}}]}, "finish_reason": None}]},
                    {"id": "gen-offline", "choices": [{"index": 0, "delta": {}, "finish_reason": "tool_calls"}], "usage": {"prompt_tokens": 100, "completion_tokens": 30, "total_tokens": 130, "prompt_tokens_details": {"cached_tokens": 20}, "cost": 0.0001}},
                ]
                if not server.include_usage:
                    chunks[-1].pop("usage")
                body = ''.join('data: ' + json.dumps(c) + '\n\n' for c in chunks) + 'data: [DONE]\n\n'
                self.send_response(200)
                self.send_header("Content-Type", "text/event-stream")
                self.send_header("Content-Length", str(len(body.encode())))
                self.end_headers()
                self.wfile.write(body.encode())

        port = int(os.environ["AUTOWORK_PORT_BASE"]) + 2
        server = HTTPServer(("127.0.0.1", port), Provider)
        server.include_usage = True
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            with tempfile.TemporaryDirectory(dir=os.environ["TMPDIR"]) as work:
                input_data = {"question_id": "synthetic", "question": "Synthetic research", "domain": "energy", "date_cap": "2020-06-15", "caps": {"tokens": 32000, "search_calls": 20, "tool_calls": 60, "http_requests": 100, "wall_clock_ms": 60000}, "system": "Use available tools.", "task": "Call submit_answer with an ordered bibliography.", "tool_instructions": "Use only the provided research tools.", "sampling": {"temperature": 0, "top_p": 1, "paired_seeds": False}, "model": {"provider": "openrouter", "version": "anthropic/claude-sonnet-4.5"}, "seed": None, "provider_policy": {"with_sources": ["crossref", "europepmc"], "continue_on_error": True, "rate_limit_action": "skip_for_run"}}
                result = observe(input_data, "without_rforge", Path(work), api_key="offline-key", model_url=f"http://127.0.0.1:{port}/v1", gateway_port=port + 1)
                self.assertEqual(result["answer"]["bibliography"], [{"identifiers": [{"scheme": "doi", "value": "10.1234/paper"}]}])
                self.assertEqual((result["input_tokens"], result["output_tokens"], result["tool_calls"]), (100, 30, 1))
                self.assertEqual(result["active_tools"], ["web_search", "submit_answer"])
                self.assertEqual(result["generation_ids"], ["gen-offline"])
                self.assertFalse(result["exhausted"])
                self.assertIsNone(result["error"])
                capped = dict(input_data, caps=dict(input_data["caps"], tokens=100))
                partial = observe(capped, "without_rforge", Path(work) / "budget-partial", api_key="offline-key", model_url=f"http://127.0.0.1:{port}/v1", gateway_port=port + 1)
                self.assertEqual(partial["answer"], result["answer"])
                self.assertTrue(partial["exhausted"])
                self.assertEqual(partial["input_tokens"] + partial["output_tokens"], 130)
                server.include_usage = False
                with self.assertRaisesRegex(RuntimeError, "incomplete LM accounting"):
                    observe(input_data, "without_rforge", Path(work) / "missing-usage", api_key="offline-key", model_url=f"http://127.0.0.1:{port}/v1", gateway_port=port + 1)
        finally:
            server.shutdown()
            thread.join()
            server.server_close()


if __name__ == "__main__":
    unittest.main()
