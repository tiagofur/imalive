#!/usr/bin/env python3
"""Stateless Clean-Context Review Handoff Generator (Checkpoint C4).

Builds a deterministic, SHA-256-pinned review bundle for the independent
`reviewer` skill. Crucially, this script NEVER includes the implementer's chat
history or conversational rationale—preventing reviewer rationalization bias.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

# Triggers mirror the "Revisión dual automática" list in .agents/skills/pr-review/SKILL.md:
# escalation engine, auth/OTP, payments/entitlements, contact notifications, personal data,
# plus factory/contract surfaces.
HIGH_RISK_PATH_PREFIXES = (
    "contracts/",
    "scripts/",
    ".github/workflows/",
    "docs/arquitectura/adr/",
)

HIGH_RISK_KEYWORDS = (
    "escalation",
    "escalada",
    "auth",
    "otp",
    "payment",
    "entitlement",
    "notification",
    "notif",
    "migration",
    "secret",
    "pii",
    "location",
)


def git_output(root: Path, *args: str) -> str:
    res = subprocess.run(
        ["git", *args],
        cwd=root,
        capture_output=True,
        text=True,
        check=True,
    )
    return res.stdout


def detect_high_risk_triggers(changed_files: list[str]) -> list[str]:
    triggers: list[str] = []
    for path in changed_files:
        lower = path.lower()
        if any(path.startswith(prefix) for prefix in HIGH_RISK_PATH_PREFIXES):
            triggers.append(f"sensitive-surface:{path}")
        elif any(keyword in lower for keyword in HIGH_RISK_KEYWORDS):
            triggers.append(f"high-risk-keyword:{path}")
    return sorted(set(triggers))


def main() -> int:
    parser = argparse.ArgumentParser(description="Build stateless clean-context reviewer handoff packet.")
    parser.add_argument(
        "--root",
        type=Path,
        default=Path(__file__).resolve().parent.parent,
        help="Repository root directory.",
    )
    parser.add_argument("--base", required=True, help="Base git ref or commit SHA.")
    parser.add_argument("--head", default="HEAD", help="Head git ref or commit SHA.")
    parser.add_argument("--issue", type=int, required=True, help="Authorized GitHub Issue number.")
    parser.add_argument(
        "--acceptance",
        default="See linked GitHub Issue acceptance criteria and odd/tasks spec.",
        help="Verbatim Issue acceptance criteria summary.",
    )
    parser.add_argument(
        "--verification-note",
        default="Run V0/V1/V2 checks per CHECKPOINTS.md.",
        help="Observed V0/V1/V2 verification command output summary.",
    )
    parser.add_argument("--output-json", type=Path, help="Optional path to write JSON bundle.")
    parser.add_argument("--output-md", type=Path, help="Optional path to write Markdown bundle.")
    args = parser.parse_args()

    root = args.root.resolve()
    try:
        base_sha = git_output(root, "rev-parse", args.base).strip()
        head_sha = git_output(root, "rev-parse", args.head).strip()
        names_raw = git_output(root, "diff", "--name-only", f"{base_sha}...{head_sha}")
        diff_stat = git_output(root, "diff", "--stat", f"{base_sha}...{head_sha}").strip()
        diff_patch = git_output(root, "diff", f"{base_sha}...{head_sha}")
    except subprocess.CalledProcessError as exc:
        print(f"ERROR running git for handoff: {exc.stderr or exc}", file=sys.stderr)
        return 1

    changed_files = [line.strip() for line in names_raw.splitlines() if line.strip()]
    high_risk_triggers = detect_high_risk_triggers(changed_files)
    judgment_day_required = len(high_risk_triggers) > 0

    canonical_scope = json.dumps(
        {
            "issue": args.issue,
            "base_sha": base_sha,
            "head_sha": head_sha,
            "changed_files": changed_files,
            "diff_sha256": hashlib.sha256(diff_patch.encode("utf-8")).hexdigest(),
        },
        sort_keys=True,
    )
    scope_sha256 = hashlib.sha256(canonical_scope.encode("utf-8")).hexdigest()

    bundle = {
        "schema": "software-factory.clean-context-handoff.v1",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "context_isolation": "STATELESS_CLEAN_CONTEXT_NO_CHAT_HISTORY",
        "issue": args.issue,
        "base_sha": base_sha,
        "head_sha": head_sha,
        "scope_sha256": scope_sha256,
        "changed_files": changed_files,
        "diff_stat": diff_stat,
        "acceptance_criteria": args.acceptance,
        "verification_evidence": args.verification_note,
        "high_risk_triggers": high_risk_triggers,
        "judgment_day_required": judgment_day_required,
        "diff": diff_patch,
    }

    md_lines = [
        f"# Clean-Context Review Bundle — Issue #{args.issue}",
        "",
        "> **Context Isolation Notice**: This packet is generated deterministically from git state.",
        "> Do NOT read the implementer's conversation history. Evaluate strictly against the",
        "> Issue acceptance criteria, Hierarchy of Truth (`AGENTS.md`), and `CHECKPOINTS.md`.",
        "",
        "## Pinned Identity",
        f"- **Issue**: `#{args.issue}`",
        f"- **Base SHA**: `{base_sha}`",
        f"- **Head SHA**: `{head_sha}`",
        f"- **Scope Digest (SHA-256)**: `{scope_sha256}`",
        f"- **Blind Dual Review (`judgment-day`) Required**: `{judgment_day_required}`",
        "",
        "## High-Risk Triggers",
    ]
    if high_risk_triggers:
        md_lines.extend([f"- `{t}`" for t in high_risk_triggers])
    else:
        md_lines.append("- None detected")

    md_lines.extend(
        [
            "",
            "## Acceptance Criteria",
            args.acceptance,
            "",
            "## Observed Verification Evidence (V0/V1/V2)",
            "```text",
            args.verification_note,
            "```",
            "",
            "## Changed Files",
            *[f"- `{p}`" for p in changed_files],
            "",
            "## Diff Stat",
            "```text",
            diff_stat or "(no file changes)",
            "```",
        ]
    )
    md_content = "\n".join(md_lines) + "\n"

    if args.output_json:
        args.output_json.parent.mkdir(parents=True, exist_ok=True)
        args.output_json.write_text(json.dumps(bundle, indent=2) + "\n", encoding="utf-8")

    if args.output_md:
        args.output_md.parent.mkdir(parents=True, exist_ok=True)
        args.output_md.write_text(md_content, encoding="utf-8")

    if not args.output_json and not args.output_md:
        print(md_content)
    else:
        print(
            f"HANDOFF_READY issue=#{args.issue} base={base_sha[:7]} head={head_sha[:7]} "
            f"scope_sha256={scope_sha256[:12]} judgment_day={judgment_day_required}"
        )

    return 0


if __name__ == "__main__":
    sys.exit(main())
