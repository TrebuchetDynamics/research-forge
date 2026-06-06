# Connector testing notes

Preferred fixtures:

- small OpenAlex JSON work response;
- small arXiv Atom response;
- small Crossref JSON response;
- small Unpaywall JSON response;
- duplicate records with DOI case differences, arXiv versions, and fuzzy title variants.

Assertions to include:

- outgoing URL/query parameters are correct;
- API response maps to normalized title, authors, year, DOI/arXiv ID, venue, abstract, and source IDs;
- raw source payload or source reference is retained;
- provenance includes source name, query, timestamp, parameters, and cache key;
- dedupe never deletes provenance from merged records.
