# Story 03 — Agent Loop Básico ✅

## Como usuario
Quiero escribir un mensaje en la terminal y que el agente me responda usando el modelo configurado, en un loop conversacional.

## Criterios de Aceptación

- [x] `Agent.Send(ctx, prompt)` agrega mensaje y ejecuta loop
- [x] Loop: enviar → recibir → si hay tool_use ejecutar → repetir
- [x] MaxTurns limita iteraciones (default 20)
- [x] `OnOutput` callback para emitir texto/tool events sin acoplar a stdout
- [x] `SendOneShot` para llamadas aisladas sin afectar historial
- [x] `ClearMessages()` limpia conversación

## Archivos Implementados

```
internal/agent/agent.go    # Agent struct + Send + loop + SendOneShot
internal/api/types.go      # Message, Block, Usage, Response, ToolDef
```

## Definición de Done ✅

- Conversación multi-turno funciona
- Tools se ejecutan y resultados vuelven al modelo
- SendOneShot no contamina el historial (usado por /commit)
