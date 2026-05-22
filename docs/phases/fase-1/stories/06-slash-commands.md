# Story 06 — Slash Commands

## Como usuario
Quiero poder ejecutar comandos con `/` para controlar el agente: cambiar modelo, ver tokens, limpiar historial.

## Criterios de Aceptación

- [ ] Input que empieza con `/` se despacha como comando, no va al modelo
- [ ] `/help` — lista todos los comandos disponibles
- [ ] `/model [nombre]` — muestra o cambia el modelo activo
- [ ] `/provider [nombre]` — muestra o cambia el provider
- [ ] `/tokens` — muestra tokens acumulados (input/output) y costo estimado
- [ ] `/clear` — limpia historial de conversación
- [ ] `/tools` — lista tools registradas
- [ ] `/exit` — sale del programa
- [ ] Comando desconocido → mensaje de error amigable

## Archivos a Crear

```
commands.go    # Registro de slash commands + dispatch
```

## Definición de Done

- `/help` muestra la lista completa
- `/model claude-sonnet-4-20250514` cambia el modelo sin reiniciar
- `/tokens` muestra uso acumulado
- `/foo` → "comando desconocido, usa /help"
