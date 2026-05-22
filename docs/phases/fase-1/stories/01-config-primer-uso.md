# Story 01 — Config y Primer Uso

## Como usuario
Quiero que al ejecutar `hoa` por primera vez me guíe un wizard interactivo para configurar mi provider y API key, y que en ejecuciones posteriores arranque directo sin preguntar.

## Criterios de Aceptación

- [ ] `config.Load()` lee `~/.hoa/config.json` si existe
- [ ] Si no existe, `config.RunWizard()` pide: provider, API key, modelo base, modelo planning
- [ ] `config.Save()` persiste la configuración en JSON
- [ ] La estructura del config soporta múltiples providers con sus API keys
- [ ] Variables de entorno (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`) se usan como fallback si no hay config

## Archivos a Crear

```
internal/config/config.go    # Struct Config + Load/Save/RunWizard
cmd/hoa/main.go       # Entry point que llama config.Load()
```

## Config Struct

```go
type Config struct {
    ActiveProvider string              `json:"provider"`
    Models         ModelsConfig        `json:"models"`
    Providers      map[string]ProviderConfig `json:"providers"`
    Database       DatabaseConfig      `json:"database"`
    Harness        HarnessConfig       `json:"harness"`
}

type ModelsConfig struct {
    Base     string `json:"base"`
    Planning string `json:"planning"`
}

type ProviderConfig struct {
    APIKey  string `json:"apiKey,omitempty"`
    BaseURL string `json:"baseUrl,omitempty"`
}
```

## Definición de Done

- `go build ./...` compila
- `go test ./internal/config/...` pasa
- Ejecutar el binario sin config muestra el wizard
- Ejecutar con config existente arranca sin preguntar
