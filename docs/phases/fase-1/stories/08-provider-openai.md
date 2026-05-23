# Story 08 — Provider OpenAI + Swap en Runtime ✅

## Como usuario
Quiero poder cambiar entre Anthropic y OpenAI a mitad de sesión sin reiniciar, manteniendo el historial de conversación.

## Criterios de Aceptación

- [x] `OpenAIProvider` implementa la interface Provider
- [x] Soporta Chat Completions API con tool_calls
- [x] `/provider` cambia el provider en runtime (recrea el client)
- [x] El historial de conversación se mantiene al cambiar
- [x] SetModel funciona con mutex para thread safety
- [x] TotalUsage acumula tokens por sesión
- [x] Se puede agregar un nuevo provider con API key desde `/provider`

## Archivos Implementados

```
internal/provider/openai.go    # OpenAI SDK implementation
```

## Definición de Done ✅

- Cambiar de Anthropic a OpenAI via `/provider` funciona sin reiniciar
- La conversación continúa con el nuevo provider
- API key se puede configurar inline desde el menú
