# 🎛️ HOA — Harness-Oriented Agents

> El paradigma donde el harness manda. El modelo es intercambiable; el control es tuyo.

**Stack:** Go 1.23 + Bubble Tea + API Keys directas + Oracle 23ai  
**Ecuación:** `Agent = Modelo + Harness` — El harness impone calidad determinista sobre inteligencia probabilística.

---

## El Paradigma HOA

```
┌─────────────────────────────────────────────────┐
│  Harness-Oriented Agents                        │
│                                                 │
│  El modelo genera. El harness valida, controla  │
│  y corrige. Si un error se repite, no se        │
│  arregla el prompt — se arregla el harness.     │
│                                                 │
│  El harness ES el producto. El modelo es un     │
│  commodity intercambiable.                      │
└─────────────────────────────────────────────────┘
```

---

## Áreas del Sistema

| # | Área | Descripción | Estado | Doc |
|---|------|-------------|--------|-----|
| 1 | **Tools** | Registro, permisos, ejecución y control granular de herramientas | 🔴 Por definir | [→ tools.md](./docs/sections/tools.md) |
| 2 | **MCPs** | Integración con servidores MCP externos e internos | 🔴 Por definir | [→ mcps.md](./docs/sections/mcps.md) |
| 3 | **Memoria** | Vectorial + amnesia por commit + Progressive Context Disclosure | 🟡 Base existente | [→ memoria.md](./docs/sections/memoria.md) |
| 4 | **Cache** | Cache efímero entre commits, invalidación, token budget | 🔴 Por definir | [→ cache.md](./docs/sections/cache.md) |
| 5 | **Code Writing** | Estrategias de escritura, diff, validación y rollback | 🔴 Por definir | [→ code-writing.md](./docs/sections/code-writing.md) |
| 6 | **Harnesses** | Catálogo completo de harnesses: SDD, verify, skills, seguridad | 🟢 Definido | [→ harnesses.md](./docs/sections/harnesses.md) |
| 7 | **Skills** | Marcos de trabajo reutilizables: procedimientos, reglas, patrones por dominio | 🟢 Definido | [→ skills.md](./docs/sections/skills.md) |
| 8 | **Sandbox** | ~~Aislamiento de ejecución~~ — Absorbido por Permission Gate en Tools | ❌ Descartado | [→ sandbox.md](./docs/sections/sandbox.md) |

---

## Roadmap

| Fase | Alcance | Doc | Stories |
|------|---------|-----|---------|
| **Fase 1** | MVP: config, providers, agent loop, tools, TUI, SDD, verify loop | [→ README](./docs/phases/fase-1/README.md) | [→ stories/](./docs/phases/fase-1/stories/) |

### Stories de Fase 1

| # | Story | Área |
|---|-------|------|
| 01 | [Config y Primer Uso](./docs/phases/fase-1/stories/01-config-primer-uso.md) | Core |
| 02 | [Provider Interface + Anthropic](./docs/phases/fase-1/stories/02-provider-interface.md) | Core |
| 03 | [Agent Loop Básico](./docs/phases/fase-1/stories/03-agent-loop.md) | Core |
| 04 | [Tool Registry + Tools Read-Only](./docs/phases/fase-1/stories/04-tool-registry.md) | Core |
| 05 | [TUI con Bubble Tea](./docs/phases/fase-1/stories/05-tui-bubble-tea.md) | Core |
| 06 | [Slash Commands](./docs/phases/fase-1/stories/06-slash-commands.md) | Core |
| 07 | [Write File + Diff Approval](./docs/phases/fase-1/stories/07-write-file-approval.md) | Core |
| 08 | [Provider OpenAI + Swap](./docs/phases/fase-1/stories/08-provider-openai.md) | Core |
| 09 | [Compaction Strategies](./docs/phases/fase-1/stories/09-compaction.md) | Core |
| 10 | [Dual-Model Router](./docs/phases/fase-1/stories/10-dual-model-router.md) | Harness |
| 11 | [SDD Engine](./docs/phases/fase-1/stories/11-sdd-engine.md) | Harness |
| 12 | [Write-Verify Loop](./docs/phases/fase-1/stories/12-write-verify-loop.md) | Harness |
| 13 | [Commit Tool con Amnesia](./docs/phases/fase-1/stories/13-commit-amnesia.md) | Harness |
| 14 | [Memory Persistente](./docs/phases/fase-1/stories/14-memory-persistente.md) | Memory |
| 15 | [Subagent Research](./docs/phases/fase-1/stories/15-subagent-research.md) | Memory |
| 16 | [Debug Panel](./docs/phases/fase-1/stories/16-debug-panel.md) | Polish |

---

## Decisiones Arquitectónicas Clave

| Decisión | Elección | Razón |
|----------|----------|-------|
| Lenguaje | Go 1.23 (goroutines) | Binario estático ~25 MB, startup <100ms, cross-compile trivial |
| Base de código | Fork/adapt de byo-coding-agent | Harness funcional probado (~1000 líneas) |
| Multi-proveedor | Interface `Provider` + SDK por vendor | Anthropic, OpenAI, Ollama, Google sin cambiar código |
| Acceso a modelos | API keys directas | Sin intermediarios cloud, latencia mínima |
| Embeddings | Oracle 23ai `DBMS_VECTOR_CHAIN` | La BD genera embeddings — cero modelo local |
| Vector Store | Oracle 23ai VECTOR columns | Configurable: Docker local o remoto |
| TUI | Bubble Tea + Lipgloss | TUI interactiva con diff approval, debug panel |
| CLI | Slash commands en REPL | `/help`, `/model`, `/provider` — sin subcommands |

---

## Principios HOA

1. **El harness manda** — Si el agente falla repetidamente, se arregla el harness, no se reza.
2. **Amnesia controlada** — El contexto se limpia post-commit. Solo lo relevante sobrevive via búsqueda vectorial.
3. **Progressive Context Disclosure** — No inyectar todo. Solo lo que la tarea necesita.
4. **Permisos explícitos** — Cada tool tiene un nivel de acceso (read-only / workspace / full).
5. **Multi-proveedor sin lock-in** — Cambiar de Claude a GPT-4o a Ollama es configuración, no refactor.
6. **Verificación determinista** — El harness valida, no el modelo. Write → Verify → Accept/Rollback.
7. **Eliminación categórica** — Errores repetidos se convierten en reglas preventivas del harness.
