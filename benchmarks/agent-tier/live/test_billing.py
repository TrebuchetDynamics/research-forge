"""Billing receipt retrieval may lag a completed generation; never rerun it."""
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import urllib.error

from custodian import billed_cost


class BillingTests(unittest.TestCase):
    def test_waits_for_eventually_visible_actual_receipt(self):
        missing = urllib.error.HTTPError('https://openrouter.ai/api/v1/generation?id=gen-one', 404, 'not yet indexed', {}, None)
        response = io.BytesIO(json.dumps({'data': {'id': 'gen-one', 'model': 'anthropic/claude-sonnet-4.5', 'provider_name': 'Anthropic', 'total_cost': 0.015, 'native_tokens_prompt': 100, 'native_tokens_completion': 30}}).encode())
        with tempfile.TemporaryDirectory(dir=os.environ['TMPDIR']) as work:
            with patch('urllib.request.urlopen', side_effect=[missing, response]), patch('time.sleep'):
                cost = billed_cost('offline-key', ['gen-one'], Path(work) / 'billing.json')
            self.assertEqual(cost, 0.015)
            receipt = json.loads((Path(work) / 'billing.json').read_text())
            self.assertEqual(receipt[0]['id'], 'gen-one')
            self.assertEqual(receipt[0]['native_tokens_prompt'], 100)


if __name__ == '__main__':
    unittest.main()
