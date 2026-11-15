import importlib.util
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

DEPLOY = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(DEPLOY))
spec = importlib.util.spec_from_file_location('deepseek_suite_release', DEPLOY / 'deepseek_suite_release.py')
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class SuiteReleaseTests(unittest.TestCase):
    def test_followup_allows_existing_suite_schema_without_new_migrations(self):
        release.validate_suite_migrations({})
        release.validate_suite_migrations(release.SUITE_MIGRATIONS)
        for invalid in ({'349_unapproved.sql': 'new'},
                        {'348_external_quality_enterprise_source.sql': 'changed'}):
            with self.assertRaises(RuntimeError):
                release.validate_suite_migrations(invalid)

    def test_combined_migrations_are_exact_and_append_only(self):
        old = {'344_chat_auxiliary.sql': 'old'}
        additions = {
            name: release.shared.migration_checksum((DEPLOY.parent / 'backend/migrations' / name).read_bytes())
            for name in release.SUITE_MIGRATIONS
        }
        self.assertEqual(additions, release.SUITE_MIGRATIONS)
        with patch.dict(release.shared.APPROVED_AUXILIARY_MIGRATIONS, additions):
            self.assertEqual(release.shared.migration_allowlist(old, {**old, **additions}), additions)
            for candidate in ({**old, **additions, '349_unapproved.sql': 'new'},
                              {**old, **additions, '348_external_quality_enterprise_source.sql': 'changed'},
                              {**old, **additions, '344_chat_auxiliary.sql': 'changed'}):
                with self.assertRaises(RuntimeError):
                    release.shared.migration_allowlist(old, candidate)

    def test_ci_gate_requires_both_successful_exact_sha_workflows(self):
        sha = 'a' * 40
        runs = [{'name': name, 'head_sha': sha, 'status': 'completed', 'conclusion': 'success'}
                for name in ('CI', 'Security Scan')]
        release.validate_ci({'sha': sha, 'workflow_runs': runs}, sha)
        for invalid in ([], runs[:1], [runs[0], {**runs[1], 'head_sha': 'b' * 40}],
                        [runs[0], {**runs[1], 'status': 'in_progress'}],
                        [{**runs[0], 'conclusion': 'failure'}, runs[1]], runs + [runs[0]]):
            with self.assertRaises(RuntimeError):
                release.validate_ci({'sha': sha, 'workflow_runs': invalid}, sha)
        with self.assertRaises(RuntimeError):
            release.validate_ci({'sha': 'b' * 40, 'workflow_runs': runs}, sha)


if __name__ == '__main__':
    unittest.main()
