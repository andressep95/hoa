# Story 02 — Provider Interface + Anthropic ✅

## Como usuario
Quiero poder enviar un mensaje al modelo y recibir una respuesta de texto, usando Anthropic como primer provider.

## Criterios de Aceptación

- [x] Interface `Provider` con `Send()`, `Model()`, `SetModel()`, `TotalUsage()`
- [x] `AnthropicProvider` implementa la interface usando el SDK oficial
- [x] Soporta tool_use blocks en request/response
- [x] Acumula tokens (input/output/cache) por sesión
- [x] SetModel cambia el modelo en runtime con mutex

## Archivos Implementados

```
internal/provider/provider.go     # Interface Provider
internal/provider/anthropic.go    # Anthropic SDK implementation
```

## Definición de Done ✅

- Enviar un mensaje y recibir respuesta de Claude
- Tool calls funcionan (el modelo puede invocar herramientas)
- TotalUsage acumula tokens correctamente
