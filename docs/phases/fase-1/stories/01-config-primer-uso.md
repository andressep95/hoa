# Story 01 — Config y Primer Uso ✅

## Como usuario
Quiero que al ejecutar `hoa` por primera vez me guíe un wizard interactivo para configurar mi provider y API key, y que en ejecuciones posteriores arranque directo sin preguntar.

## Criterios de Aceptación

- [x] `config.Load()` lee `~/.hoa/config.json` si existe
- [x] Si no existe, `config.RunWizard()` pide: provider, API key, modelo base, modelo planning
- [x] `config.Save()` persiste la configuración en JSON
- [x] La estructura del config soporta múltiples providers con sus API keys
- [x] Variables de entorno (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`) se usan como fallback si no hay config
- [x] API keys encriptadas con AES-256-GCM en disco
- [x] Paso opcional para configurar memoria persistente (Oracle)
- [x] Model resolution chain: env var → config → default

## Archivos Implementados

```
internal/config/config.go    # Config struct + Load/Save + ResolveModel + MemoryConfig
internal/config/crypto.go    # AES-256-GCM encrypt/decrypt + keyring
internal/config/wizard.go    # TUI wizard con selectores Bubble Tea
```

## Definición de Done ✅

- `go build ./...` compila
- Ejecutar sin config muestra wizard con selectores interactivos
- Ejecutar con config existente arranca directo
- Cambios en runtime (/model, /provider, /mode) se persisten automáticamente
