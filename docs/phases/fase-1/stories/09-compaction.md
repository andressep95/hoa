# Story 09 — Compaction Strategies

## Como usuario
Quiero que conversaciones largas no exploten el context window — el harness debe compactar automáticamente.

## Criterios de Aceptación

- [ ] Interface `CompactionStrategy` con `Compact(ctx, messages) (messages, error)`
- [ ] `NoCompaction` — default, no toca nada
- [ ] `SlidingWindow{KeepLast: N}` — descarta mensajes antiguos
- [ ] `Summarize{Provider, Threshold, KeepRecent}` — resume turnos antiguos via LLM
- [ ] `SafeSplitPoint()` — nunca corta entre tool_use y tool_result
- [ ] `/compact [sliding|summarize|none]` — ejecuta compactación ad-hoc
- [ ] Se ejecuta automáticamente al inicio de cada turn del agent loop

## Archivos a Crear

```
internal/compact/strategy.go        # Interface + SafeSplitPoint
internal/compact/nocompaction.go    # NoCompaction
internal/compact/slidingwindow.go   # SlidingWindow
internal/compact/summarize.go       # Summarize
```

## Definición de Done

- Conversación de 50+ turnos no falla por context overflow
- `/compact summarize` reduce el historial y el agente sigue coherente
- SafeSplitPoint nunca deja un tool_use huérfano
