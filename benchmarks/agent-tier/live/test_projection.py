import unittest
from custodian import paper_summary


class ProjectionTests(unittest.TestCase):
    def test_tool_metadata_is_bounded_without_changing_identifiers(self):
        result = paper_summary({'Title': 'Paper', 'Identifiers': {'DOI': '10.1234/paper'}, 'Year': 2019, 'URLs': ['https://example.org/paper'], 'Abstract': 'x' * 401, 'SourceRefs': ['private/run/path']})
        self.assertEqual(result, {'title': 'Paper', 'identifiers': {'DOI': '10.1234/paper'}, 'year': 2019, 'urls': ['https://example.org/paper'], 'abstract': 'x' * 400, 'abstract_truncated': True})


if __name__ == '__main__':
    unittest.main()
