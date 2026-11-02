#!/usr/bin/env python3
"""Valida la metadata de un PR (P2, issue #28 — adaptado del ASF).

Comprueba, sobre el cuerpo del PR:
  1. Las líneas `Delivery:` y `Risk:` con valores válidos (vocabulario ODD
     del plan del architect: single-pr | auto-chain | ask-on-risk |
     exception-ok; riesgo: baja | media | alta).
  2. Los encabezados obligatorios de la plantilla (español): `## Qué` o
     `## Qué y por qué`, `## Verificación` o `## Cómo probarlo`.
  3. El checklist con las casillas de la fábrica (`CI verde`, `Self-review`).

Uso:
  python3 scripts/check_pr_metadata.py --body-file /tmp/pr.md
  python3 scripts/check_pr_metadata.py --pr 84        # vía `gh pr view`
Salida: hallazgos en español; exit 0 = metadata completa, exit 1 = faltas.
`--strict` promueve advertencias a faltas (para CI).
"""

from __future__ import annotations

import argparse
import subprocess
import sys

DELIVERY_VALUES = ("single-pr", "auto-chain", "ask-on-risk", "exception-ok")
RISK_VALUES = ("baja", "media", "alta")

# Encabezados de la plantilla: cada grupo tiene alternativas válidas (los PRs
# de producto usan `Qué y por qué`; algunos históricos usaron `Qué`).
REQUIRED_HEADER_GROUPS = (
    ("## qué", "## qué y por qué"),
    ("## verificación", "## cómo probarlo"),
)


def read_body(args: argparse.Namespace) -> str:
    if args.body_file:
        with open(args.body_file, encoding="utf-8") as handle:
            return handle.read()
    if args.pr:
        result = subprocess.run(
            ["gh", "pr", "view", str(args.pr), "--json", "body", "--jq", ".body"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout
    raise SystemExit("necesitas --body-file FILE o --pr N")


def findings_for(body: str) -> tuple[list[str], list[str]]:
    """Devuelve (faltas, advertencias) en español."""
    faults: list[str] = []
    warnings: list[str] = []
    lower = body.lower()

    delivery = next((line for line in body.splitlines() if line.strip().lower().startswith("delivery:")), None)
    if delivery is None:
        faults.append("falta la línea `Delivery: <estrategia>` (single-pr | auto-chain | ask-on-risk | exception-ok).")
    else:
        value = delivery.split(":", 1)[1].strip().lower()
        if value not in DELIVERY_VALUES:
            faults.append(
                f"`Delivery: {value}` no es válido — usa uno de: {', '.join(DELIVERY_VALUES)}."
            )

    risk = next((line for line in body.splitlines() if line.strip().lower().startswith("risk:")), None)
    if risk is None:
        faults.append("falta la línea `Risk: <baja|media|alta>`.")
    else:
        value = risk.split(":", 1)[1].strip().lower()
        if value not in RISK_VALUES:
            faults.append(f"`Risk: {value}` no es válido — usa uno de: {', '.join(RISK_VALUES)}.")
        elif value == "alta":
            warnings.append("`Risk: alta` — el PR exige revisión dual ciega (judgment-day) o nota del fundador.")

    for group in REQUIRED_HEADER_GROUPS:
        if not any(header in lower for header in group):
            faults.append(f"falta un encabezado obligatorio: una de {', '.join(repr(g) for g in group)}.")

    if "ci verde" not in lower:
        warnings.append("el checklist no menciona `CI verde` — sin CI verde no hay merge (guardarrail 6).")
    if "self-review" not in lower:
        warnings.append("el checklist no menciona `Self-review` — el diff completo se revisa a sí mismo antes de la ronda.")

    return faults, warnings


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--body-file", help="archivo con el cuerpo del PR")
    group.add_argument("--pr", type=int, help="número de PR (vía gh)")
    parser.add_argument("--strict", action="store_true", help="las advertencias también fallan")
    args = parser.parse_args()

    body = read_body(args)
    faults, warnings = findings_for(body)

    for fault in faults:
        print(f"FALTA: {fault}")
    for warning in warnings:
        print(f"AVISO: {warning}")
    if faults or (args.strict and warnings):
        print(f"check_pr_metadata: FALLA ({len(faults)} faltas, {len(warnings)} avisos)")
        return 1
    print(f"check_pr_metadata: OK ({len(warnings)} avisos)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
