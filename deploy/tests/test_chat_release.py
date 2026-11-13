import datetime as dt
import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('chat_release', Path(__file__).resolve().parents[1] / 'chat_release.py')
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class ChatReleaseTests(unittest.TestCase):
    def test_chat_migrations_are_append_only_and_exact(self):
        old = {'001_initial.sql': 'a', '002_previous.sql': 'b'}
        candidate = {**old, '003_user_chat.sql': 'c'}
        allowed = release.migration_allowlist(old, candidate)
        self.assertEqual(allowed, {'003_user_chat.sql': 'c'})
        self.assertEqual(release.verify_migrations(old, candidate, allowed), allowed)
        for invalid in ({**candidate, '004_other.sql': 'd'}, {**candidate, '001_initial.sql': 'changed'},
                        {**old, '002_user_chat.sql': 'c'}, old):
            with self.assertRaises(RuntimeError):
                release.migration_allowlist(old, invalid)
        with self.assertRaises(RuntimeError):
            release.verify_migrations(old, {**candidate, '004_other.sql': 'd'}, allowed)
        with self.assertRaises(RuntimeError):
            release.verify_migrations(old, {**candidate, '002_previous.sql': 'changed'}, allowed)

    def test_chat_retry_reuses_preapplied_chat_checksums_without_db_rollback(self):
        source_old = {'001_initial.sql': 'a'}
        previous_failed = {**source_old, '002_user_chat.sql': 'b'}
        candidate = {**previous_failed, '003_user_chat.sql': 'c'}
        allowed = release.migration_allowlist(source_old, candidate)
        self.assertEqual(release.verify_migrations(previous_failed, candidate, allowed), {'003_user_chat.sql': 'c'})
        self.assertEqual(release.verify_migrations(candidate, candidate, allowed), {})
        with self.assertRaises(RuntimeError):
            release.verify_migrations({**previous_failed, '002_user_chat.sql': 'wrong'}, candidate, allowed)

    def test_chat_migration_checksum_matches_trimmed_lf_build(self):
        self.assertEqual(release.migration_checksum(b' \r\nSELECT 1;\r\n '), release.digest(b'SELECT 1;'))

    def test_chat_compose_edits_only_application_image(self):
        original = ('name: deployment\nservices:\n  sub2api:\n    image: old:tag # preserve\n'
                    '    environment:\n      IMAGE: old:tag\n  postgres:\n    image: old:tag\n'
                    'volumes:\n  data:\n')
        actual = release.replace_application_image(original, 'sub2api:hk-chat-abcdef')
        self.assertEqual(actual, original.replace('image: old:tag # preserve', 'image: sub2api:hk-chat-abcdef # preserve'))
        with self.assertRaises(RuntimeError):
            release.replace_application_image('services:\n  postgres:\n    image: postgres:18\n', 'new')
        with self.assertRaises(RuntimeError):
            release.replace_application_image('services:\n  sub2api:\n    image: a\n    image: b\n', 'new')

    def test_chat_history_allows_only_expired_removal(self):
        at = dt.datetime(2026, 11, 13, tzinfo=dt.timezone.utc)
        before = {'1': {'created_at': '2026-01-01T00:00:00+00:00', 'sha256': 'old'},
                  '2': {'created_at': '2026-11-12T00:00:00+00:00', 'sha256': 'new'}}
        result = release.verify_history(before, {'2': before['2']}, 30, at)
        self.assertEqual(result['expired_rows_removed'], 1)
        with self.assertRaises(RuntimeError):
            release.verify_history(before, {}, 30, at)
        with self.assertRaises(RuntimeError):
            release.verify_history(before, {'2': before['2']}, 0, at)
        with self.assertRaises(RuntimeError):
            release.verify_history(before, {**before, '2': {**before['2'], 'sha256': 'changed'}}, 30, at)

    def test_chat_unrelated_container_restart_is_fatal(self):
        before = {'postgres': {'Id': 'db', 'State': {'StartedAt': 'then'}, 'RestartCount': 0}}
        release.verify_other_containers(before, before)
        with self.assertRaises(RuntimeError):
            release.verify_other_containers(before, {'postgres': {**before['postgres'], 'RestartCount': 1}})

    def test_chat_evidence_required_before_any_switch(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(release, 'RELEASES', Path(directory)):
            worker = release.Release('a' * 40, 'b' * 40)
            worker.load = lambda: None
            worker.manifest = {'phase': 'built'}
            called = []
            worker.containers = lambda: called.append('containers')
            worker.compose = lambda *args: called.append('compose')
            with self.assertRaises(RuntimeError):
                worker.deploy()
            self.assertEqual(called, [])

    def test_chat_existing_subscription_identity_requires_bounded_prepared_runner(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(release, 'RELEASES', Path(directory)):
            worker = release.Release('a' * 40, 'b' * 40)
            proof = {'user_id': 12, 'authorized': True, 'dedicated_test_identity': True,
                     'group_id': 179, 'model': release.MODEL, 'login_verified': True,
                     'group_access_verified': True, 'model_catalog_verified': True,
                     'concurrency': 1, 'budget_usd': '1', 'prepared_at': release.now().isoformat(),
                     'authorization_evidence': {}, 'login_evidence': {}, 'availability_evidence': {},
                     'budget_control': 'bounded_runner', 'budget_guard_evidence': {},
                     'budget_guard': {'max_model_requests': 12, 'max_tool_requests': 4,
                                      'max_input_bytes': 8192, 'max_output_tokens': 1024,
                                      'stop_on_unknown_billing': True, 'stop_at_budget': True}}
            worker.evidence = lambda name: proof
            worker.artifact = lambda value: value
            worker.rows = lambda statement: [{'role': 'user', 'status': 'active', 'balance': 5,
                                               'concurrency': 1, 'deleted': False}]
            worker.sql = lambda statement: self.fail('bounded runner must not forbid authorized subscription accounts')
            self.assertEqual(worker.identity()['user_id'], 12)
            proof['budget_guard']['max_model_requests'] = 13
            with self.assertRaises(RuntimeError):
                worker.identity()

    def test_chat_missing_real_paid_evidence_rolls_back_not_passes(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(release, 'RELEASES', Path(directory)):
            worker = release.Release('a' * 40, 'b' * 40)
            worker.load = lambda: None
            worker.manifest = {'phase': 'deployed-awaiting-real-acceptance'}
            calls = []
            worker.guard_after = lambda: calls.append('guard')
            worker.health_and_assets = lambda: calls.append('health')
            worker.rollback = lambda: calls.append('rollback')
            with self.assertRaises(RuntimeError):
                worker.verify()
            self.assertEqual(calls, ['guard', 'health', 'rollback'])

    def test_chat_health_or_mock_cannot_claim_real_acceptance(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(release, 'RELEASES', Path(directory)):
            worker = release.Release('a' * 40, 'b' * 40)
            worker.manifest = {'acceptance_user_id': 12}
            worker.evidence = lambda name: {'user_id': 12, 'group_id': 179, 'model': release.MODEL,
                                            'real_paid_calls': False, 'mock': True}
            with self.assertRaises(RuntimeError):
                worker.real_acceptance()

    def test_chat_real_evidence_requires_matching_owned_billing_and_cloud_runs(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(release, 'RELEASES', Path(directory)):
            worker = release.Release('a' * 40, 'b' * 40)
            worker.manifest = {'acceptance_user_id': 12, 'billing_cutoff_id': 10,
                               'deploy_started_at': (release.now() - dt.timedelta(minutes=5)).isoformat()}
            run_ids = ['00000000-0000-4000-8000-' + str(index).zfill(12) for index in range(4)]
            ids = ['chat-request-' + str(index) for index in range(4)]
            evidence = {'user_id': 12, 'group_id': 179, 'model': release.MODEL,
                        'real_paid_calls': True, 'mock': False, 'completed_at': release.now().isoformat(),
                        'checks': {name: {'passed': True, 'evidence': {}} for name in release.CHECKS},
                        'capabilities': {name: {'available': False, 'result': 'ui_disabled_backend_rejected', 'evidence': {}}
                                         for name in ('image', 'document', 'search', 'read_url')},
                        'request_ids': ids, 'tool_request_ids': [], 'actual_cost_usd': '0.04',
                        'run_ids': run_ids, 'marker': 'CHAT_OK_abcdef'}
            worker.evidence = lambda name: evidence
            worker.artifact = lambda value: value
            bills = [{'id': index + 11, 'request_id': request, 'group_id': 179, 'model': release.MODEL,
                      'actual_cost': '0.01', 'api_key_id': 20, 'key_user_id': 12, 'key_group_id': 179}
                     for index, request in enumerate(ids)]
            runs = [{'id': run_id, 'group_id': 179, 'model': release.MODEL,
                     'status': 'done' if index < 3 else 'stopped',
                     'gateway_request_ids': [ids[index]], 'session_id': 'cloud-session'}
                    for index, run_id in enumerate(run_ids)]
            def rows(statement):
                if 'FROM usage_logs' in statement:
                    return bills
                if 'FROM chat_generation_runs' in statement:
                    return runs
                return [{'id': 'cloud-session', 'revision': 2, 'has_marker': True}]
            worker.rows = rows
            worker.sql = lambda statement: '1' if 'FROM users' in statement else '0'
            self.assertEqual(worker.real_acceptance(), {'request_count': 4, 'actual_cost_usd': '0.04'})
            bills[0]['key_user_id'] = 99
            with self.assertRaises(RuntimeError):
                worker.real_acceptance()

    def test_chat_rollback_never_changes_database_or_other_services(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(release, 'RELEASES', Path(directory)), \
                patch.object(release, 'COMPOSE', Path(directory) / 'compose.yml'):
            worker = release.Release('a' * 40, 'b' * 40)
            original = 'services:\n  sub2api:\n    image: old:tag\n  postgres:\n    image: postgres:18\n'
            candidate = release.replace_application_image(original, 'new:tag')
            release.atomic_write(worker.audit / 'compose.before.yml', original)
            release.atomic_write(release.COMPOSE, candidate)
            old = {'Image': 'sha256:old', 'Config': {'Labels': {'org.opencontainers.image.revision': 'b' * 40}}}
            worker.manifest = {'image_id': 'sha256:new', 'old_image_id': 'sha256:old',
                               'backup_checksums': {'compose.before.yml': release.digest(original.encode())},
                               'candidate_compose_sha256': release.digest(candidate.encode())}
            worker.inspect = lambda name: old
            worker.wait_healthy = lambda revision: old
            worker.containers = lambda: {}
            (worker.audit / 'containers.before.json').write_text('{}')
            worker.fetch = lambda *args, **kwargs: (b'{"status":"ok"}', '')
            commands = []
            worker.compose = lambda *args: commands.append(args)
            worker.sql = lambda statement: self.fail('rollback must not mutate/restore DB')
            worker.rollback()
            self.assertEqual(commands, [('config', '--quiet'), ('up', '-d', '--no-deps', 'sub2api')])
            self.assertIn('image: sha256:old', release.COMPOSE.read_text())
            self.assertIn('image: postgres:18', release.COMPOSE.read_text())
            self.assertFalse(worker.manifest['database_restored'])

    def test_chat_rollback_refuses_parallel_compose_edit(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(release, 'RELEASES', Path(directory)), \
                patch.object(release, 'COMPOSE', Path(directory) / 'compose.yml'):
            worker = release.Release('a' * 40, 'b' * 40)
            release.COMPOSE.write_text('parallel change')
            worker.manifest = {'image_id': 'new', 'old_image_id': 'old',
                               'backup_checksums': {'compose.before.yml': 'old-checksum'}}
            worker.inspect = lambda name: {'Image': 'new'}
            with self.assertRaises(RuntimeError):
                worker.rollback()
            self.assertEqual(release.COMPOSE.read_text(), 'parallel change')


if __name__ == '__main__':
    unittest.main()
