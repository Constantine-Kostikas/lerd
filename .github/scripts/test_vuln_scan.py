import contextlib
import io
import os
import subprocess
import tempfile
import unittest
from unittest import mock

import vuln_scan


def match(kind, severity, name='pkg'):
    return {'artifact': {'type': kind, 'name': name, 'version': '1.0'},
            'vulnerability': {'id': 'CVE-1', 'severity': severity, 'fix': {'versions': ['1.1']}}}


class RebuildDecision(unittest.TestCase):
    def test_keeps_only_high_and_critical(self):
        report = {'matches': [match('apk', 'Medium'), match('apk', 'High'), match('npm', 'Critical'), match('apk', 'Low')]}
        self.assertEqual([m['vulnerability']['severity'] for m in vuln_scan.severe(report)], ['High', 'Critical'])

    def test_only_an_alpine_package_asks_for_a_rebuild(self):
        self.assertTrue(vuln_scan.needs_rebuild([match('npm', 'High'), match('apk', 'High')]))
        self.assertFalse(vuln_scan.needs_rebuild([match('npm', 'Critical'), match('rust-crate', 'High')]))
        self.assertFalse(vuln_scan.needs_rebuild([]))


class Main(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.dir.cleanup)
        root = self.dir.name
        os.makedirs(os.path.join(root, os.path.dirname(vuln_scan.RECIPE)))
        with open(os.path.join(root, vuln_scan.RECIPE), 'w') as f:
            f.write('FROM php:{{.Version}}-fpm-alpine\n')
        self.summary = os.path.join(root, 'summary')
        self.output = os.path.join(root, 'output')
        env = {'GITHUB_STEP_SUMMARY': self.summary, 'GITHUB_OUTPUT': self.output}
        self.env = mock.patch.dict(os.environ, env)
        self.env.start()
        self.addCleanup(self.env.stop)
        cwd = os.getcwd()
        os.chdir(root)
        self.addCleanup(os.chdir, cwd)
        # The script prints workflow commands, which GitHub would raise as
        # annotations on a passing run.
        quiet = contextlib.redirect_stdout(io.StringIO())
        quiet.__enter__()
        self.addCleanup(quiet.__exit__, None, None, None)

    def stale(self):
        with open(self.output) as f:
            return f.read().strip()

    def test_a_failed_scan_keeps_the_other_rebuilds(self):
        def scan(image):
            if 'php82' in image:
                raise subprocess.CalledProcessError(1, 'grype')
            return {'matches': [match('apk', 'High')] if 'php85' in image else []}
        self.assertEqual(vuln_scan.main(scan, lambda image: ['Upgrading brotli-libs']), 1)
        self.assertEqual(self.stale(), 'stale=8.5')

    def test_nothing_newer_to_install_rebuilds_nothing(self):
        def scan(image):
            return {'matches': [match('apk', 'High')]}
        def upgrades(image):
            return ['(1/1) Upgrading brotli-libs (1.2.0-r1 -> 1.2.1-r0)'] if 'php83' in image else []
        self.assertEqual(vuln_scan.main(scan, upgrades), 0)
        self.assertEqual(self.stale(), 'stale=8.3')

    def test_clean_images_rebuild_nothing(self):
        self.assertEqual(vuln_scan.main(lambda image: {'matches': []}, lambda image: self.fail('asked for upgrades of a clean image')), 0)
        self.assertEqual(self.stale(), 'stale=')


if __name__ == '__main__':
    unittest.main()
