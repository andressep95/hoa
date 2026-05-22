# Story 07 — Write File + Diff Approval

## Como usuario
Quiero que el agente pueda escribir archivos, pero que me muestre un diff y pida aprobación antes de aplicar cambios.

## Criterios de Aceptación

- [ ] Tool `write_file`: escribe contenido a un path
- [ ] Antes de escribir, computa diff unificado contra contenido actual
- [ ] Modal de aprobación muestra el diff y espera y/n del usuario
- [ ] Si aprueba → escribe. Si rechaza → tool result "user denied"
- [ ] `agent.Confirm` es un callback inyectable (para testing: auto-approve)
- [ ] Archivos nuevos muestran diff como "todo verde" (creación)

## Archivos a Crear/Modificar

```
internal/tool/writefile.go     # write_file tool
internal/agent/diff.go         # buildWriteDiff helper
internal/ui/approval.go        # Modal de aprobación en TUI
```

## Definición de Done

- "crea un archivo hello.txt con un haiku" → muestra diff → apruebo → archivo creado
- "escribe X en main.go" → muestra diff con cambios → rechazo → no se modifica
- El modelo recibe "user denied" y puede intentar otra cosa
