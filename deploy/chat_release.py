"""HK-only, application-only chat releases. Evidence and backups never leave HK.

Automated acceptance (default) requires private, SHA256-pinned evidence:
  <release-dir>/acceptance-identity.json before deploy;
  <release-dir>/real-acceptance.json before verify.
Explicit user handoff requires --manual-acceptance on deploy and a private
manual-acceptance.json bound to this release. It leaves paid acceptance pending;
health checks never substitute for real authenticated model/billing evidence.
No credentials belong in evidence documents. All DB access below is read-only.
"""
import argparse
import datetime as dt
from decimal import Decimal
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import time
import urllib.parse

COMPOSE = Path('/opt/sub2api/docker-compose.yml')
RELEASES = Path('/opt/sub2api-deploy')
REMOTE = 'ssh://git@gh-sub2api-hk-src/dude1wudv/sub2api-hk-src.git'
DOMAINS = ('patrickapi.microedulab.com', 'sub.sunmmyapi.xyz', 'cf.sunmmyapi.xyz')
MODEL = 'deepseek-v4.1-flash'
CHECKS = ('stream_saved', 'second_browser_cloud_history', 'edit_regenerate_branches',
          'stop_partial_no_duplicate_billing', 'manual_group_and_compact_picker',
          'capability_boundaries', 'login_and_iframe_security')


def require(ok, message):
    if not ok:
        raise RuntimeError(message)


def now():
    return dt.datetime.now(dt.timezone.utc)


def timestamp(value):
    result = dt.datetime.fromisoformat(value.replace('Z', '+00:00'))
    require(result.tzinfo is not None, 'Evidence timestamp must have a timezone')
    return result


def digest(data):
    return hashlib.sha256(data).hexdigest()


def file_digest(path):
    checksum = hashlib.sha256()
    with path.open('rb') as source:
        for block in iter(lambda: source.read(1024 * 1024), b''):
            checksum.update(block)
    return checksum.hexdigest()


def atomic_write(path, data):
    tmp = path.with_name(path.name + '.tmp')
    with tmp.open('wb') as out:
        os.chmod(tmp, 0o600)
        out.write(data if isinstance(data, bytes) else data.encode())
        out.flush()
        os.fsync(out.fileno())
    os.replace(tmp, path)


def migration_checksum(data):
    # Docker canonicalizes CRLF; Go's migration runner hashes TrimSpace(content).
    return digest(data.replace(b'\r\n', b'\n').decode().strip().encode())


def migration_allowlist(before, candidate):
    require(all(candidate.get(name) == checksum for name, checksum in before.items()),
            'Existing migration source removed or changed')
    added = {name: value for name, value in candidate.items() if name not in before}
    prefixes_before = []
    for name in before:
        prefix = re.match(r'^(\d+)[A-Za-z]*_', name)
        require(prefix is not None, 'Unsupported migration filename: ' + name)
        prefixes_before.append(int(prefix.group(1)))
    maximum = max(prefixes_before, default=0)
    require(bool(added), 'Candidate must contain appended chat migrations')
    require(all(re.fullmatch(r'\d+_user_chat\.sql', name) for name in added), 'Only append-only user_chat migrations may be introduced')
    prefixes = sorted(int(name.split('_')[0]) for name in added)
    require(prefixes == list(range(maximum + 1, maximum + 1 + len(added))), 'Chat migrations must use consecutive new prefixes')
    return added


def verify_migrations(before, after, allowed):
    require(all(after.get(name) == value for name, value in before.items()), 'Existing migration checksum changed')
    require(all(name not in before or before[name] == value for name, value in allowed.items()), 'Previously applied chat migration checksum changed')
    added = {name: value for name, value in after.items() if name not in before}
    pending = {name: value for name, value in allowed.items() if name not in before}
    require(added == pending, 'Applied migrations differ from the exact manifest allowlist')
    return added


def verify_history(before, after, days, at):
    require(not (after.keys() - before.keys()), 'Unexpected historical billing row')
    require(all(before[key] == after[key] for key in before.keys() & after.keys()), 'Historical billing row changed')
    removed = before.keys() - after.keys()
    boundary = at - dt.timedelta(days=days)
    require(not removed or (days > 0 and all(timestamp(before[key]['created_at']) < boundary for key in removed)),
            'Unexpired historical billing row disappeared')
    return {'unchanged_retained_rows': len(after), 'expired_rows_removed': len(removed),
            'retention_days': days, 'expiration_cutoff': boundary.isoformat()}


def replace_application_image(text, image):
    # Do not parse/re-serialize the rest of Compose, including interpolation/comments.
    lines = text.splitlines(keepends=True)
    in_services = False
    in_app = False
    service_indent = None
    matches = []
    for index, line in enumerate(lines):
        if re.fullmatch(r'services:\s*(?:#.*)?\n?', line):
            in_services = True
            continue
        if not in_services or not line.strip() or line.lstrip().startswith('#'):
            continue
        indent = len(line) - len(line.lstrip())
        if indent == 0:
            in_services = False
            in_app = False
            continue
        if re.match(r'^\s+sub2api:\s*(?:#.*)?$', line.rstrip()):
            in_app = True
            service_indent = indent
            continue
        if in_app and indent <= service_indent:
            in_app = False
        if in_app and re.match(r'^\s+image:\s*', line):
            matches.append(index)
    require(len(matches) == 1, 'Compose must contain exactly one sub2api image field')
    index = matches[0]
    prefix = re.match(r'^(\s+image:\s*)', lines[index]).group(1)
    comment = re.search(r'\s+#.*', lines[index].rstrip())
    lines[index] = prefix + image + (comment.group(0) if comment else '') + '\n'
    return ''.join(lines)


def runtime_fingerprint(container):
    config = container['HostConfig']
    return {'env': digest(json.dumps(sorted(container['Config']['Env'])).encode()),
            'mounts': container['Mounts'], 'networks': sorted(container['NetworkSettings']['Networks']),
            'host': {key: config.get(key) for key in ('PortBindings', 'Memory', 'MemorySwap', 'NanoCpus',
                     'CpuQuota', 'CpuPeriod', 'CpusetCpus', 'PidsLimit', 'RestartPolicy', 'SecurityOpt',
                     'CapDrop', 'CapAdd', 'ReadonlyRootfs', 'Ulimits', 'LogConfig')}}


def verify_other_containers(before, after):
    require(set(before) - {'sub2api'} == set(after) - {'sub2api'}, 'Other container inventory changed')
    for name, value in before.items():
        if name == 'sub2api':
            continue
        other = after[name]
        require((value['Id'], value['State']['StartedAt'], value['RestartCount']) ==
                (other['Id'], other['State']['StartedAt'], other['RestartCount']),
                'Unrelated container identity/start/restart changed: ' + name)


class Release:
    def __init__(self, sha, expected):
        self.sha, self.expected = sha, expected
        self.base = RELEASES / ('chat-' + sha)
        self.base.mkdir(mode=0o700, parents=True, exist_ok=True)
        os.chmod(self.base, 0o700)
        self.audit = self.base / 'audit'
        self.audit.mkdir(mode=0o700, exist_ok=True)
        os.chmod(self.audit, 0o700)
        self.src = self.base / 'source'
        self.manifest_path = self.base / 'manifest.json'
        self.command_index = 0
        self.manifest = None

    def save(self, name, value):
        atomic_write(self.audit / name, json.dumps(value, ensure_ascii=False, indent=2) + '\n')

    def persist(self):
        atomic_write(self.manifest_path, json.dumps(self.manifest, indent=2) + '\n')

    def load(self):
        self.manifest = json.loads(self.manifest_path.read_text())
        require(self.manifest['sha'] == self.sha and self.manifest['expected_current_sha'] == self.expected,
                'Manifest release/baseline mismatch')
        require(self.manifest['script_sha256'] == file_digest(Path(__file__)), 'Release script changed between phases')

    def run(self, args, *, cwd=None, data=None):
        # Never propagate workstation proxy settings to HK's Git/Docker/network.
        env = {key: value for key, value in os.environ.items() if key.lower() not in
               ('http_proxy', 'https_proxy', 'all_proxy')}
        env['GIT_TERMINAL_PROMPT'] = '0'
        result = subprocess.run(args, cwd=cwd, env=env, input=data, capture_output=True)
        self.command_index += 1
        log = self.audit / f'command-{time.time_ns()}-{self.command_index}.log'
        atomic_write(log, result.stdout + b'\n' + result.stderr)
        require(result.returncode == 0, 'Command failed; private evidence: ' + str(log))
        return result.stdout.decode().strip()

    def inspect(self, name):
        return json.loads(self.run(['docker', 'inspect', name]))[0]

    def containers(self):
        return {name: self.inspect(name) for name in self.run(['docker', 'ps', '-a', '--format', '{{.Names}}']).splitlines()}

    def pg_command(self, command):
        pg = self.inspect('sub2api-postgres')
        env = dict(item.split('=', 1) for item in pg['Config']['Env'] if '=' in item)
        user = env.get('POSTGRES_USER', 'postgres')
        return ['docker', 'exec', 'sub2api-postgres', command, '-U', user, '-d', env.get('POSTGRES_DB', user)]

    def sql(self, statement):
        return self.run(self.pg_command('psql') + ['-X', '-v', 'ON_ERROR_STOP=1', '-Atc', statement])

    def rows(self, statement):
        return [json.loads(line) for line in self.sql(statement).splitlines()]

    def migrations(self):
        return dict(line.split('|', 1) for line in self.sql('SELECT filename, checksum FROM schema_migrations ORDER BY filename').splitlines())

    def retention(self, app):
        runtime = dict(item.split('=', 1) for item in app['Config']['Env'] if '=' in item)
        value = self.sql("SELECT coalesce(value::jsonb->>'request_retention_days','') FROM settings WHERE key='ops_runtime_log_config'")
        value = value or runtime.get('DASHBOARD_AGGREGATION_RETENTION_USAGE_LOGS_DAYS')
        require(value is not None, 'Explicit runtime usage retention policy required')
        days = int(value)
        require(0 <= days <= 3650, 'Invalid retention days')
        return days

    def history(self, cutoff, filename):
        # Stream large billing tables; keep only id/time/hash, not raw prompts, in evidence.
        command = self.pg_command('psql') + ['-X', '-v', 'ON_ERROR_STOP=1', '-Atc',
                    f'SELECT to_jsonb(u)::text FROM usage_logs u WHERE id <= {int(cutoff)} ORDER BY id']
        rows = {}
        with (self.audit / (filename + '.stderr')).open('wb') as error:
            proc = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=error)
            for line in proc.stdout:
                row = json.loads(line)
                rows[str(row['id'])] = {'created_at': row['created_at'], 'sha256': digest(line)}
            require(proc.wait() == 0, 'Historical billing fingerprint query failed')
        self.save(filename, rows)
        return rows

    def artifact(self, evidence):
        name, checksum = evidence['path'], evidence['sha256']
        require(isinstance(name, str) and not Path(name).is_absolute(), 'Evidence artifact must be relative')
        path = (self.base / name).resolve()
        require(path.is_relative_to(self.base.resolve()) and not path.is_symlink() and path.is_file(), 'Unsafe/missing evidence artifact')
        require(re.fullmatch(r'[0-9a-f]{64}', checksum) and file_digest(path) == checksum, 'Evidence artifact checksum mismatch')
        require(path.stat().st_mode & 0o077 == 0, 'Evidence must be private (0600)')
        return evidence

    def evidence(self, name):
        path = self.base / name
        require(path.is_file() and not path.is_symlink() and path.stat().st_mode & 0o077 == 0,
                f'Required private evidence missing: {path}')
        value = json.loads(path.read_text())
        require(value.get('sha') == self.sha and value.get('expected_current_sha') == self.expected,
                'Evidence is not bound to this release and baseline')
        return value

    def manual_acceptance_authorization(self):
        evidence = self.evidence('manual-acceptance.json')
        require(evidence.get('deployment_authorized') is True and
                evidence.get('paid_acceptance_by') == 'user' and
                evidence.get('automatic_paid_calls') is False,
                'Explicit user authorization to deploy with manual paid acceptance required')
        require(dt.timedelta(0) <= now() - timestamp(evidence['prepared_at']) <= dt.timedelta(hours=4),
                'Manual acceptance authorization is stale')
        self.artifact(evidence['authorization_evidence'])
        self.save('manual-acceptance-preflight.json', evidence)
        return evidence

    def identity(self):
        evidence = self.evidence('acceptance-identity.json')
        user_id = evidence.get('user_id')
        require(type(user_id) is int and user_id > 0, 'Explicit authorized acceptance user required')
        require(evidence.get('authorized') is True and evidence.get('dedicated_test_identity') is True,
                'Dedicated authorized acceptance identity required before switching')
        require(evidence.get('group_id') == 179 and evidence.get('model') == MODEL,
                'Acceptance must use the authorized group/model')
        require(evidence.get('login_verified') is True and evidence.get('concurrency') == 1 and
                Decimal(str(evidence.get('budget_usd', '0'))) > 0 and Decimal(str(evidence['budget_usd'])) <= 1,
                'Authenticated login, concurrency 1, and <= USD 1 budget required before deploy')
        require(dt.timedelta(0) <= now() - timestamp(evidence['prepared_at']) <= dt.timedelta(hours=4), 'Identity preparation is stale')
        self.artifact(evidence['authorization_evidence'])
        self.artifact(evidence['login_evidence'])
        require(evidence.get('group_access_verified') is True and evidence.get('model_catalog_verified') is True,
                'Verify authorized target-group access and model catalog before switching')
        self.artifact(evidence['availability_evidence'])
        users = self.rows(f"SELECT json_build_object('role',role,'status',status,'balance',balance,'concurrency',concurrency,'deleted',deleted_at IS NOT NULL)::text FROM users WHERE id={user_id}")
        require(len(users) == 1 and users[0]['role'] == 'user' and users[0]['status'] == 'active' and not users[0]['deleted'] and users[0]['concurrency'] == 1,
                'Acceptance identity must be an active ordinary user with concurrency 1')
        control = evidence.get('budget_control')
        require(control in ('dedicated_balance', 'bounded_runner'), 'Explicit acceptance budget control required')
        if control == 'dedicated_balance':
            # A dedicated pay-as-you-go balance bounds all automatically created Keys.
            require(0 < Decimal(str(users[0]['balance'])) <= Decimal(str(evidence['budget_usd'])), 'Dedicated account balance exceeds acceptance budget')
            require(self.sql(f"SELECT count(*) FROM user_subscriptions WHERE user_id={user_id} AND deleted_at IS NULL AND status='active' AND expires_at>now()") == '0',
                    'Subscription acceptance requires a prepared bounded runner')
        else:
            # Existing authorized balance/subscription identities require an inspected,
            # SHA256-pinned bounded acceptance runner prepared before switching.
            guard = evidence.get('budget_guard', {})
            limits = {'max_model_requests': 12, 'max_tool_requests': 4, 'max_input_bytes': 8192, 'max_output_tokens': 1024}
            require(all(type(guard.get(name)) is int and 0 < guard[name] <= limit for name, limit in limits.items()) and
                    guard.get('stop_on_unknown_billing') is True and guard.get('stop_at_budget') is True,
                    'Bounded runner must enforce all authorized request/token/expense limits')
            self.artifact(evidence['budget_guard_evidence'])
        self.save('identity-preflight.json', evidence)
        return evidence

    def git(self, *args):
        return self.run(['git', '-C', str(self.src), *args])

    def source_migrations(self, ref):
        names = self.git('ls-tree', '-r', '--name-only', ref, 'backend/migrations/').splitlines()
        return {Path(name).name: migration_checksum(self.run(['git', '-C', str(self.src), 'show', f'{ref}:{name}']).encode())
                for name in names if Path(name).parent.as_posix() == 'backend/migrations' and name.endswith('.sql')}

    def build(self):
        require(not self.manifest_path.exists(), 'Release manifest already exists; use a new candidate SHA instead of overwriting evidence')
        current = self.inspect('sub2api')
        require(current['Config']['Labels'].get('org.opencontainers.image.revision') == self.expected,
                'Running production SHA differs from expected baseline')
        owner_path = self.base / 'source-owner.json'
        owner = {'sha': self.sha, 'expected_current_sha': self.expected, 'remote': REMOTE}
        if owner_path.exists():
            require(json.loads(owner_path.read_text()) == owner and not self.src.is_symlink(), 'Private source ownership mismatch')
        else:
            require(not self.src.exists() or not any(self.src.iterdir()), 'Unowned source directory is not empty')
            atomic_write(owner_path, json.dumps(owner))
        self.src.mkdir(mode=0o700, exist_ok=True)
        if not (self.src / '.git').exists():
            self.run(['git', 'init', str(self.src)])
            self.git('remote', 'add', 'origin', REMOTE)
        require(self.git('remote', 'get-url', 'origin') == REMOTE, 'Unexpected private source remote')
        require(self.git('status', '--porcelain') == '', 'Private release source has uncommitted changes')
        self.git('fetch', '--no-tags', 'origin', self.sha, self.expected)
        require(self.git('merge-base', '--is-ancestor', self.expected, self.sha) == '', 'Candidate does not include current production')
        self.git('checkout', '--detach', self.sha)
        require(self.git('rev-parse', 'HEAD') == self.sha, 'Candidate source mismatch')
        require(file_digest(Path(__file__)) == file_digest(self.src / 'deploy/chat_release.py'),
                'Run the release entrypoint from the fixed candidate SHA, not historical tooling')
        require(self.git('config', '-f', '.gitmodules', '--get', 'submodule.chat-demo.url') == 'https://github.com/dude1wudv/chat-demo.git', 'Unexpected fork origin')
        self.git('submodule', 'update', '--init', '--', 'chat-demo')
        link = self.git('ls-tree', 'HEAD', 'chat-demo').split()
        require(len(link) >= 3 and link[0] == '160000', 'chat-demo is not a gitlink')
        fork = self.run(['git', '-C', str(self.src / 'chat-demo'), 'rev-parse', 'HEAD'])
        require(fork == link[2], 'Fork checkout differs from gitlink')
        old = self.source_migrations(self.expected)
        allowed = migration_allowlist(old, self.source_migrations(self.sha))
        image = 'sub2api:hk-chat-' + self.sha[:12]
        self.run(['docker', 'build', '--build-arg', 'COMMIT=' + self.sha, '--build-arg', 'VERSION=' + self.sha,
                  '--label', 'org.opencontainers.image.revision=' + self.sha, '-t', image, str(self.src)])
        built = self.inspect(image)
        require(built['Config']['Labels'].get('org.opencontainers.image.revision') == self.sha, 'Built image revision mismatch')
        # Record every embedded asset hash; verify bytes, not just filenames/health.
        staging = self.audit / 'embedded'
        require(not staging.is_symlink(), 'Unsafe generated artifact directory')
        if staging.exists():
            shutil.rmtree(staging)
        staging.mkdir(mode=0o700)
        # This target reuses exactly the cached frontend stage embedded above.
        builder = image + '-frontend'
        self.run(['docker', 'build', '--target', 'frontend-builder', '-t', builder, str(self.src)])
        frontend_container = self.run(['docker', 'create', builder])
        try:
            self.run(['docker', 'cp', frontend_container + ':/app/backend/internal/web/dist/.', str(staging)])
        finally:
            self.run(['docker', 'rm', frontend_container])
        assets = {path.relative_to(staging).as_posix(): digest(path.read_bytes()) for path in staging.rglob('*') if path.is_file()}
        require(all(name in assets for name in ('index.html', 'chat-app/index.html', 'chat-app/LICENSE')) and
                any(name.startswith('chat-app/static/js/') for name in assets), 'Build is missing chat bundle/license')
        self.manifest = {'version': 1, 'sha': self.sha, 'expected_current_sha': self.expected,
                         'fork_gitlink': fork, 'image': image, 'image_id': built['Id'],
                         'script_sha256': file_digest(Path(__file__)),
                         'old_image': current['Config']['Image'], 'old_image_id': current['Image'],
                         'migration_allowlist': allowed, 'assets': assets, 'built_at': now().isoformat(),
                         'backup_path': str(self.audit), 'phase': 'built'}
        self.persist()

    def compose(self, *args):
        return self.run(['docker', 'compose', '-f', str(COMPOSE), *args], cwd=str(COMPOSE.parent))

    def wait_healthy(self, revision):
        for _ in range(90):
            app = self.inspect('sub2api')
            require(app['Config']['Labels'].get('org.opencontainers.image.revision') == revision, 'Unexpected running revision')
            if app['State'].get('Health', {}).get('Status') == 'healthy':
                return app
            if app['State'].get('Status') in ('dead', 'exited'):
                break
            time.sleep(2)
        raise RuntimeError('Application did not become healthy')

    def backup(self, before):
        atomic_write(self.audit / 'compose.before.yml', COMPOSE.read_bytes())
        environment = COMPOSE.parent / '.env'
        if environment.is_file():
            atomic_write(self.audit / 'compose.env.before', environment.read_bytes())
        self.save('containers.before.json', before)
        self.save('migrations.before.json', self.migrations())
        self.manifest['retention_days'] = self.retention(before['sub2api'])
        cutoff = int(self.sql('SELECT coalesce(max(id),0) FROM usage_logs'))
        self.manifest['billing_cutoff_id'] = cutoff
        self.history(cutoff, 'history.before.json')
        dump = self.audit / 'db.before.dump'
        with dump.open('wb') as out, (self.audit / 'backup.stderr').open('wb') as error:
            subprocess.run(self.pg_command('pg_dump') + ['-Fc'], stdout=out, stderr=error, check=True)
        require(dump.stat().st_size > 0, 'Database backup empty')
        with dump.open('rb') as source, (self.audit / 'db.before.list').open('wb') as listing:
            subprocess.run(['docker', 'exec', '-i', 'sub2api-postgres', 'pg_restore', '--list'],
                           stdin=source, stdout=listing, stderr=subprocess.PIPE, check=True)
        require((self.audit / 'db.before.list').stat().st_size > 0, 'Database dump unreadable')
        app = before['sub2api']
        env = dict(item.split('=', 1) for item in app['Config']['Env'] if '=' in item)
        def host_path(container_path):
            require(container_path.startswith('/') and '..' not in container_path.split('/'), 'Persistent container paths must be absolute and normalized')
            matching = [item for item in app['Mounts'] if container_path == item['Destination'] or
                        container_path.startswith(item['Destination'].rstrip('/') + '/')]
            require(bool(matching), 'Cannot resolve configured private storage into a persistent mount')
            mount = max(matching, key=lambda value: len(value['Destination']))
            return Path(mount['Source']) / container_path[len(mount['Destination']):].lstrip('/')
        data_dir = env.get('DATA_DIR') or '/app/data'
        config = host_path(data_dir.rstrip('/') + '/config.yaml')
        configured = None
        if config.exists():
            atomic_write(self.audit / 'config.before.yaml', config.read_bytes())
            try:
                import yaml
            except ImportError as error:
                raise RuntimeError('Install the existing .github/release-tools/requirements-release.txt on HK before deploy') from error
            document = yaml.safe_load(config.read_text()) or {}
            require(isinstance(document, dict), 'Invalid application YAML configuration')
            chat_config = document.get('chat') or {}
            require(isinstance(chat_config, dict), 'Invalid chat YAML configuration')
            configured = chat_config.get('storage_dir')
        chat_dir = env.get('CHAT_STORAGE_DIR') or configured or data_dir.rstrip('/') + '/chat'
        require(isinstance(chat_dir, str) and chat_dir.rstrip('/') != data_dir.rstrip('/'), 'Chat backup must not target the general data directory')
        host_chat = host_path(chat_dir.rstrip('/'))
        require(not host_chat.is_symlink() and (not host_chat.exists() or host_chat.is_dir()), 'Unsafe private chat storage path')
        self.manifest['chat_storage_path'] = str(host_chat)
        if host_chat.exists():
            archive = self.audit / 'chat.before.tar'
            self.run(['tar', '-cpf', str(archive), '-C', str(host_chat.parent), host_chat.name])
            self.run(['tar', '-tf', str(archive)])
        checksums = {path.name: file_digest(path) for path in self.audit.iterdir()
                     if path.is_file() and path.name in ('db.before.dump', 'db.before.list', 'compose.before.yml',
                     'compose.env.before', 'config.before.yaml', 'chat.before.tar', 'containers.before.json', 'migrations.before.json', 'history.before.json')}
        self.manifest['backup_checksums'] = checksums
        self.persist()

    def guard_after(self):
        app = self.wait_healthy(self.sha)
        require(app['Image'] == self.manifest['image_id'], 'Running image ID differs from built artifact')
        require(not app['State'].get('OOMKilled') and app['RestartCount'] == 0, 'Candidate OOM/restart instability')
        before = json.loads((self.audit / 'containers.before.json').read_text())
        after = self.containers()
        verify_other_containers(before, after)
        require(runtime_fingerprint(before['sub2api']) == runtime_fingerprint(app), 'Application runtime/resource configuration changed')
        require(self.retention(app) == self.manifest['retention_days'], 'Usage retention policy changed')
        migration_before = json.loads((self.audit / 'migrations.before.json').read_text())
        migrations = self.migrations()
        verify_migrations(migration_before, migrations, self.manifest['migration_allowlist'])
        self.save('migrations.after.json', migrations)
        history = self.history(self.manifest['billing_cutoff_id'], 'history.after.json')
        result = verify_history(json.loads((self.audit / 'history.before.json').read_text()), history,
                                self.manifest['retention_days'], now())
        self.save('history.acceptance.json', result)
        self.save('containers.after.json', after)
        version = self.run(['docker', 'exec', 'sub2api', '/app/sub2api', '-version'])
        # The binary may report its version to stderr, retained in the command log.
        if self.sha not in version:
            version = self.run(['docker', 'exec', 'sub2api', 'sh', '-c', '/app/sub2api -version 2>&1'])
        require(self.sha in version, 'Binary version does not contain release SHA')
        return result

    def deploy(self, manual=False):
        self.load()
        require(self.manifest['phase'] == 'built', 'Deploy requires a freshly built manifest')
        identity = None
        if manual:
            self.manual_acceptance_authorization()
        else:
            identity = self.identity()  # MUST complete before backups or switching.
        before = self.containers()
        app = before['sub2api']
        require(app['Config']['Labels'].get('org.opencontainers.image.revision') == self.expected and
                app['Image'] == self.manifest['old_image_id'], 'Production changed; do not replace parallel releases')
        version = self.run(['docker', 'exec', 'sub2api', 'sh', '-c', '/app/sub2api -version 2>&1'])
        require(self.expected in version, 'Running binary differs from expected current SHA')
        require(self.inspect(self.manifest['image'])['Id'] == self.manifest['image_id'], 'Candidate image tag changed')
        self.run(['caddy', 'validate', '--config', '/etc/caddy/Caddyfile'])
        self.backup(before)
        old_migrations = json.loads((self.audit / 'migrations.before.json').read_text())
        baseline = set(self.source_migrations(self.expected))
        allowed = self.manifest['migration_allowlist']
        # Application rollback deliberately leaves append-only chat tables in place.
        # A repaired candidate may reuse them only with the identical pinned checksum.
        require(baseline <= old_migrations.keys() and old_migrations.keys() - baseline <= allowed.keys(),
                'Database baseline has missing or unrelated migrations')
        require(all(old_migrations[name] == allowed[name] for name in old_migrations.keys() & allowed.keys()),
                'Previously applied chat migration differs from candidate')
        self.manifest['preapplied_chat_migrations'] = sorted(old_migrations.keys() & allowed.keys())
        self.manifest['acceptance_mode'] = 'user_manual' if manual else 'automated'
        self.manifest['paid_acceptance'] = False
        if identity is not None:
            self.manifest['acceptance_user_id'] = identity['user_id']
        self.manifest['deploy_started_at'] = now().isoformat()
        self.manifest['phase'] = 'switching'
        self.persist()
        try:
            # Recheck immediately before mutation, not only before a potentially slow dump.
            require(self.inspect('sub2api')['Id'] == app['Id'], 'Application replaced during backup')
            require(COMPOSE.read_bytes() == (self.audit / 'compose.before.yml').read_bytes(), 'Compose changed during backup')
            if manual:
                self.manual_acceptance_authorization()
            else:
                self.identity()
            updated = replace_application_image(COMPOSE.read_text(), self.manifest['image'])
            self.manifest['candidate_compose_sha256'] = digest(updated.encode())
            self.persist()
            atomic_write(COMPOSE, updated)
            self.compose('config', '--quiet')
            self.compose('up', '-d', '--no-deps', 'sub2api')
            self.guard_after()
            self.health_and_assets()
            self.manifest['phase'] = ('deployed-awaiting-user-acceptance' if manual
                                      else 'deployed-awaiting-real-acceptance')
            self.persist()
        except Exception as error:
            self.save('rollback-trigger.json', {'phase': 'deploy', 'error': str(error), 'at': now().isoformat()})
            self.rollback()
            raise

    def fetch(self, url, *, resolve=None, expected=200):
        command = ['curl', '--silent', '--show-error', '--compressed', '--max-time', '25']
        if resolve:
            command += ['--resolve', resolve]
        header = self.audit / 'http.headers'
        body = self.audit / 'http.body'
        status = self.run(command + ['-D', str(header), '-o', str(body), '-w', '%{http_code}', url])
        require(status == str(expected), 'HTTP status differs from expected for ' + url)
        return body.read_bytes(), header.read_text().lower()

    def health_endpoints(self):
        origin = 'http://127.0.0.1:8080'
        for url, resolve in [(origin + '/health', None)] + [(f'https://{domain}/health',
                f'{domain}:443:127.0.0.1' if local else None) for domain in DOMAINS for local in (True, False)]:
            body, _ = self.fetch(url, resolve=resolve)
            require(json.loads(body).get('status') == 'ok', 'Health response is not ok')

    def health_and_assets(self):
        self.health_endpoints()
        origin = 'http://127.0.0.1:8080'
        # Validate every byte of the exact frontend build, including lazy route chunks.
        # index.html on the main SPA has nonce/settings injection, so check assets exactly
        # and let authenticated browser evidence prove route/login behavior.
        for host in (origin,) + tuple('https://' + domain for domain in DOMAINS):
            for route in ('/chat', '/chat-models'):
                body, headers = self.fetch(host + route)
                require(b'<html' in body.lower() and 'x-frame-options: deny' in headers, 'Main route/header regression')
            _, headers = self.fetch(host + '/chat-app', expected=308)
            require('location: /chat-app/' in headers, 'Chat slash redirect missing')
            chat_index, _ = self.fetch(host + '/chat-app/')
            if digest(chat_index) != self.manifest['assets']['chat-app/index.html']:
                atomic_write(self.audit / 'chat-index-mismatch.html', chat_index)
                raise RuntimeError('Chat root did not serve this release bundle: ' + host)
            for name, checksum in self.manifest['assets'].items():
                if not (name.startswith('chat-app/') or name.startswith('assets/')):
                    continue
                body, headers = self.fetch(host + '/' + urllib.parse.quote(name, safe='/'))
                require(digest(body) == checksum, 'Embedded asset differs from candidate: ' + name)
                if name.startswith('chat-app/'):
                    require('x-frame-options: sameorigin' in headers, 'Chat framing header missing')
                    csp = next((line for line in headers.splitlines() if line.startswith('content-security-policy:')), '')
                    require("frame-ancestors 'self'" in csp, 'Chat CSP must permit only same-origin ancestors')
            self.fetch(host + '/chat-app/static/missing-chat-release-proof.js', expected=404)
            self.fetch(host + '/api/v1/chat/presets', expected=401)
        self.save('health-static.json', {'sha': self.sha, 'checked_at': now().isoformat(),
                  'domains': DOMAINS, 'assets': len(self.manifest['assets']), 'paid_acceptance': False})

    def real_acceptance(self):
        evidence = self.evidence('real-acceptance.json')
        user_id = self.manifest['acceptance_user_id']
        require(evidence.get('user_id') == user_id and evidence.get('group_id') == 179 and evidence.get('model') == MODEL,
                'Real acceptance identity/group/model mismatch')
        require(evidence.get('real_paid_calls') is True and evidence.get('mock') is False,
                'Real paid evidence required; health/mock is not acceptance')
        require(timestamp(evidence['completed_at']) >= timestamp(self.manifest['deploy_started_at']) and
                timestamp(evidence['completed_at']) <= now(), 'Real acceptance evidence timestamp invalid')
        checks = evidence.get('checks', {})
        require(all(checks.get(name, {}).get('passed') is True for name in CHECKS), 'Mandatory authenticated browser scenario missing/failed')
        for name in CHECKS:
            self.artifact(checks[name]['evidence'])
        require(set(evidence.get('capabilities', {})) == {'image', 'document', 'search', 'read_url'}, 'Capability evidence incomplete')
        for capability, value in evidence['capabilities'].items():
            require(type(value.get('available')) is bool and value.get('result') == ('real_pass' if value['available'] else 'ui_disabled_backend_rejected'),
                    'Capability must be really tested or explicitly rejected: ' + capability)
            self.artifact(value['evidence'])
        request_ids = evidence.get('request_ids', [])
        tools = evidence.get('tool_request_ids', [])
        require(isinstance(request_ids, list) and 1 <= len(request_ids) <= 16 and len(set(request_ids)) == len(request_ids), 'Real request evidence invalid')
        require(isinstance(tools, list) and len(tools) <= 4 and set(tools) <= set(request_ids), 'Tool call budget exceeded')
        require(all(isinstance(value, str) and re.fullmatch(r'[A-Za-z0-9_.:-]{1,128}', value) for value in request_ids), 'Invalid request ID')
        rows = self.rows(f"SELECT json_build_object('id',u.id,'request_id',u.request_id,'group_id',u.group_id,'model',coalesce(u.requested_model,u.model),'billing_model',u.model,'actual_cost',u.actual_cost,'api_key_id',u.api_key_id,'key_user_id',k.user_id,'key_group_id',k.group_id)::text FROM usage_logs u LEFT JOIN api_keys k ON k.id=u.api_key_id WHERE u.user_id={user_id} AND u.id>{self.manifest['billing_cutoff_id']} ORDER BY u.id")
        require(len(rows) == len(request_ids) and {row['request_id'] for row in rows} == set(request_ids), 'DB usage does not match all real acceptance requests')
        require(all(row['group_id'] == 179 and (row['request_id'] in tools or row['model'] == MODEL) for row in rows), 'Unauthorized billing group/model')
        require(all(row['request_id'] not in tools or row['billing_model'] in ('deepseek-search', 'deepseek-fetch') for row in rows), 'Tool evidence does not refer to billed Search/Fetch calls')
        require(all(row['key_user_id'] == user_id and row['key_group_id'] == 179 for row in rows), 'Billed Key ownership/group mismatch')
        require(all(Decimal(str(row['actual_cost'])) >= 0 for row in rows), 'Negative charge cannot offset the acceptance budget')
        require(len(rows) - len(tools) <= 12, 'Model call budget exceeded')
        cost = sum((Decimal(str(row['actual_cost'])) for row in rows), Decimal(0))
        require(0 < cost <= 1 and cost == Decimal(str(evidence['actual_cost_usd'])), 'Actual paid cost missing, mismatched, or above USD 1')
        run_ids = evidence.get('run_ids', [])
        require(isinstance(run_ids, list) and len(run_ids) >= 4 and len(set(run_ids)) == len(run_ids) and
                all(re.fullmatch(r'[0-9a-fA-F-]{36}', value) for value in run_ids), 'Real cloud run evidence incomplete')
        runs = self.rows(f"SELECT json_build_object('id',id,'group_id',group_id,'model',model,'status',status,'gateway_request_ids',gateway_request_ids,'session_id',session_id)::text FROM chat_generation_runs WHERE user_id={user_id}")
        selected = [row for row in runs if str(row['id']) in run_ids]
        require(len(selected) == len(run_ids) and all(row['group_id'] == 179 and row['model'] == MODEL and row['status'] in ('done','stopped') for row in selected), 'Run ownership/status/model mismatch')
        linked = {value for row in selected for value in (row['gateway_request_ids'] or [])}
        require(set(request_ids) <= linked and any(row['status'] == 'stopped' for row in selected), 'Billing/run linkage or stopped partial run missing')
        require(sum(row['status'] == 'done' for row in selected) >= 3, 'Completed multi-turn/regeneration runs missing')
        marker = evidence.get('marker', '')
        require(isinstance(marker, str) and re.fullmatch(r'CHAT_OK_[A-Za-z0-9_-]{6,64}', marker), 'Random real-response marker missing')
        sessions = self.rows(f"SELECT json_build_object('id',id,'revision',revision,'has_marker',position('{marker}' in snapshot::text)>0)::text FROM chat_sessions WHERE user_id={user_id}")
        referenced = {str(row['session_id']) for row in selected}
        saved = [row for row in sessions if str(row['id']) in referenced]
        require(len(saved) == len(referenced) and all(row['revision'] > 0 for row in saved) and any(row['has_marker'] for row in saved),
                'Canonical server sessions/revisions/real response marker missing')
        require(self.sql(f"SELECT count(*) FROM chat_generation_runs WHERE user_id={user_id} AND status='running'") == '0', 'Acceptance left in-flight generation')
        # Test identity or every task Key must be disabled through normal product APIs.
        disabled = self.sql(f"SELECT count(*) FROM users WHERE id={user_id} AND (status<>'active' OR deleted_at IS NOT NULL)") == '1'
        active_keys = int(self.sql(f"SELECT count(*) FROM api_keys WHERE user_id={user_id} AND status='active' AND deleted_at IS NULL"))
        require(disabled or active_keys == 0, 'Acceptance identity/task Keys not disabled')
        self.save('real-acceptance-verified.json', {'sha': self.sha, 'user_id': user_id, 'request_count': len(rows),
                  'model_calls': len(rows) - len(tools), 'tool_calls': len(tools), 'actual_cost_usd': str(cost),
                  'run_ids': run_ids, 'verified_at': now().isoformat()})
        return {'request_count': len(rows), 'actual_cost_usd': str(cost)}

    def verify(self):
        self.load()
        require(self.manifest['phase'] == 'deployed-awaiting-real-acceptance', 'Verify requires deployed candidate')
        try:
            self.guard_after()
            self.health_and_assets()
            paid = self.real_acceptance()
            self.manifest.update(phase='verified', verified_at=now().isoformat(), paid_acceptance=paid)
            self.persist()
        except Exception as error:
            self.save('rollback-trigger.json', {'phase': 'verify', 'error': str(error), 'at': now().isoformat()})
            self.rollback()
            raise

    def rollback(self):
        if self.manifest is None:
            self.load()
        require('backup_checksums' in self.manifest, 'Rollback requires completed backups')
        current = self.inspect('sub2api')
        require(current['Image'] in (self.manifest['image_id'], self.manifest['old_image_id']),
                'Refusing to roll back an unrelated parallel release')
        allowed_compose = {self.manifest['backup_checksums']['compose.before.yml'],
                           self.manifest.get('candidate_compose_sha256'), self.manifest.get('rollback_compose_sha256')}
        require(digest(COMPOSE.read_bytes()) in allowed_compose, 'Compose changed outside this release; refusing overwrite')
        saved = self.audit / 'compose.before.yml'
        require(digest(saved.read_bytes()) == self.manifest['backup_checksums']['compose.before.yml'], 'Rollback Compose checksum mismatch')
        old = self.inspect(self.manifest['old_image_id'])
        require(old['Config']['Labels'].get('org.opencontainers.image.revision') == self.expected, 'Saved rollback image revision mismatch')
        # Pin immutable ID if an old tag was moved after deployment.
        text = replace_application_image(saved.read_text(), self.manifest['old_image_id'])
        self.manifest['rollback_compose_sha256'] = digest(text.encode())
        self.persist()
        atomic_write(COMPOSE, text)
        self.compose('config', '--quiet')
        self.compose('up', '-d', '--no-deps', 'sub2api')
        app = self.wait_healthy(self.expected)
        require(app['Image'] == self.manifest['old_image_id'], 'Rollback image ID mismatch')
        before = json.loads((self.audit / 'containers.before.json').read_text())
        verify_other_containers(before, self.containers())
        self.health_endpoints()
        # Database and chat files deliberately remain forward-only, including billed calls.
        self.manifest.update(phase='rolled-back', rolled_back_at=now().isoformat(), database_restored=False)
        self.persist()
        self.save('rollback.json', {'sha': self.sha, 'restored_image_id': self.manifest['old_image_id'],
                  'healthy': True, 'database_restored': False, 'chat_data_deleted': False})


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--phase', choices=('build', 'deploy', 'verify', 'rollback'), required=True)
    parser.add_argument('--sha', required=True)
    parser.add_argument('--expected-current-sha', required=True)
    parser.add_argument('--manual-acceptance', action='store_true',
                        help='Deploy with explicit user-owned paid acceptance; requires pinned authorization evidence')
    args = parser.parse_args()
    require(not args.manual_acceptance or args.phase == 'deploy',
            '--manual-acceptance is only valid for deploy, never a paid verify shortcut')
    require(all(re.fullmatch(r'[0-9a-f]{40}', value) for value in (args.sha, args.expected_current_sha)), 'Fixed lowercase 40-hex SHAs required')
    require(args.sha != args.expected_current_sha, 'Release SHA must differ from production baseline')
    require(sys.platform == 'linux' and COMPOSE.is_file() and Path('/etc/caddy/Caddyfile').is_file(), 'Run only on the designated HK deployment host')
    import fcntl
    os.umask(0o077)
    RELEASES.mkdir(mode=0o700, parents=True, exist_ok=True)
    # Global lock prevents concurrent candidates from racing the same Compose/DB guards.
    with (RELEASES / '.chat-release.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        release = Release(args.sha, args.expected_current_sha)
        try:
            if args.manual_acceptance:
                release.deploy(manual=True)
            else:
                getattr(release, args.phase)()
        except Exception as error:
            release.save('failure-' + str(time.time_ns()) + '.json', {'phase': args.phase, 'error': str(error), 'at': now().isoformat()})
            print('Chat release failed; inspect private HK evidence: ' + str(release.audit), file=sys.stderr)
            return 1
        print(json.dumps({'phase': args.phase, 'sha': args.sha, 'status': release.manifest['phase'],
                          'manifest': str(release.manifest_path)}))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
