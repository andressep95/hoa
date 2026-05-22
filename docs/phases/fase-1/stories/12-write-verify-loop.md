# Story 12 — Write-Verify Loop

## Como usuario
Quiero que cada escritura de código sea verificada automáticamente, y si falla, el agente reintente hasta 3 veces antes de hacer rollback.

## Criterios de Aceptación

- [ ] Cada `write_file` trigger un verify loop post-escritura
- [ ] Niveles de verificación: L0 (archivo existe), L1 (syntax), L2 (compila)
- [ ] L3 (lint) y L4 (tests) son configurables
- [ ] Si falla → feedback al agente con el error → retry (max 3)
- [ ] Si 3 retries fallan → `git checkout -- <archivo>` (rollback)
- [ ] Rollback registra el fallo para no repetirlo
- [ ] Tool `verify` permite ejecutar verificación manual

## Archivos a Crear

```
internal/harness/verify.go      # WriteVerifyLoop + verification levels
internal/tool/verify.go         # verify tool (manual trigger)
```

## Niveles

```go
type VerifyLevel int
const (
    L0_FileExists VerifyLevel = iota  // siempre
    L1_SyntaxValid                     // siempre
    L2_Compiles                        // si hay build tool
    L3_LintPasses                      // configurable
    L4_TestsPass                       // si hay tests afectados
    L5_SpecMet                         // en fase Verify final
)
```

## Definición de Done

- Escribir código con syntax error → agente recibe error → corrige → pasa
- 3 fallos seguidos → rollback automático + mensaje al usuario
- `go build ./...` se ejecuta como L2 check en proyectos Go
