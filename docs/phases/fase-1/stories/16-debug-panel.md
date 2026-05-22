# Story 16 — Debug Panel

## Como usuario
Quiero poder ver en tiempo real qué está haciendo el harness: llamadas al provider, tool dispatch, compactación, costos.

## Criterios de Aceptación

- [ ] `/debug on` activa panel lateral en la TUI
- [ ] Muestra: llamadas al provider (modelo, tokens, latencia)
- [ ] Muestra: dispatch de tools (nombre, input truncado, duración, resultado)
- [ ] Muestra: eventos de compactación (antes/después)
- [ ] `/debug off` oculta el panel
- [ ] `/debug clear` limpia el historial de eventos
- [ ] `HARNESS_DEBUG=1` arranca con debug activo

## Archivos a Crear

```
internal/debug/debug.go    # Event recording + SetEnabled + SetSink
```

## Definición de Done

- `/debug on` → panel aparece → hacer una pregunta → ver request/response al provider
- Ver timing de cada tool call
- `/debug off` → panel desaparece, no afecta performance
