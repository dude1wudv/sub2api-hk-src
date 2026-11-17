"""Fixed-SHA upstream synchronization release for the existing HK application.

Preserves live configuration, historical billing, shared containers and volumes.
Only the three reviewed migration byte sequences may be added, including the two
upstream 251 filenames that sort before the existing HK migrations. Health and
embedded-resource checks do not claim real paid-call or browser acceptance.
"""
import argparse
import json
import os
from pathlib import Path
import re
import sys
import time

import chat_release as shared


UPSTREAM_TAG_COMMIT = 'b0fa9f3e6eeab1968a53e1289b69daa4bb40ca2b'
UPSTREAM_MIGRATIONS = {
    '251_group_force_openai_ultrafast.sql': 'd78e605d4fc2038c57f620dff398022823a850ea8994b129b50774ac49e20af1',
    '251_group_usage_rollup_nonblocking_build.sql': '5b6e9b57f9b354a96c6870185c034db8adc7def944efcddf654e39cd97770718',
    '349_group_usage_rollup_retention_nonblocking.sql': '1241aac3633b14a9ecdf58878b5e8c704f428e0ac0941d0a78d7ceb1c54a42f2',
}


def upstream_migration_allowlist(before, candidate):
    shared.require(all(candidate.get(name) == checksum for name, checksum in before.items()),
                   'Existing migration source removed or changed')
    additions = {name: checksum for name, checksum in candidate.items() if name not in before}
    shared.require(additions == UPSTREAM_MIGRATIONS,
                   'Upstream synchronization requires exactly the reviewed migration bytes')
    return additions



def validate_ci(evidence, sha):
    shared.require(evidence.get('sha') == sha, 'CI evidence belongs to another SHA')
    runs = evidence.get('workflow_runs', [])
    for name in ('CI', 'Security Scan'):
        matches = [run for run in runs if run.get('name') == name and run.get('head_sha') == sha]
        shared.require(len(matches) == 1 and matches[0].get('status') == 'completed'
                       and matches[0].get('conclusion') == 'success',
                       'Successful fixed-SHA ' + name + ' evidence required')




class UpstreamRelease(shared.Release):
    allow_newly_visible_billing_rows = True

    def ci(self):
        evidence = self.evidence('ci.json')
        validate_ci(evidence, self.sha)
        self.save('ci-verified.json', evidence)

    def authorization(self):
        evidence = self.evidence('deployment-authorization.json')
        shared.require(evidence.get('deployment_authorized') is True
                       and evidence.get('upstream_merge_authorized') is True
                       and evidence.get('preserve_hk_billing') is True,
                       'Explicit approved upstream merge and deployment authorization required')
        self.artifact(evidence['authorization_evidence'])
        self.save('deployment-authorization-verified.json', evidence)

    def build(self):
        self.ci()
        original_allowlist = shared.migration_allowlist
        shared.migration_allowlist = upstream_migration_allowlist
        try:
            super().build()
        finally:
            shared.migration_allowlist = original_allowlist
        self.git('merge-base', '--is-ancestor', UPSTREAM_TAG_COMMIT, self.sha)
        wrapper = Path(__file__)
        shared.require(shared.file_digest(wrapper) == shared.file_digest(self.src / 'deploy' / wrapper.name),
                       'Upstream entrypoint differs from the fixed source commit')
        shared.require(self.manifest['migration_allowlist'] == UPSTREAM_MIGRATIONS, 'Migration allowlist mismatch')
        self.manifest['upstream_script_sha256'] = shared.file_digest(wrapper)
        self.persist()

    def load(self):
        super().load()
        shared.require(self.manifest.get('upstream_script_sha256') == shared.file_digest(Path(__file__)),
                       'Upstream entrypoint changed between phases')

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
                       'Previously applied upstream migration checksum differs')
        self.manifest.update(acceptance_mode='upstream_health_and_history', paid_acceptance=False,
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
            self.manifest.update(phase='deployed-upstream-verified',
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
        release = UpstreamRelease(args.sha, args.expected_current_sha)
        try:
            getattr(release, args.phase)()
        except Exception as error:
            release.save('failure-' + str(time.time_ns()) + '.json',
                         {'phase': args.phase, 'error': str(error), 'at': shared.now().isoformat()})
            print('Upstream release failed; inspect private HK evidence: ' + str(release.audit), file=sys.stderr)
            return 1
        print(json.dumps({'sha': release.sha, 'phase': release.manifest['phase'],
                          'image': release.manifest['image'], 'evidence': str(release.audit)}))
    return 0


if __name__ == '__main__':
    sys.exit(main())
