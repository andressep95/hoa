# Story 08 — Provider OpenAI + Swap en Runtime

## Como usuario
Quiero poder cambiar entre Anthropic y OpenAI a mitad de sesión sin reiniciar, manteniendo el historial de conversación.

## Criterios de Aceptación

- [ ] `OpenAIProvider` implementa la interface `Provider`
- [ ] Traduce Message/Block genéricos al formato OpenAI (tool results como mensajes separados)
- [ ] `/provider openai [modelo]` cambia el provider activo
- [ ] `/provider anthropic` vuelve a Anthropic
- [ ] El historial de conversación se mantiene al cambiar
- [ ] Cada provider acumula su propio usage
- [ ] `$LLM_PROVIDER` y `$LLM_MODEL` como defaults al arranque

## Archivos a Crear

```
internal/provider/openai.go    # OpenAIProvider
```

## Definición de Done

- Arrancar con Anthropic → conversar → `/provider openai` → seguir conversando
- El modelo OpenAI entiende el historial previo
- `/tokens` muestra totales correctos
