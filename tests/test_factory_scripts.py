import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
SCRIPTS = REPO_ROOT / "scripts"


def git(cwd: Path, *args: str) -> str:
    return subprocess.run(
        ["git", *args], cwd=cwd, check=True, capture_output=True, text=True
    ).stdout.strip()


class PreflightTest(unittest.TestCase):
    def test_real_repo_passes_without_asf_only_root_files(self) -> None:
        # imalive has no root Makefile or feature_list.json: preflight must not require them.
        res = subprocess.run(
            [sys.executable, str(SCRIPTS / "factory_preflight.py"), "--json"],
            capture_output=True,
            text=True,
        )
        self.assertEqual(res.returncode, 0, msg=res.stdout + res.stderr)
        payload = json.loads(res.stdout)
        self.assertEqual(payload["status"], "PREFLIGHT_OK_NOT_VERIFIED")
        self.assertNotIn("Makefile", payload["missing_files"])

    def test_missing_factory_skill_fails_and_dirty_tree_fails_with_require_clean(self) -> None:
        tmp = Path(tempfile.mkdtemp(prefix="imalive-pre-"))
        try:
            git(tmp, "init", "-q", "-b", "main")
            git(tmp, "config", "user.email", "t@example.com")
            git(tmp, "config", "user.name", "T")
            (tmp / "AGENTS.md").write_text("x\n", encoding="utf-8")
            (tmp / "tracked.txt").write_text("x\n", encoding="utf-8")
            git(tmp, "add", ".")
            git(tmp, "commit", "-q", "-m", "chore: base")
            res = subprocess.run(
                [sys.executable, str(SCRIPTS / "factory_preflight.py"), "--root", str(tmp)],
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(res.returncode, 0)
            (tmp / "dirty.txt").write_text("d\n", encoding="utf-8")
            res_dirty = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPTS / "factory_preflight.py"),
                    "--root",
                    str(tmp),
                    "--require-clean",
                ],
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(res_dirty.returncode, 0)
        finally:
            shutil.rmtree(tmp, ignore_errors=True)


class HandoffTest(unittest.TestCase):
    def _bundle_for(self, rel_path: str) -> dict:
        tmp = Path(tempfile.mkdtemp(prefix="imalive-handoff-"))
        try:
            git(tmp, "init", "-q", "-b", "main")
            git(tmp, "config", "user.email", "t@example.com")
            git(tmp, "config", "user.name", "T")
            (tmp / "README.md").write_text("x\n", encoding="utf-8")
            git(tmp, "add", ".")
            git(tmp, "commit", "-q", "-m", "chore: base")
            base = git(tmp, "rev-parse", "HEAD")
            target = tmp / rel_path
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text("change\n", encoding="utf-8")
            git(tmp, "add", ".")
            git(tmp, "commit", "-q", "-m", "feat: change")
            head = git(tmp, "rev-parse", "HEAD")
            out = tmp / "bundle.json"
            res = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPTS / "factory_handoff.py"),
                    "--root",
                    str(tmp),
                    "--base",
                    base,
                    "--head",
                    head,
                    "--issue",
                    "27",
                    "--output-json",
                    str(out),
                ],
                capture_output=True,
                text=True,
            )
            self.assertEqual(res.returncode, 0, msg=res.stderr or res.stdout)
            return json.loads(out.read_text(encoding="utf-8"))
        finally:
            shutil.rmtree(tmp, ignore_errors=True)

    def test_escalation_engine_path_requires_judgment_day(self) -> None:
        bundle = self._bundle_for("backend/internal/escalation/engine.go")
        self.assertTrue(bundle["judgment_day_required"])

    def test_otp_auth_path_requires_judgment_day(self) -> None:
        bundle = self._bundle_for("backend/internal/auth/otp.go")
        self.assertTrue(bundle["judgment_day_required"])

    def test_plain_docs_change_does_not_require_judgment_day_and_has_no_chat_fields(self) -> None:
        bundle = self._bundle_for("docs/notes.md")
        self.assertFalse(bundle["judgment_day_required"])
        self.assertEqual(len(bundle["scope_sha256"]), 64)
        self.assertNotIn("chat_history", bundle)
        self.assertEqual(bundle["context_isolation"], "STATELESS_CLEAN_CONTEXT_NO_CHAT_HISTORY")


class LabelsScriptTest(unittest.TestCase):
    def test_setup_labels_declares_contract_break_and_judgment_day(self) -> None:
        text = (SCRIPTS / "setup-labels.sh").read_text(encoding="utf-8")
        for label in ("contract-break", "judgment-day", "size:exception"):
            self.assertIn(f'"{label}"', text)
        # Existing labels must keep working (no forced migration).
        for label in ("status:nuevo", "status:bloqueado", "type:feature"):
            self.assertIn(f'"{label}"', text)


if __name__ == "__main__":
    unittest.main()
