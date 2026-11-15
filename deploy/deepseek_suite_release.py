"""Fixed-SHA HK suite release, reusing the existing application-only safeguards.

All credentials, backups and raw evidence remain in the private HK release
directory. This entrypoint owns application switching; account enrollment and
bounded provider acceptance follow successful application verification.
"""
import argparse
import json
import os
from pathlib import Path
import re
import sys
import time

import chat_release as shared


SUITE_MIGRATIONS = {
    '345_group_usage_rollup_retention_guard.sql': '243fd3ffa69e94665407b2b53d948db539def0e826d0868156841325ff5bab61',
    '346_widen_proxy_password.sql': 'd82059ef30644e7794f7c8dd9cade1efa992d80ea70a5ecb46de6c454faa020c',
    '347_deepseek_suite.sql': 'f9a72022994fca71350e707da12aebffe8a80c7028a5fe528b028eac8614642b',
    '348_external_quality_enterprise_source.sql': 'cf1afac9ed7772bfe32e2f5dd12e33c9bcd1dd824cc8578f34406a80af1db904',
}


def validate_ci(evidence, sha):
    shared.require(evidence.get('sha') == sha, 'CI evidence belongs to another SHA')
    runs = evidence.get('workflow_runs', [])
    for name in ('CI', 'Security Scan'):
        matches = [run for run in runs if run.get('name') == name and run.get('head_sha') == sha]
        shared.require(len(matches) == 1 and matches[0].get('status') == 'completed'
                       and matches[0].get('conclusion') == 'success',
                       'Successful fixed-SHA ' + name + ' evidence required')


def validate_suite_migrations(migrations):
    # A follow-up application fix may start from the already migrated suite.
    shared.require(all(name in SUITE_MIGRATIONS and checksum == SUITE_MIGRATIONS[name]
                       for name, checksum in migrations.items()),
                   'Suite release may only append the approved migration bytes')


class SuiteRelease(shared.Release):
    def ci(self):
        evidence = self.evidence('ci.json')
        validate_ci(evidence, self.sha)
        self.save('ci-verified.json', evidence)

    def authorization(self):
        evidence = self.evidence('deployment-authorization.json')
        shared.require(evidence.get('deployment_authorized') is True
                       and evidence.get('enrollment_authorized') is True
                       and evidence.get('quality_sync_update_authorized') is True,
                       'Explicit combined deployment authorization required')
        self.artifact(evidence['authorization_evidence'])
        self.save('deployment-authorization-verified.json', evidence)

    def build(self):
        self.ci()
        shared.APPROVED_AUXILIARY_MIGRATIONS.update(SUITE_MIGRATIONS)
        super().build()
        wrapper = Path(__file__)
        shared.require(shared.file_digest(wrapper) == shared.file_digest(self.src / 'deploy' / wrapper.name),
                       'Suite entrypoint differs from the fixed source commit')
        validate_suite_migrations(self.manifest['migration_allowlist'])
        self.manifest['suite_script_sha256'] = shared.file_digest(wrapper)
        self.persist()

    def load(self):
        super().load()
        shared.require(self.manifest.get('suite_script_sha256') == shared.file_digest(Path(__file__)),
                       'Suite entrypoint changed between phases')

    def deploy(self):
        self.load()
        shared.require(self.manifest['phase'] == 'built', 'Deploy requires a freshly built manifest')
        self.ci()
        self.authorization()
        before = self.containers()
        app = before['sub2api']
        shared.require(app['Config']['Labels'].get('org.opencontainers.image.revision') == self.expected
                       and app['Image'] == self.manifest['old_image_id'], 'Production baseline changed')
        version = self.run(['docker', 'exec', 'sub2api', 'sh', '-c', '/app/sub2api -version 2>&1'])
        shared.require(self.expected in version, 'Running binary differs from the baseline')
        shared.require(self.inspect(self.manifest['image'])['Id'] == self.manifest['image_id'],
                       'Candidate image tag changed')
        self.run(['caddy', 'validate', '--config', '/etc/caddy/Caddyfile'])
        self.backup(before)
        old = json.loads((self.audit / 'migrations.before.json').read_text())
        baseline = self.source_migrations(self.expected)
        allowed = self.manifest['migration_allowlist']
        shared.require(set(baseline) <= old.keys() and old.keys() - baseline.keys() <= allowed.keys(),
                       'Database baseline has missing or unrelated migrations')
        shared.require(all(old[name] == allowed[name] for name in old.keys() & allowed.keys()),
                       'Previously applied suite migration checksum differs')
        self.manifest.update(acceptance_mode='suite', paid_acceptance=False,
                             deploy_started_at=shared.now().isoformat(), phase='switching')
        self.persist()
        try:
            shared.require(self.inspect('sub2api')['Id'] == app['Id'], 'Application replaced during backup')
            shared.require(shared.COMPOSE.read_bytes() == (self.audit / 'compose.before.yml').read_bytes(),
                           'Compose changed during backup')
            updated = shared.replace_application_image(shared.COMPOSE.read_text(), self.manifest['image'])
            self.manifest['candidate_compose_sha256'] = shared.digest(updated.encode())
            self.persist()
            shared.atomic_write(shared.COMPOSE, updated)
            self.compose('config', '--quiet')
            self.compose('up', '-d', '--no-deps', 'sub2api')
            self.guard_after()
            self.health_and_assets()
            self.manifest.update(phase='deployed-awaiting-suite-acceptance',
                                 application_verified_at=shared.now().isoformat())
            self.persist()
        except Exception as error:
            self.save('rollback-trigger.json', {'phase': 'deploy', 'error': str(error),
                                               'at': shared.now().isoformat()})
            self.rollback()
            raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--phase', choices=('build', 'deploy', 'rollback'), required=True)
    parser.add_argument('--sha', required=True)
    parser.add_argument('--expected-current-sha', required=True)
    args = parser.parse_args()
    shared.require(all(re.fullmatch(r'[0-9a-f]{40}', value)
                       for value in (args.sha, args.expected_current_sha))
                   and args.sha != args.expected_current_sha, 'Distinct fixed 40-hex SHAs required')
    shared.require(sys.platform == 'linux' and shared.COMPOSE.is_file()
                   and Path('/etc/caddy/Caddyfile').is_file(), 'Run only on the HK deployment host')
    import fcntl
    os.umask(0o077)
    shared.RELEASES.mkdir(mode=0o700, parents=True, exist_ok=True)
    # Share the existing chat release lock, including across different candidates.
    with (shared.RELEASES / '.chat-release.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        release = SuiteRelease(args.sha, args.expected_current_sha)
        try:
            getattr(release, args.phase)()
        except Exception as error:
            release.save('failure-' + str(time.time_ns()) + '.json',
                         {'phase': args.phase, 'error': str(error), 'at': shared.now().isoformat()})
            print('Suite release failed; inspect private HK evidence: ' + str(release.audit), file=sys.stderr)
            return 1
        print(json.dumps({'sha': release.sha, 'phase': release.manifest['phase'],
                          'image': release.manifest['image'], 'evidence': str(release.audit)}))
    return 0


if __name__ == '__main__':
    sys.exit(main())
