<p align="center">
  <img src="docs/hoa-pet.png" alt="HOA" width="100%"/>
</p>

<h1 align="center">HOA — Harness-Oriented Agents</h1>

<p align="center">
  <strong>El modelo genera. El harness valida, controla y corrige.</strong><br/>
  <em>Un paradigma donde el control es tuyo, no del modelo.</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go 1.26"/>
  <img src="https://img.shields.io/badge/Status-Fase%201-blueviolet" alt="Fase 1"/>
  <img src="https://img.shields.io/badge/Providers-Anthropic%20%7C%20OpenAI-green" alt="Providers"/>
  <img src="https://img.shields.io/badge/License-MIT-yellow" alt="License"/>
</p>

---

## ¿Qué es HOA?

**HOA** es un coding agent de terminal construido sobre un paradigma nuevo: **Harness-Oriented Agents**.

La premisa es simple: el modelo de IA es un commodity intercambiable. Lo que realmente importa es el **harness** — el sistema determinista que valida, controla y corrige al modelo.

```
Agente = Modelo + Harness
```

---

## ✨ Features

### 🎨 TUI Completa (Bubble Tea)

Interfaz de terminal con alt-screen, input con historial (↑/↓), viewport scrollable (PageUp/PageDown), spinner mientras piensa, y autocomplete de comandos con dropdown filtrable.

### 🔌 Multi-Provider con Hot-Swap

Cambia entre providers en runtime sin reiniciar. API keys encriptadas con AES-256-GCM.

```
❯ /provider
  anthropic    ✔ activo
  openai       configurado
  ───────────────────
  ＋ Agregar ollama
  ＋ Agregar google
  ───────────────────
  🔑 Cambiar API key de anthropic
```

### 🧠 Modelos y Modos

Selección interactiva de modelo con dos modos de operación:

| Modo | Comportamiento |
|------|---------------|
| `execute` | Modelo base responde directo |
| `plan+execute` | Planning model planea, base ejecuta |

```
❯ /model
  🧠 anthropic · execute: claude-sonnet-4-6 · plan: claude-opus-4-7

  ── execute ──
    claude-sonnet-4-6  ✔
    claude-opus-4-7
    claude-haiku-4-5

  ── planning ──
    claude-sonnet-4-6
    claude-opus-4-7  ✔
    claude-haiku-4-5
```

### 📝 /commit — Commits Inteligentes con LLM

El agente analiza tu diff, genera mensajes Conventional Commits en JSON estructurado, propone splits si detecta cambios no relacionados, y ejecuta con confirmación:

```
❯ /commit
  ⎿  Analizando cambios...

  [1/2] feat(agent): add SendOneShot for isolated LLM calls
    what: Adds SendOneShot to use LLM without conversation history
    why:  Enables one-time operations without altering session state
    breaking: false
    ⎿  internal/agent/agent.go

  [2/2] refactor(ui): pass banner as lazy func
    what: Banner re-evaluates on each render for live state
    why:  Static banner was stale after model or mode changes
    breaking: false
    ⎿  internal/ui/program.go

  ✓ Commitear 2 commits separados
  ⊕ Unificar en 1 solo commit
  ✎ Dar feedback (regenerar)
  ✗ Cancelar
```

Incluye validación pre-commit (Conventional Commits), detección de archivos sensibles, y feedback con hash:

```
  ⎿  a1b2c3d feat(agent): add SendOneShot for isolated LLM calls
  ⎿  e4f5g6h refactor(ui): pass banner as lazy func
```

### 🛠️ Tools Integradas

| Tool | Descripción |
|------|-------------|
| `bash` | Ejecutar comandos shell (30s timeout) |
| `read_file` | Leer archivos con offset/limit |
| `grep` | Búsqueda regex en archivos |
| `glob` | Buscar archivos por patrón |

### 💰 Cost Tracking

```
❯ /tokens
  tokens: 1250 in · 340 out · 1590 total
  costo:  $0.0089 (estimado)
  modelo: claude-sonnet-4-6
```

### ⚡ Slash Commands

| Comando | Descripción |
|---------|-------------|
| `/mode` | Alterna execute / plan+execute |
| `/model` | Selecciona modelo (menú interactivo) |
| `/provider` | Cambia provider / agrega nuevo / modifica API key |
| `/tokens` | Muestra tokens y costo estimado |
| `/commit` | Commit inteligente con LLM |
| `/memory` | Gestiona memoria persistente (placeholder) |
| `/tools` | Lista herramientas disponibles |
| `/clear` | Limpia historial |
| `/exit` | Salir |

---

## 🚀 Quick Start

```bash
git clone https://github.com/cloudcentinel/hoa.git
cd hoa
go build -o hoa ./cmd/hoa
./hoa
```

Primera ejecución → wizard interactivo configura provider, modelo y memoria opcional.

Variables de entorno soportadas (override config):
- `ANTHROPIC_API_KEY` / `ANTHROPIC_MODEL`
- `OPENAI_API_KEY` / `OPENAI_MODEL`

---

## 📐 Arquitectura

```
cmd/hoa/main.go              Entry point + wiring

internal/
├── agent/agent.go           Agent loop (Send → model → tools → repeat)
├── api/types.go             Tipos genéricos (Message, Block, Usage)
├── command/                  Slash commands (1 archivo por comando)
│   ├── registry.go          Dispatch + Context
│   ├── commit.go            /commit con LLM + validación
│   ├── model.go             /model selector
│   ├── provider.go          /provider con setup
│   ├── mode.go              /mode toggle
│   ├── tokens.go            /tokens + cost
│   ├── validate.go          Pre-commit validation
│   └── ...
├── config/                   Load/Save + AES-256-GCM + Wizard
├── cost/tracker.go          Per-model cost estimation
├── provider/                 Interface + Anthropic + OpenAI
├── tool/                     Registry + bash/read/grep/glob
└── ui/
    ├── program.go           Bubble Tea main model
    ├── styles.go            Lipgloss styles centralizados
    ├── textinput.go         Input component
    └── selector.go          Selector component
```

---

## 🗺️ Roadmap — Fase 1

| # | Story | Estado |
|---|-------|--------|
| 01 | Config y Primer Uso | ✅ |
| 02 | Provider Interface + Anthropic | ✅ |
| 03 | Agent Loop Básico | ✅ |
| 04 | Tool Registry + Tools Read-Only | ✅ |
| 05 | TUI con Bubble Tea | ✅ |
| 06 | Slash Commands | ✅ |
| 07 | Write File + Diff Approval | ⬜ |
| 08 | Provider OpenAI + Swap en Runtime | ✅ |
| 09 | Compaction Strategies | ⬜ |
| 10 | Dual-Model Router | ⬜ (switch listo, lógica pendiente) |
| 11 | SDD Engine | ⬜ |
| 12 | Write-Verify Loop | ⬜ |
| 13 | Commit con Amnesia | ⬜ |
| 14 | Memory Persistente | ⬜ (config listo, Oracle pendiente) |
| 15 | Subagent Research | ⬜ |
| 16 | Debug Panel | ⬜ |

---

## 🧬 El Paradigma HOA

| Principio | Descripción |
|-----------|-------------|
| **El harness manda** | Si el agente falla, se arregla el harness, no el prompt |
| **Verificación determinista** | Write → Verify → Accept/Rollback |
| **Amnesia controlada** | Post-commit se limpia el contexto |
| **Multi-proveedor** | Cambiar de Claude a GPT es configuración |
| **Skills como marco** | El agente sabe CÓMO hacer las cosas |

---

## 📄 Licencia

MIT

---

<p align="center">
  <em>Built with 🎛️ by <a href="https://github.com/cloudcentinel">CloudCentinel</a></em>
</p>
