# Story 14 — Memory Persistente Entre Sesiones

## Como usuario
Quiero que el agente recuerde decisiones y contexto de sesiones anteriores sin que yo tenga que repetirlo.

## Criterios de Aceptación

- [ ] Al shutdown: auto-summarize la sesión (pide al modelo un párrafo + tags)
- [ ] Persistir en `.hoa/sessions/` como archivos markdown
- [ ] Al startup: cargar preamble con resúmenes de sesiones recientes al system prompt
- [ ] Tool `remember(content, kind, tags)` — guardar dato explícitamente
- [ ] Tool `recall(query)` — buscar en sesiones anteriores (keyword match)
- [ ] Límite de preamble: máx 5 sesiones recientes o N tokens

## Archivos a Crear

```
internal/memory/store.go           # Interface MemoryStore
internal/memory/sessionfiles.go    # File-based implementation
internal/tool/remember.go          # remember tool
internal/tool/recall.go            # recall tool
```

## Definición de Done

- Sesión 1: "el proyecto usa Oracle 23ai" → shutdown → summary guardado
- Sesión 2: el agente ya sabe que usamos Oracle (está en el preamble)
- `recall("qué base de datos usamos")` → devuelve la info
