#!/usr/bin/env bash
# Labels de la fábrica imalive. Idempotente: ejecutar de nuevo es seguro.
set -euo pipefail

mk() {
  gh label create "$1" --color "$2" --description "$3" 2>/dev/null \
    || gh label edit "$1" --color "$2" --description "$3"
}

# Tipo
mk "type:feature" "0e8a16" "Nueva funcionalidad de producto o fábrica"
mk "type:bug"     "d73a4a" "Algo no funciona como debe"
mk "type:task"    "0052cc" "Trabajo de infraestructura o fábrica"
mk "type:chore"   "c5def5" "Mantenimiento menor"

# Estado
mk "status:nuevo"       "ededed" "Sin procesar; espera triage"
mk "status:ready"       "1d76db" "Pasa DoR; implementable sin preguntas"
mk "status:in-progress" "fbca04" "Rama abierta, trabajo en curso"
mk "status:in-review"   "5319e7" "PR abierto esperando revisión"
mk "status:bloqueado"   "b60205" "Espera decisión o credencial humana"

# Prioridad y tamaño
mk "needs-human" "e99695" "Requiere acción humana (dinero, cuentas, legal)"
mk "ux:shape" "c7def8" "Cambio UX considerable: brief del designer antes del plan"
mk "priority:p0" "d93f0b" "Urgente: bloquea el loop del producto"
mk "priority:p1" "f9d0c4" "Esta semana"
mk "priority:p2" "fef2c0" "Cuando quepa"
mk "size:xs"     "c2e0c6" "Menos de 1h"
mk "size:s"      "c2e0c6" "Menos de 4h"
mk "size:m"      "bfd4f2" "Menos de 2 días; plan del architect si es feature"
mk "size:l"      "bfd4f2" "Más de 2 días; partir o chained-pr"

echo "Labels listos."
