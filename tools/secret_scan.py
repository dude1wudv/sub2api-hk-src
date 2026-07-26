#!/usr/bin/env python3
"""Fail CI when tracked source contains an obvious credential."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

PATTERNS = (
    ("private key", re.compile(r"-----BEGIN [A-Z ]*PRIVATE KEY-----")),
    ("AWS access key", re.compile(r"\b(?:AKIA|ASIA)[A-Z0-9]{16}\b")),
    ("GitHub token", re.compile(r"\bgh[pousr]_[A-Za-z0-9_]{30,}\b")),
    ("GitLab token", re.compile(r"\bglpat-[A-Za-z0-9_-]{20,}\b")),
    ("Slack token", re.compile(r"\bxox[baprs]-[A-Za-z0-9-]{20,}\b")),
    ("Stripe live secret", re.compile(r"\bsk_live_[A-Za-z0-9]{20,}\b")),
)
SKIPPED_SUFFIXES = {".md", ".png", ".jpg", ".jpeg", ".svg", ".woff", ".woff2", ".ttf", ".map"}
SKIPPED_NAMES = {"pnpm-lock.yaml", "go.sum"}


def tracked_files() -> list[Path]:
    result = subprocess.run(
        ["git", "ls-files", "-z"], check=True, stdout=subprocess.PIPE
    )
    return [Path(path.decode()) for path in result.stdout.split(b"\0") if path]


def main() -> int:
    findings: list[str] = []
    for path in tracked_files():
        is_fixture = (
            path.name.endswith("_test.go")
            or ".spec." in path.name
            or "tests" in path.parts
            or "testdata" in path.parts
        )
        if (
            not path.is_file()
            or is_fixture
            or path.name in SKIPPED_NAMES
            or path.suffix.lower() in SKIPPED_SUFFIXES
        ):
            continue
        try:
            content = path.read_text(encoding="utf-8")
        except UnicodeDecodeError:
            continue
        for label, pattern in PATTERNS:
            if pattern.search(content):
                findings.append(f"{path}: possible {label}")

    if findings:
        print("secret scan failed:", *findings, sep="\n", file=sys.stderr)
        return 1
    print("secret scan passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())