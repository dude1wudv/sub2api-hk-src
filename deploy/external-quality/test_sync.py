import contextlib
import importlib.util
import io
import json
import tempfile
import unittest
import urllib.error
from pathlib import Path
from unittest import mock


SCRIPT = Path(__file__).with_name('sync.py')
SPEC = importlib.util.spec_from_file_location('external_quality_sync', SCRIPT)
sync = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(sync)


class SyncTests(unittest.TestCase):
    def setUp(self):
        original_root = sync.ROOT
        self.addCleanup(setattr, sync, 'ROOT', original_root)

    def test_admin_key_prefers_server_file_without_reading_database(self):
        with tempfile.TemporaryDirectory() as directory:
            sync.ROOT = Path(directory)
            (sync.ROOT / 'admin.key').write_text('adm_sc_server-only-secret\n')
            with mock.patch.object(sync.subprocess, 'check_output') as query:
                self.assertEqual(sync.admin_key(), 'adm_sc_server-only-secret')
            query.assert_not_called()

    def test_admin_key_empty_server_file_does_not_fall_back(self):
        with tempfile.TemporaryDirectory() as directory:
            sync.ROOT = Path(directory)
            (sync.ROOT / 'admin.key').write_text(' \n')
            with mock.patch.object(sync.subprocess, 'check_output') as query:
                with self.assertRaises(sync.AdminIntegrationKeyMissing):
                    sync.admin_key()
            query.assert_not_called()

    def test_admin_key_supports_legacy_setting_when_file_is_absent(self):
        with tempfile.TemporaryDirectory() as directory:
            sync.ROOT = Path(directory)
            with mock.patch.object(sync.subprocess, 'check_output', return_value='legacy-secret\n') as query:
                self.assertEqual(sync.admin_key(), 'legacy-secret')
            self.assertIn("SELECT value FROM settings WHERE key='admin_api_key'", query.call_args.args[0])

    def test_admin_key_missing_has_actionable_secret_free_status(self):
        with tempfile.TemporaryDirectory() as directory:
            sync.ROOT = Path(directory)
            stderr = io.StringIO()
            with mock.patch.object(sync, 'access_token', return_value='access-secret'), \
                    mock.patch.object(sync.subprocess, 'check_output', return_value=''), \
                    contextlib.redirect_stderr(stderr):
                self.assertEqual(sync.main(), 1)
            status = json.loads((sync.ROOT / 'status.json').read_text())
            self.assertEqual(status['error'], 'AdminIntegrationKeyMissing')
            self.assertNotIn('access-secret', stderr.getvalue())

    def test_refresh_rotates_token_only_in_server_storage(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'mdkj.refresh').write_text('r' * 48)
            sync.ROOT = root
            with mock.patch.object(sync, 'request', return_value={
                'access_token': 'access-secret', 'refresh_token': 'rotated-secret', 'expires_in': 600,
            }) as request:
                self.assertEqual(sync.access_token(), 'access-secret')
            sent = request.call_args.args
            self.assertEqual(sent[0], 'https://mdkj.lol/api/v1/auth/refresh')
            self.assertEqual(sent[1], {'refresh_token': 'r' * 48})
            self.assertEqual((root / 'mdkj.refresh').read_text(), 'rotated-secret')
            state = json.loads((root / 'session.json').read_text())
            self.assertEqual(state['access_token'], 'access-secret')
            self.assertFalse((root / 'mdkj.refresh.tmp').exists())
            self.assertFalse((root / 'session.tmp').exists())

    def test_redirect_handler_refuses_forwarding_authorization(self):
        handler = sync.NoRedirect()
        request = __import__('urllib.request').request.Request(
            'https://mdkj.lol/api', headers={'Authorization': 'Bearer secret'})
        redirected = handler.redirect_request(request, None, 302, 'Found', {}, 'https://other.example/api')
        self.assertIsNone(redirected)
        self.assertEqual(request.get_header('Authorization'), 'Bearer secret')

    def test_completed_events_allowlists_models_status_group_and_limit(self):
        rows = []
        for model in sync.MODELS:
            for index in range(12):
                rows.append({'id': f'{model}-{index}', 'group_id': 62, 'status': 'completed',
                             'model': model, 'started_at': f'2026-11-09T00:{index:02}:00+08:00',
                             'html': '<p>artwork</p>'})
        rows += [
            {'id': 'wrong-group', 'group_id': 170, 'status': 'completed', 'model': 'gpt-6-astra',
             'started_at': '2026-11-09T00:59:00+08:00'},
            {'id': 'wrong-status', 'group_id': 62, 'status': 'failed', 'model': 'gpt-6-astra',
             'started_at': '2026-11-09T00:59:00+08:00'},
            {'id': 'unknown-model', 'group_id': 62, 'status': 'completed', 'model': 'gpt-5',
             'started_at': '2026-11-09T00:59:00+08:00'},
        ]
        selected = sync.completed_events(rows, 62)
        self.assertEqual(len(selected), 20)
        self.assertEqual({event['model'] for event in selected}, sync.MODELS)
        for model in sync.MODELS:
            group = [event for event in selected if event['model'] == model]
            self.assertEqual(len(group), 10)
            self.assertEqual(group[0]['id'], f'{model}-11')

    def test_sync_refreshes_after_401_and_imports_only_fixed_group_bindings(self):
        sync.ROOT = Path('/server-only')
        events = {
            62: [{'id': 'pro', 'group_id': 62, 'status': 'completed', 'model': 'gpt-6-astra',
                  'started_at': '2026-11-09T00:00:00+08:00', 'finished_at': '2026-11-09T00:01:00+08:00',
                  'html': '<p>private artwork</p>'}],
            170: [{'id': 'plus', 'group_id': 170, 'status': 'completed', 'model': 'gpt-6.1-sol',
                   'started_at': '2026-11-09T00:00:00+08:00', 'finished_at': '2026-11-09T00:01:00+08:00',
                   'html': '<p>private artwork</p>'}],
        }
        calls = []

        def fake_request(url, body=None, headers=None):
            calls.append((url, body, headers))
            if url.startswith('https://mdkj.lol/api/v1/channel-monitor-v2/') and headers['Authorization'] == 'Bearer old-access':
                raise urllib.error.HTTPError(url, 401, 'Unauthorized', {}, None)
            if url == 'https://mdkj.lol/api/v1/auth/refresh':
                return {'access_token': 'new-access'}
            if '/quality/history/62?' in url:
                return events[62]
            if '/quality/history/170?' in url:
                return events[170]
            return {'imported': 1}

        with mock.patch.object(sync, 'access_token', side_effect=['old-access', 'new-access']) as token, \
                mock.patch.object(sync, 'admin_key', return_value='legacy-admin-secret'), \
                mock.patch.object(sync, 'request', side_effect=fake_request):
            result = sync.sync()

        self.assertEqual(token.call_args_list, [mock.call(), mock.call(force=True)])
        self.assertEqual([(item[1]['group_id'], item[1]['remote_group_id'])
                          for item in calls if item[1] and item[1].get('source') == 'mdkj'],
                         [(72, 62), (237, 170)])
        imports = [item for item in calls if item[0].startswith('http://127.0.0.1')]
        self.assertEqual(len(imports), 2)
        self.assertTrue(all(item[2]['x-api-key'] == 'legacy-admin-secret' for item in imports))
        self.assertEqual([item['group_id'] for item in result], [72, 237])

    def test_failure_log_and_status_do_not_include_tokens_or_artwork(self):
        with tempfile.TemporaryDirectory() as directory:
            sync.ROOT = Path(directory)
            secret_text = 'refresh-secret access-secret <private html>'
            stderr = io.StringIO()
            with mock.patch.object(sync, 'sync', side_effect=ValueError(secret_text)), \
                    contextlib.redirect_stderr(stderr):
                self.assertEqual(sync.main(), 1)
            output = stderr.getvalue() + (Path(directory) / 'status.json').read_text()
            for secret in ('refresh-secret', 'access-secret', '<private html>'):
                self.assertNotIn(secret, output)
            status = json.loads((Path(directory) / 'status.json').read_text())
            self.assertEqual(status['error'], 'ValueError')


if __name__ == '__main__':
    unittest.main()
