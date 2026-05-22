# Story 03 — Agent Loop Básico

## Como usuario
Quiero escribir un mensaje en la terminal y que el agente me responda usando el modelo configurado, en un loop conversacional.

## Criterios de Aceptación

- [ ] `Agent` struct con: Provider, messages slice, system prompt, MaxTurns
- [ ] `agent.Send(ctx, prompt)` agrega mensaje user y ejecuta el loop
- [ ] El loop: envía mensajes → recibe respuesta → si hay tool_use itera, si no retorna
- [ ] El texto del asistente se imprime en stdout conforme llega
- [ ] MaxTurns previene loops infinitos
- [ ] REPL externo: lee input → llama agent.Send → espera siguiente input

## Archivos a Crear

```
internal/agent/agent.go     # Agent struct + New() + Send() + loop()
cmd/hoa/main.go      # Actualizar con REPL básico (stdin scanner)
```

## Loop Simplificado

```go
func (a *Agent) loop(ctx context.Context) (string, error) {
    for turn := 0; turn < a.MaxTurns; turn++ {
        resp, err := a.Provider.Send(ctx, a.messages, nil) // sin tools aún
        if err != nil { return "", err }
        a.messages = append(a.messages, api.Message{Role: api.RoleAssistant, Content: resp.Content})
        // Imprimir texto
        if resp.StopReason != api.StopToolUse {
            return text, nil
        }
    }
    return "", fmt.Errorf("max turns reached")
}
```

## Definición de Done

- Ejecutar el binario → escribir "hola" → recibir respuesta del modelo
- La conversación mantiene contexto (segundo mensaje recuerda el primero)
- Ctrl+D sale limpiamente
