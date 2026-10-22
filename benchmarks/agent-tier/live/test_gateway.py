"""Offline public gateway seam: no provider or model network calls."""
import json
import unittest

from gateway import Gateway, GatewayError


class GatewayTests(unittest.TestCase):
    def gateway(self, transport, **kwargs):
        return Gateway("2020-06-15", [{"title": "Forbidden review", "identifiers": [{"scheme": "doi", "value": "10.1234/review"}, {"scheme": "pmid", "value": "12345"}, {"scheme": "semantic_scholar", "value": "reviewhash"}, {"scheme": "openalex", "value": "W12345"}, {"scheme": "ads_bibcode", "value": "2020AbCd.12345"}]}], transport=transport, **kwargs)

    def test_raw_dates_are_checked_after_provider_filter(self):
        def transport(url):
            self.assertIn("until-pub-date%3A2020-06-15", url)
            return 200, {}, json.dumps({"message": {"items": [
                {"DOI": "10.1234/old", "issued": {"date-parts": [[2019]]}},
                {"DOI": "10.1234/future", "issued": {"date-parts": [[2021]]}},
                {"DOI": "10.1234/partial", "issued": {"date-parts": [[2020, 6]]}},
                {"DOI": "10.1234/unknown"},
            ]}}).encode()
        g = self.gateway(transport)
        body = g.request("crossref", "/works?query=energy")
        self.assertEqual([x["DOI"] for x in json.loads(body)["message"]["items"]], ["10.1234/old"])
        self.assertEqual(g.date_excluded, 3)

    def test_europepmc_filter_and_raw_dates(self):
        def transport(url):
            self.assertTrue(url.startswith("https://www.ebi.ac.uk/europepmc/webservices/rest/search?"))
            self.assertIn("FIRST_PDATE", url)
            self.assertIn("2020-06-15", url)
            return 200, {}, b'{"resultList":{"result":[{"id":"1","firstPublicationDate":"2019-01-01"},{"id":"2","firstPublicationDate":"2021-01-01"}]}}'
        g = self.gateway(transport)
        body = g.request("europepmc", "/webservices/rest/search?query=energy&format=json")
        self.assertEqual([r["id"] for r in json.loads(body)["resultList"]["result"]], ["1"])
        self.assertEqual(g.date_excluded, 1)

    def test_reference_review_invalidates_before_date_filter(self):
        cases = [
            b'{"DOI":"https://doi.org/10.1234/REVIEW"}',
            b'{"id":"12345","source":"MED"}',
            b'{"paperId":"reviewhash"}',
            b'{"id":"https://openalex.org/W12345"}',
            b'{"bibcode":"2020AbCd.12345"}',
            b'{"title":"Forbidden\\u0020review"}',
        ]
        for raw in cases:
            with self.subTest(raw=raw):
                g = self.gateway(lambda _: (200, {}, b'{"message":{"items":[' + raw + b']}}'))
                with self.assertRaisesRegex(GatewayError, "reference_review_leak"):
                    g.request("crossref", "/works?query=energy")
                self.assertTrue(g.reference_review_leak)
        g = self.gateway(lambda _: (200, {}, b'<a class="result__a" href="https://example.org/paper">Forbidden <b>review</b></a>'))
        with self.assertRaisesRegex(GatewayError, "reference_review_leak"):
            g.native("https://html.duckduckgo.com/html/?q=energy")
        g = self.gateway(lambda url: (200, {}, b'<a class="result__a" href="https://example.org/paper">Old paper</a>') if 'duckduckgo' in url else (302, {"location": "https://doi.org/10.1234/review"}, b""))
        with self.assertRaisesRegex(GatewayError, "reference_review_leak"):
            g.native("https://html.duckduckgo.com/html/?q=energy")

    def test_first_429_quarantines_across_queries_without_retry(self):
        reached = []
        def transport(url):
            reached.append(url)
            return (429, {"retry-after": "3600"}, b"limited") if "crossref" in url else (200, {}, b'{"data":[]}')
        g = self.gateway(transport)
        for query in ["first", "second"]:
            with self.assertRaises(GatewayError):
                g.request("crossref", "/works?query=" + query)
        self.assertEqual(json.loads(g.request("semantic-scholar", "/graph/v1/paper/search?query=energy")), {"data": []})
        self.assertEqual(len(reached), 2)
        self.assertEqual([e["action"] for e in g.events], ["failed", "skipped", "completed"])
        self.assertEqual(g.events[0]["retry_after"], "3600")
        self.assertEqual(g.http_requests, 2)

    def test_http_cap_prevents_network_not_hides_counters(self):
        reached = []
        g = self.gateway(lambda url: (reached.append(url) or 200, {}, b'{"message":{"items":[]}}'), max_http=1)
        g.request("crossref", "/works?query=first")
        with self.assertRaisesRegex(GatewayError, "budget_exhausted"):
            g.request("crossref", "/works?query=second")
        self.assertEqual(len(reached), 1)
        self.assertEqual(g.http_requests, 1)

    def test_native_search_dates_enriched_without_engine_or_gold(self):
        reached = []
        def transport(url):
            reached.append(url)
            if "duckduckgo" in url:
                self.assertIn("df=1900-01-01..2020-06-15", url)
                return 200, {}, b'<a class="result__a" href="https://example.org/paper">Old paper</a><a class="result__snippet">Result text</a><a class="result__a" href="https://example.org/unknown">Unknown paper</a>'
            if url.endswith("/paper"):
                return 200, {}, b'<html><script type="application/ld+json">{"@type":"ScholarlyArticle","url":"https://example.org/paper","datePublished":"2019-02-03"}</script></html>'
            return 200, {}, b'<html><script type="application/ld+json">{"@type":"ScholarlyArticle","url":"https://example.org/related","datePublished":"2010-01-01"}</script></html>'
        g = self.gateway(transport)
        body = g.native("https://html.duckduckgo.com/html/?q=energy").decode()
        self.assertIn("Old paper", body)
        self.assertNotIn("Unknown paper", body)
        self.assertEqual(g.date_excluded, 1)
        self.assertEqual(g.http_requests, 3)
        self.assertEqual(len(reached), 3)
        reached.clear()
        limited = self.gateway(transport)
        limited.native("https://html.duckduckgo.com/html/?q=energy", limit=1)
        self.assertEqual(len(reached), 2)
        self.assertEqual(limited.date_excluded, 0)


if __name__ == "__main__":
    unittest.main()
