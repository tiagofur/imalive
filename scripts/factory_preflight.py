#!/usr/bin/env python3
"""Read-only factory preflight check (imalive, adapted from software-factory-base).

Verifies git repository status, required factory governance files, and basic
toolchain availability without running test suites or mutating repository state.
"""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import sys
from pathlib import Path

REQUIRED_FILES = [
    "AGENTS.md",
    "README.md",
    "docs/README.md",
    "docs/agentes/fabrica-software.md",
    "docs/procesos/flujo-desarrollo.md",
    ".agents/skills/triage/SKILL.md",
    ".agents/skills/issue-to-pr/SKILL.md",
    ".agents/skills/pr-review/SKILL.md",
    ".agents/skills/ux-brief/SKILL.md",
    ".github/PULL_REQUEST_TEMPLATE.md",
    "contracts/openapi.yaml",
    "scripts/setup-labels.sh",
]


def run_git(root: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        capture_output=True,
        text=True,
    )


def main() -> int:
    parser = argparse.ArgumentParser(description="Read-only Software Factory preflight check.")
    parser.add_argument(
        "--root",
        type=Path,
        default=Path(__file__).resolve().parent.parent,
        help="Repository root directory.",
    )
    parser.add_argument(
        "--require-clean",
        action="store_true",
        help="Fail if the git working tree has uncommitted or untracked changes.",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Emit machine-readable JSON output.",
    )
    args = parser.parse_args()
    root = args.root.resolve()

    errors: list[str] = []

    git_check = run_git(root, "rev-parse", "--is-inside-work-tree")
    if git_check.returncode != 0 or git_check.stdout.strip() != "true":
        errors.append(f"Not a valid git repository: {root}")

    branch = ""
    head_sha = ""
    clean_tree = False
    if not errors:
        branch_res = run_git(root, "rev-parse", "--abbrev-ref", "HEAD")
        head_res = run_git(root, "rev-parse", "HEAD")
        status_res = run_git(root, "status", "--porcelain")
        branch = branch_res.stdout.strip() if branch_res.returncode == 0 else "unknown"
        head_sha = head_res.stdout.strip() if head_res.returncode == 0 else "unknown"
        clean_tree = status_res.returncode == 0 and status_res.stdout.strip() == ""
        if args.require_clean and not clean_tree:
            errors.append("Working tree is dirty (--require-clean specified).")

    missing_files: list[str] = []
    for rel_path in REQUIRED_FILES:
        if not (root / rel_path).is_file():
            missing_files.append(rel_path)

    if missing_files:
        errors.append(f"Missing required factory files: {', '.join(missing_files)}")

    tools = {
        "git": shutil.which("git") is not None,
        "python3": shutil.which("python3") is not None,
        "gh": shutil.which("gh") is not None,
        "go": shutil.which("go") is not None,
        "node": shutil.which("node") is not None,
    }

    status = "PREFLIGHT_OK_NOT_VERIFIED" if not errors else "PREFLIGHT_FAILED"
    payload = {
        "status": status,
        "root": str(root),
        "branch": branch,
        "head_sha": head_sha,
        "clean_working_tree": clean_tree,
        "missing_files": missing_files,
        "tools": tools,
        "errors": errors,
    }

    if args.json:
        print(json.dumps(payload, indent=2))
    else:
        print(f"[{status}] branch={branch} head={head_sha[:7] if head_sha else 'none'} clean={clean_tree}")
        for err in errors:
            print(f"  ERROR: {err}", file=sys.stderr)

    return 0 if not errors else 1


if __name__ == "__main__":
    sys.exit(main())
