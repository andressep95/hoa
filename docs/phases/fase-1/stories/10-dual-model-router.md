# Story 10 — Dual-Model Router

## Como usuario
Quiero que el agente use un modelo potente (opus/o3) para planear y uno rápido (sonnet/gpt-4o) para ejecutar, seleccionando automáticamente según la tarea.

## Criterios de Aceptación

- [ ] Config tiene `models.base` y `models.planning`
- [ ] Router clasifica intent: ¿es planning o ejecución?
- [ ] Planning: SDD phases (proposal, spec, design) → usa planning model
- [ ] Ejecución: tool calls, writes, verification → usa base model
- [ ] `/model` muestra ambos modelos activos
- [ ] Se puede forzar modelo con flag en el prompt (ej: `@planning explica...`)

## Archivos a Crear

```
internal/provider/router.go     # DualModelRouter que wrappea dos providers
internal/agent/router.go        # Intent classification logic
```

## Definición de Done

- Pedir "planea una feature de auth" → usa planning model (visible en /debug)
- Pedir "escribe el archivo config.go" → usa base model
- `/tokens` muestra uso separado por modelo
