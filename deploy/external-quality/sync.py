#!/usr/bin/env python3
"""Poll completed mdkj outputs. Credentials and state stay on the HK host."""
import datetime as dt
import json
import os
from pathlib import Path
import subprocess
import sys
import time
import urllib.error
import urllib.request

ROOT = Path('/opt/sub2api-external-quality')
ORIGIN = 'https://mdkj.lol'
BINDINGS = ((72, 62), (237, 170))
MODELS = {'gpt-6-astra', 'gpt-6.1-sol'}
MAX_BYTES = 8 << 20


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


OPENER = urllib.request.build_opener(NoRedirect)


def request(url, body=None, headers=None):
    data = None if body is None else json.dumps(body).encode()
    common = {'User-Agent': 'Mozilla/5.0', 'Accept': 'application/json'}
    if data is not None:
        common['Content-Type'] = 'application/json'
    common.update(headers or {})
    req = urllib.request.Request(url, data=data, headers=common)
    with OPENER.open(req, timeout=35) as response:
        raw = response.read(MAX_BYTES + 1)
    if len(raw) > MAX_BYTES:
        raise ValueError('response size limit')
    value = json.loads(raw)
    if not isinstance(value, dict) or value.get('code') != 0:
        raise ValueError('unsuccessful response envelope')
    return value['data']


def atomic_json(path, value):
    tmp = path.with_suffix('.tmp')
    fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, 'w') as stream:
        json.dump(value, stream)
    os.replace(tmp, path)


def access_token(force=False):
    state = ROOT / 'session.json'
    if not force and state.exists():
        saved = json.loads(state.read_text())
        if saved.get('expires_at', 0) > time.time() + 120:
            return saved['access_token']
    refresh = (ROOT / 'mdkj.refresh').read_text().strip()
    if len(refresh) < 30:
        raise ValueError('refresh credential missing')
    data = request(ORIGIN + '/api/v1/auth/refresh', {'refresh_token': refresh},
                   {'Referer': ORIGIN + '/monitor/cards'})
    if not data.get('access_token') or not data.get('refresh_token'):
        raise ValueError('invalid refresh response')
    # Persist the rotated credential before using the access token.
    tmp = ROOT / 'mdkj.refresh.tmp'
    fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, 'w') as stream:
        stream.write(data['refresh_token'])
    os.replace(tmp, ROOT / 'mdkj.refresh')
    atomic_json(state, {'access_token': data['access_token'],
                        'expires_at': time.time() + max(0, int(data.get('expires_in', 900)))})
    return data['access_token']


def completed_events(data, remote_id):
    if not isinstance(data, list) or len(data) > 100:
        raise ValueError('invalid history list')
    selected = []
    for model in sorted(MODELS):
        rows = [row for row in data if isinstance(row, dict)
                and row.get('group_id') == remote_id
                and row.get('status') == 'completed' and row.get('model') == model]
        rows.sort(key=lambda row: dt.datetime.fromisoformat(row['started_at']), reverse=True)
        selected.extend(rows[:10])
    return selected


def admin_key():
    # Capture only in server process memory, never logs or an on-disk export.
    return subprocess.check_output([
        'docker', 'exec', 'sub2api-postgres', 'psql', '-X', '-At', '-U', 'sub2api',
        '-d', 'sub2api', '-c', "SELECT value FROM settings WHERE key='admin_api_key'"
    ], text=True, timeout=15).strip()


def sync():
    token = access_token()
    key = admin_key()
    if not key:
        raise ValueError('local admin integration key missing')
    summaries = []
    for local_id, remote_id in BINDINGS:
        url = ORIGIN + f'/api/v1/channel-monitor-v2/quality/history/{remote_id}?include_status=1&timezone=Etc%2FGMT%2B7'
        try:
            data = request(url, headers={'Authorization': 'Bearer ' + token,
                                        'Referer': ORIGIN + '/monitor/cards'})
        except urllib.error.HTTPError as error:
            if error.code != 401:
                raise
            token = access_token(force=True)
            data = request(url, headers={'Authorization': 'Bearer ' + token,
                                        'Referer': ORIGIN + '/monitor/cards'})
        events = completed_events(data, remote_id)
        imported = 0
        # Each batch stays below the application's 4 MiB import limit.
        for offset in range(0, len(events), 10):
            payload = {'source': 'mdkj', 'group_id': local_id, 'remote_group_id': remote_id,
                       'payload': {'code': 0, 'data': events[offset:offset + 10]}}
            result = request('http://127.0.0.1:8080/api/v1/admin/channel-monitor-v2/external-quality/import',
                             payload, {'x-api-key': key})
            imported += result['imported']
        summaries.append({'group_id': local_id, 'received': len(events), 'imported': imported,
                          'latest_test_at': max((e['finished_at'] for e in events), default=None)})
    return summaries


def main():
    os.umask(0o077)
    ROOT.mkdir(mode=0o700, parents=True, exist_ok=True)
    try:
        groups = sync()
        state = {'ok': True, 'synced_at': dt.datetime.now(dt.timezone.utc).isoformat(), 'groups': groups}
        atomic_json(ROOT / 'status.json', state)
        print(json.dumps(state))
        return 0
    except Exception as error:
        # Exception bodies/URLs can contain credentials; log class/status only.
        state = {'ok': False, 'attempted_at': dt.datetime.now(dt.timezone.utc).isoformat(),
                 'error': type(error).__name__, 'http_status': getattr(error, 'code', None)}
        atomic_json(ROOT / 'status.json', state)
        print(json.dumps(state), file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
