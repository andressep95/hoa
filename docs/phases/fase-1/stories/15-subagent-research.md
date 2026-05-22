# Story 15 — Subagent Research (Delegación Read-Only)

## Como usuario
Quiero que el agente pueda delegar investigaciones a un subagente con su propio contexto, sin contaminar mi conversación principal.

## Criterios de Aceptación

- [ ] `Research` subagent: Agent con tool subset limitado (solo `read_file`)
- [ ] Tool `delegate_research(query)` — el root agent delega la pregunta
- [ ] El subagent lee archivos, investiga, y devuelve un resumen
- [ ] El resumen se inyecta como tool_result en el root agent
- [ ] El subagent tiene su propio context window (no gasta el del root)
- [ ] `/subagents` muestra subagents registrados

## Archivos a Crear

```
internal/subagent/registry.go    # Subagent registry
internal/subagent/research.go    # Research subagent
delegate.go                      # DelegateTool wrapper
```

## Definición de Done

- "investiga cómo funciona el provider router" → delega → resumen aparece
- El context window del root no crece con las lecturas del subagent
- `/subagents` muestra "research: idle"
