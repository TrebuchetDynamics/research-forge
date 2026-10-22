import unittest
from extraction import extract_answer


class ExtractionTests(unittest.TestCase):
    def test_schema_extraction_preserves_nonempty_identities_and_all_ranks(self):
        original = {'bibliography': [
            {'identifiers': [{'scheme': 'DOI', 'value': 'https://doi.org/10.1234/UPPER'}, {'scheme': 'CrossrefID', 'value': '10.1234/UPPER'}]},
            {'identifiers': [{'scheme': 'PMID', 'value': ''}, {'scheme': 'PMCID', 'value': ''}]},
            {'identifiers': []},
            {'identifiers': [{'scheme': 'PMID', 'value': '12345'}, {'scheme': 'PMCID', 'value': 'PMC12345'}]},
        ], 'claims': []}
        result, changes = extract_answer(original)
        self.assertEqual(result, {'bibliography': [
            {'identifiers': [{'scheme': 'doi', 'value': 'https://doi.org/10.1234/UPPER'}, {'scheme': 'doi', 'value': '10.1234/UPPER'}]},
            {'identifiers': []},
            {'identifiers': []},
            {'identifiers': [{'scheme': 'pmid', 'value': '12345'}, {'scheme': 'pmcid', 'value': 'PMC12345'}]},
        ], 'claims': []})
        self.assertEqual(changes, 4)
        self.assertEqual(original['bibliography'][0]['identifiers'][0]['scheme'], 'DOI')
        for scheme, value in [('invented', 'not-resolved'), ('CrossrefID', 'eric:EJ1296630')]:
            with self.subTest(scheme=scheme), self.assertRaises(ValueError):
                extract_answer({'bibliography': [{'identifiers': [{'scheme': scheme, 'value': value}]}], 'claims': []})


if __name__ == '__main__':
    unittest.main()
