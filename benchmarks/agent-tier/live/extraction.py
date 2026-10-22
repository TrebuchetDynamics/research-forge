"""Blind, identity-preserving schema extraction, before scoring or label join.

Only scheme labels and empty identity placeholders are projected into the
scorer schema. Nonempty values, ranks, duplicates, claims and quotes are immutable.
No gold, relevance or metrics input.
CrossrefID is a DOI alias only for DOI-shaped values from this pilot's Crossref
connector; other library source-specific IDs must not be relabeled as DOI.
"""
import argparse
import copy
import datetime as dt
import hashlib
import json
from pathlib import Path

SCHEMES = {'doi', 'pmid', 'pmcid', 'arxiv', 'openalex', 'semantic_scholar', 'ads_bibcode', 'url'}


def extract_answer(answer):
    result = copy.deepcopy(answer)
    changes = 0
    for paper in result['bibliography']:
        # Empty aliases express no identity. Keep their bibliography rank even
        # when no aliases remain; never guess a replacement or drop the paper.
        paper['identifiers'] = [i for i in paper['identifiers'] if i['value'] != '']
        for identifier in paper['identifiers']:
            original = identifier['scheme']
            scheme = original.casefold()
            if scheme == 'crossrefid':
                if not identifier['value'].startswith('10.') or '/' not in identifier['value']:
                    raise ValueError('CrossrefID is not a DOI; no identity guessing permitted')
                scheme = 'doi'
            if scheme not in SCHEMES:
                raise ValueError('unknown identifier scheme; no identity guessing permitted')
            if scheme != original:
                changes += 1
                identifier['scheme'] = scheme
    return result, changes


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--private', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    source = (args.private / 'experiment.json').read_bytes()
    experiment = json.loads(source)
    changes = 0
    empty_placeholders = 0
    for row in experiment['runs']:
        original = row['answer']
        extracted, n = extract_answer(original)
        # Check the invariants independently of the normalization mapping.
        assert original['claims'] == extracted['claims']
        assert len(original['bibliography']) == len(extracted['bibliography'])
        for before, after in zip(original['bibliography'], extracted['bibliography']):
            assert [i['value'] for i in before['identifiers'] if i['value'] != ''] == [i['value'] for i in after['identifiers']]
            empty_placeholders += sum(i['value'] == '' for i in before['identifiers'])
        changes += n
        row['answer'] = extracted
    output = (json.dumps(experiment, sort_keys=True, separators=(',', ':'), allow_nan=False) + '\n').encode()
    args.out.mkdir(mode=0o700)
    for name in ('protocol.json', 'schedule.json'):
        with (args.out / name).open('xb') as f:
            f.write((args.private / name).read_bytes())
    with (args.out / 'experiment.json').open('xb') as f:
        f.write(output)
    receipt = {'version': 'identifier-schema-projection-v2', 'created_at': dt.datetime.now(dt.timezone.utc).isoformat(), 'rule': 'Case-fold supported scheme labels; DOI-shaped CrossrefID -> doi (Crossref connector alias); omit only empty-string identity placeholders, preserving their bibliography ranks. Never edit nonempty values or infer identities.', 'input': 'structured answers only; no gold, relevance labels or scores', 'source_experiment_file_sha256': hashlib.sha256(source).hexdigest(), 'extracted_experiment_file_sha256': hashlib.sha256(output).hexdigest(), 'scheme_labels_normalized': changes, 'empty_identity_placeholders_omitted': empty_placeholders, 'nonempty_identifier_values_changed': 0, 'bibliography_ranks_changed': 0, 'duplicates_removed': 0, 'claims_or_quotes_changed': 0, 'unknown_schemes_guessed': 0, 'original_artifacts_retained': True}
    with (args.out / 'extraction.json').open('x') as f:
        json.dump(receipt, f, sort_keys=True, indent=2)
        f.write('\n')
    print(json.dumps(receipt, sort_keys=True))


if __name__ == '__main__':
    main()
