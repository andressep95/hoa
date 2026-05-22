# Story 11 — SDD Engine (Spec-Driven Development)

## Como usuario
Quiero que el agente NO salte directo al código — debe pasar por fases obligatorias de planeamiento antes de escribir.

## Criterios de Aceptación

- [ ] State machine con fases: Proposal → Spec → Design → Task → Apply → Verify
- [ ] Cada fase produce un artefacto en `.hoa/sdd/current/`
- [ ] Gates de avance: no se puede pasar a Apply sin Design aprobado
- [ ] Tool `plan` inicia/avanza el flujo SDD
- [ ] El usuario puede aprobar/rechazar cada artefacto
- [ ] `/sdd` muestra el estado actual del flujo
- [ ] Artefactos se archivan en `history/` post-commit

## Archivos a Crear

```
internal/harness/sdd.go     # SDDEngine state machine + gates
internal/tool/plan.go       # plan tool (trigger SDD phases)
```

## Fases y Artefactos

| Fase | Artefacto | Gate |
|------|-----------|------|
| Proposal | `proposal.md` | Usuario aprueba |
| Spec | `spec.md` | Harness valida completitud |
| Design | `design.md` | Harness valida coherencia con spec |
| Task | `tasks.json` | Cada tarea es verificable |
| Apply | Código | Cada write pasa verify loop |
| Verify | Validación | Tests + spec cumplida |

## Definición de Done

- "implementa auth" → el agente genera proposal.md primero, no código
- Intentar escribir código sin spec → el harness lo bloquea
- `/sdd` muestra "fase actual: Design, 2/6 completadas"
