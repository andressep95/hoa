<p align="center">
  <img src="docs/hoa-pet.png" alt="HOA" width="100%"/>
</p>

<h1 align="center">HOA — Harness-Oriented Agents</h1>

<p align="center">
  <strong>El modelo genera. El harness valida, controla y corrige.</strong><br/>
  <em>Un paradigma donde el control es tuyo, no del modelo.</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white" alt="Go 1.23"/>
  <img src="https://img.shields.io/badge/Status-Fase%201-blueviolet" alt="Fase 1"/>
  <img src="https://img.shields.io/badge/Providers-Anthropic%20%7C%20OpenAI-green" alt="Providers"/>
  <img src="https://img.shields.io/badge/License-MIT-yellow" alt="License"/>
</p>

---

## ¿Qué es HOA?

**HOA** es un coding agent de terminal construido sobre un paradigma nuevo: **Harness-Oriented Agents**.

La premisa es simple: el modelo de IA es un commodity intercambiable. Lo que realmente importa es el **harness** — el sistema determinista que valida, controla y corrige al modelo. Si un error se repite, no se arregla el prompt: se arregla el harness.

```
Agente = Modelo + Harness
```

HOA te da control total sobre cómo el agente piensa, actúa y se corrige.

---

## ✨ Features (Fase 1)

### 🔌 Multi-Provider con API Keys Directas

Cambia entre Anthropic y OpenAI sin reiniciar. API keys encriptadas con AES-256-GCM en disco.

```
❯ /provider
  ▸ Anthropic (Claude)
    OpenAI (GPT)
    Ollama (local)
    Google (Gemini)
```

### 🧠 Modelo Dual: Planning + Ejecución

Un modelo potente para planear (opus/o3), uno rápido para ejecutar (sonnet/gpt-4o). El harness decide cuál usar según la tarea.

### 🛠️ Tools Integradas

El agente puede actuar sobre tu filesystem:

| Tool | Descripción |
|------|-------------|
| `bash` | Ejecutar comandos shell (30s timeout) |
| `read_file` | Leer archivos con offset/limit |
| `grep` | Búsqueda regex en archivos |
| `glob` | Buscar archivos por patrón |

### 🔐 Config Encriptada

API keys nunca en plaintext en disco. Master key en `~/.hoa/keyring` con permisos `0600`.

### 🎨 TUI con Bubble Tea

Wizard de configuración con selectores de flechas. Banner estilizado. Prompt coloreado.

---

## 🚀 Quick Start

```bash
git clone https://github.com/cloudcentinel/hoa.git
cd hoa
export ANTHROPIC_API_KEY=sk-ant-...   # o OPENAI_API_KEY=sk-...
go run ./cmd/hoa
```

Primera ejecución → wizard interactivo te guía. Después arranca directo.

```
  ██╗  ██╗ ██████╗  █████╗ 
  ██║  ██║██╔═══██╗██╔══██╗
  ███████║██║   ██║███████║
  ██╔══██║██║   ██║██╔══██║
  ██║  ██║╚██████╔╝██║  ██║
  ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝
  Harness-Oriented Agents

  provider: anthropic
  base: claude-sonnet-4-20250514
  planning: claude-opus-4-20250414

  /help para comandos · /exit para salir

❯ lista los archivos .go del proyecto
[tool] glob
...
```

---

## 📐 Arquitectura

```
┌─────────────────────────────────────────────────────┐
│  cmd/hoa/main.go       REPL + banner + commands     │
└─────────────────────────────────────────────────────┘
        │
        ├── internal/api/          Tipos genéricos (Message, Block, ToolDef)
        │
        ├── internal/provider/     Interface Provider + Anthropic + OpenAI
        │
        ├── internal/agent/        Agent loop (Send → model → tools → repeat)
        │
        ├── internal/tool/         Registry auto-registrante + bash/read/grep/glob
        │
        ├── internal/config/       Load/Save + AES-256-GCM + Wizard
        │
        └── internal/ui/           Selector + TextInput (Bubble Tea)
```

### El Agent Loop

```
[tu input]
    │
    ▼
[agregar a messages]
    │
    ▼
[llamar al modelo con tools] ──┐
    │                          │
    ▼                          │
[¿hay tool_use?] ──no─────────┴──▶ [imprimir respuesta]
    │
   sí
    │
    ▼
[ejecutar herramientas]
    │
    ▼
[agregar tool_results]
    │
    ▼
(volver a llamar al modelo)
```

---

## 🗺️ Roadmap

### Fase 1 — Core Agent ✅ (actual)
- [x] Config con wizard TUI + encriptación
- [x] Provider Interface (Anthropic + OpenAI)
- [x] Agent Loop con tool execution
- [x] Tools: bash, read_file, grep, glob
- [x] Banner + REPL estilizado

### Fase 1B — Harness Layer (próximo)
- [ ] Write file + diff approval
- [ ] Slash commands: /provider, /model con selectores
- [ ] Compaction strategies (SlidingWindow, Summarize)
- [ ] Dual-model router (planning vs execution)
- [ ] SDD Engine (Spec-Driven Development)
- [ ] Write-Verify Loop (L0-L5)
- [ ] Commit tool con amnesia

### Fase 1C — Memory & Intelligence
- [ ] Memory persistente entre sesiones
- [ ] Oracle 23ai vector store
- [ ] Subagent research (delegación read-only)
- [ ] Eliminación categórica de errores

### Fase 2 — Skills
- [ ] Skill template (YAML canónico)
- [ ] Skill discovery pre-LLM (LIKE + vector)
- [ ] Context injection por task (breadcrumbs del planner)
- [ ] Skill creator (/skill create)

---

## 🧬 El Paradigma HOA

| Principio | Descripción |
|-----------|-------------|
| **El harness manda** | Si el agente falla repetidamente, se arregla el harness, no se reza |
| **Verificación determinista** | El harness valida, no el modelo. Write → Verify → Accept/Rollback |
| **Amnesia controlada** | Post-commit se limpia el contexto. Solo lo relevante sobrevive |
| **Multi-proveedor** | Cambiar de Claude a GPT-4o a Ollama es configuración, no refactor |
| **Eliminación categórica** | Errores repetidos se convierten en reglas preventivas |
| **Progressive Disclosure** | Solo inyectar al modelo lo que la tarea necesita |
| **Skills como marco** | El agente sabe CÓMO hacer las cosas, no solo QUÉ hacer |

---

## 🛠️ Requisitos

- Go 1.23+
- Una API key: [Anthropic](https://console.anthropic.com) o [OpenAI](https://platform.openai.com)

---

## 📄 Documentación

| Doc | Descripción |
|-----|-------------|
| [INDEX.md](INDEX.md) | Mapa del proyecto |
| [Fase 1](docs/phases/fase-1/README.md) | Arquitectura y decisiones |
| [Stories](docs/phases/fase-1/stories/) | Historias de usuario paso a paso |
| [Harnesses](docs/sections/harnesses.md) | Catálogo de harnesses |
| [Skills](docs/sections/skills.md) | Sistema de skills + template |

---

## 📝 Licencia

MIT

---

<p align="center">
  <em>Built with 🎛️ by <a href="https://github.com/cloudcentinel">CloudCentinel</a></em>
</p>
