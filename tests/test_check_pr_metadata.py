"""Tests del validador de metadata de PR (P2, issue #28)."""

from __future__ import annotations

import unittest

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "scripts"))

from check_pr_metadata import findings_for  # noqa: E402

VALID_BODY = """Delivery: auto-chain
Risk: baja

## Issue

Closes #28

## Qué y por qué

Los endurecimientos P2.

## Verificación

- Suite completa: `python3 -m unittest discover -s tests` → OK

## Checklist

- [x] CI verde en los jobs del stack tocado
- [x] Self-review del diff completo hecha
"""


class TestCheckPrMetadata(unittest.TestCase):
    def test_valid_body_has_no_faults(self):
        faults, _ = findings_for(VALID_BODY)
        self.assertEqual(faults, [])

    def test_missing_delivery_is_a_fault(self):
        body = VALID_BODY.replace("Delivery: auto-chain\n", "")
        faults, _ = findings_for(body)
        self.assertTrue(any("Delivery" in f for f in faults))

    def test_invalid_delivery_value_is_a_fault(self):
        faults, _ = findings_for(VALID_BODY.replace("Delivery: auto-chain", "Delivery: cuando-sea"))
        self.assertTrue(any("no es válido" in f and "Delivery" in f for f in faults))

    def test_missing_risk_is_a_fault(self):
        faults, _ = findings_for(VALID_BODY.replace("Risk: baja\n", ""))
        self.assertTrue(any("Risk" in f for f in faults))

    def test_high_risk_is_a_warning_not_a_fault(self):
        faults, warnings = findings_for(VALID_BODY.replace("Risk: baja", "Risk: alta"))
        self.assertEqual(faults, [])
        self.assertTrue(any("Risk: alta" in w for w in warnings))

    def test_missing_what_header_is_a_fault(self):
        body = VALID_BODY.replace("## Qué y por qué", "## Otra cosa")
        faults, _ = findings_for(body)
        self.assertTrue(any("encabezado" in f for f in faults))

    def test_missing_verification_header_is_a_fault(self):
        body = VALID_BODY.replace("## Verificación", "## Salida")
        faults, _ = findings_for(body)
        self.assertTrue(any("encabezado" in f for f in faults))

    def test_missing_ci_green_is_a_warning(self):
        body = VALID_BODY.replace("CI verde en los jobs del stack tocado", "verde")
        _, warnings = findings_for(body)
        self.assertTrue(any("CI verde" in w for w in warnings))


if __name__ == "__main__":
    unittest.main()
