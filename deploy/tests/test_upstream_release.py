import importlib.util
import datetime as dt
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time
import unittest
import uuid

DEPLOY = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(DEPLOY))
spec = importlib.util.spec_from_file_location('upstream_release', DEPLOY / 'upstream_release.py')
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class UpstreamReleaseTests(unittest.TestCase):
    def test_reviewed_source_bytes_and_historical_migrations(self):
        candidate = {
            path.name: release.shared.migration_checksum(path.read_bytes())
            for path in (DEPLOY.parent / 'backend/migrations').glob('*.sql')
        }
        before = {name: value for name, value in candidate.items()
                  if name not in release.UPSTREAM_MIGRATIONS}
        self.assertEqual(len(before), 350)
        self.assertEqual(
            release.upstream_migration_allowlist(before, candidate),
            release.UPSTREAM_MIGRATIONS,
        )
        old = next(iter(before))
        for invalid in (
            {**candidate, old: 'changed'},
            {name: value for name, value in candidate.items() if name != old},
            {**candidate, '350_unreviewed.sql': 'extra'},
            {**candidate, '349_group_usage_rollup_retention_nonblocking.sql': 'changed'},
            before,
        ):
            with self.assertRaises(RuntimeError):
                release.upstream_migration_allowlist(before, invalid)

    def test_lower_numbered_upstream_additions_are_not_silently_skipped(self):
        before = {'348_existing_hk.sql': 'original'}
        candidate = {**before, **release.UPSTREAM_MIGRATIONS}
        self.assertEqual(
            release.upstream_migration_allowlist(before, candidate),
            release.UPSTREAM_MIGRATIONS,
        )
        for missing in release.UPSTREAM_MIGRATIONS:
            with self.assertRaises(RuntimeError):
                release.upstream_migration_allowlist(
                    before, {name: value for name, value in candidate.items() if name != missing})

    def test_ci_is_bound_to_both_successful_candidate_workflows(self):
        sha = 'a' * 40
        runs = [
            {'name': name, 'head_sha': sha, 'status': 'completed', 'conclusion': 'success'}
            for name in ('CI', 'Security Scan')
        ]
        release.validate_ci({'sha': sha, 'workflow_runs': runs}, sha)
        for invalid in (runs[:1], runs + [runs[0]],
                        [runs[0], {**runs[1], 'head_sha': 'b' * 40}],
                        [runs[0], {**runs[1], 'conclusion': 'failure'}]):
            with self.assertRaises(RuntimeError):
                release.validate_ci({'sha': sha, 'workflow_runs': invalid}, sha)

    def test_newly_visible_ids_do_not_weaken_existing_row_protection(self):
        at = dt.datetime.now(dt.timezone.utc)
        row = {'created_at': at.isoformat(), 'sha256': 'original'}
        before, after = {'102': row}, {'101': row, '102': row}
        result = release.shared.verify_history(before, after, 30, at, allow_newly_visible=True)
        self.assertEqual(result['unchanged_retained_rows'], 1)
        self.assertEqual(result['newly_visible_rows'], 1)
        for invalid in ({'101': row}, {'101': row, '102': {**row, 'sha256': 'changed'}}):
            with self.assertRaises(RuntimeError):
                release.shared.verify_history(before, invalid, 30, at, allow_newly_visible=True)
        with self.assertRaises(RuntimeError):
            release.shared.verify_history(before, after, 30, at)

    @unittest.skipUnless(os.environ.get('SUB2API_RELEASE_PG_REGRESSION') == '1' and shutil.which('docker'),
                         'opt-in isolated PostgreSQL commit-order regression')
    def test_two_connections_commit_sequence_ids_in_reverse_order(self):
        name = 'sub2api-history-review-' + uuid.uuid4().hex[:12]
        sql = ['docker', 'exec', '-i', name, 'psql', '-U', 'postgres', '-X', '-qAt',
               '-v', 'ON_ERROR_STOP=1']
        transaction = None
        subprocess.run(['docker', 'run', '--rm', '-d', '--name', name,
                        '-e', 'POSTGRES_PASSWORD=test-password', 'postgres:18.1-alpine3.23'],
                       check=True, capture_output=True)
        try:
            for _ in range(60):
                # The entrypoint's temporary initialization server only listens
                # on a Unix socket and shuts down before the final TCP server.
                if subprocess.run(['docker', 'exec', name, 'pg_isready', '-h', '127.0.0.1', '-U', 'postgres'],
                                  capture_output=True).returncode == 0:
                    break
                time.sleep(.5)
            else:
                self.fail('PostgreSQL TCP server did not become ready')
            def query(text):
                return subprocess.check_output(sql + ['-c', text], text=True).strip()
            query('CREATE TABLE bills(id BIGSERIAL PRIMARY KEY, amount NUMERIC NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now())')
            transaction = subprocess.Popen(sql, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                           stderr=subprocess.PIPE, text=True)
            transaction.stdin.write('BEGIN;\nINSERT INTO bills(amount) VALUES(1) RETURNING id;\n')
            transaction.stdin.flush()
            self.assertEqual(transaction.stdout.readline().strip(), '1')
            self.assertEqual(query('INSERT INTO bills(amount) VALUES(2) RETURNING id'), '2')
            def snapshot():
                rows = json.loads(query("SELECT coalesce(json_agg(b), '[]') FROM bills b"))
                return {str(row['id']): {'created_at': row['created_at'],
                                        'sha256': str(row['amount'])} for row in rows}
            before = snapshot()
            transaction.stdin.write('COMMIT;\n')
            transaction.stdin.flush()
            transaction.communicate(timeout=20)
            self.assertEqual(transaction.returncode, 0)
            after = snapshot()
            self.assertEqual(set(before), {'2'})
            self.assertEqual(set(after), {'1', '2'})
            result = release.shared.verify_history(before, after, 30, release.shared.now(),
                                                   allow_newly_visible=True)
            self.assertEqual((result['unchanged_retained_rows'], result['newly_visible_rows']), (1, 1))
            query('UPDATE bills SET amount=3 WHERE id=2')
            with self.assertRaises(RuntimeError):
                release.shared.verify_history(before, snapshot(), 30, release.shared.now(),
                                               allow_newly_visible=True)
        finally:
            if transaction is not None and transaction.poll() is None:
                transaction.kill()
                transaction.communicate()
            subprocess.run(['docker', 'rm', '-f', name], check=True, capture_output=True)


if __name__ == '__main__':
    unittest.main()
