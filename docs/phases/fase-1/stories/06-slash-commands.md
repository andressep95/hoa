# Story 06 — Slash Commands ✅

## Como usuario
Quiero poder ejecutar comandos con `/` para controlar el agente: cambiar modelo, ver tokens, limpiar historial.

## Criterios de Aceptación

- [x] Input que empieza con `/` se despacha como comando, no va al modelo
- [x] `/help` — lista todos los comandos disponibles
- [x] `/model` — menú interactivo con secciones execute/planning y ✔ en activo
- [x] `/provider` — menú con providers configurados, agregar nuevo, cambiar API key
- [x] `/mode` — alterna entre execute y plan+execute
- [x] `/tokens` — muestra tokens acumulados + costo estimado USD
- [x] `/commit` — genera commits con LLM (JSON estructurado, split, confirmación, hash feedback)
- [x] `/memory` — placeholder para gestión de memoria persistente
- [x] `/clear` — limpia historial de conversación
- [x] `/tools` — lista tools registradas
- [x] `/exit` — sale del programa
- [x] Comando desconocido → mensaje de error amigable
- [x] Comandos async con spinner (AsyncFn)
- [x] Validación pre-commit (Conventional Commits format)
- [x] Persistencia: cambios en model/provider/mode se guardan a config.json

## Archivos Implementados

```
internal/command/
├── registry.go     # Dispatch + Context + MenuItem + Result (con AsyncFn)
├── help.go         # /help
├── model.go        # /model (execute/planning sections)
├── provider.go     # /provider (switch, add, modify key)
├── mode.go         # /mode (execute / plan+execute)
├── tokens.go       # /tokens + cost estimation
├── commit.go       # /commit (LLM JSON, split, validation, hash feedback)
├── memory.go       # /memory (placeholder)
├── clear.go        # /clear
├── tools.go        # /tools
├── exit.go         # /exit
└── validate.go     # Conventional Commits validator
```

## Definición de Done ✅

- `/help` muestra lista completa
- `/model` muestra menú con modelos del provider activo, ✔ en el seleccionado
- `/provider` permite cambiar, agregar nuevo con API key, o modificar key existente
- `/mode` alterna y persiste
- `/commit` analiza diff con LLM, propone 1-N commits, confirma con yes/no, muestra hashes
- `/tokens` muestra uso + costo estimado
- Comando desconocido → error amigable
- Todos los cambios persisten entre sesiones
