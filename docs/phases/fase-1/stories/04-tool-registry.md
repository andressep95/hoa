# Story 04 — Tool Registry + Tools Read-Only ✅

## Como usuario
Quiero que el agente pueda leer archivos, buscar texto y listar archivos de mi proyecto para responder preguntas sobre el código.

## Criterios de Aceptación

- [x] `tool.Registry` con registro automático de tools
- [x] `Definitions()` retorna las tool definitions para el modelo
- [x] `Execute(ctx, name, input)` despacha la tool correcta
- [x] Tool `bash` — ejecuta comandos shell (30s timeout)
- [x] Tool `read_file` — lee archivos con offset/limit
- [x] Tool `grep` — búsqueda regex en archivos
- [x] Tool `glob` — buscar archivos por patrón

## Archivos Implementados

```
internal/tool/registry.go    # Registry + Definitions + Execute
internal/tool/bash.go        # Shell execution
internal/tool/readfile.go    # File reading
internal/tool/grep.go        # Regex search
internal/tool/glob.go        # Pattern matching
```

## Definición de Done ✅

- El modelo puede invocar las 4 tools
- bash tiene timeout de 30s
- read_file soporta offset/limit para archivos grandes
