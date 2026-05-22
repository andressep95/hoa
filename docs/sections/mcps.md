# MCPs — Model Context Protocol Integration

[← INDEX](../INDEX.md)

---

## Concepto

MCP permite exponer y consumir tools/resources de servidores externos. HOA actúa como **cliente MCP** (consume tools de otros servers) y opcionalmente como **servidor MCP** (expone sus capacidades a otros clientes).

---

## Roles

| Rol | Descripción | Ejemplo |
|-----|-------------|---------|
| **MCP Client** | Consume tools de servidores externos | Conectar a tu memory-management-mcp |
| **MCP Server** | Expone tools propias a otros clientes | Que VS Code consuma tus tools |

---

## Transportes Soportados

| Transporte | Uso | Notas |
|------------|-----|-------|
| **stdio** | Servidores locales (procesos hijo) | Más simple, sin red |
| **SSE (HTTP)** | Servidores remotos | Tu MCP actual usa esto |
| **Streamable HTTP** | Nuevo estándar MCP | Stateful con resumability |

---

## Temas a Definir

- [ ] Auto-discovery de MCP servers (`.mcp.json` en el proyecto)
- [ ] Aprobación de MCP servers nuevos (seguridad supply-chain)
- [ ] Timeout y retry para conexiones MCP
- [ ] Cómo exponer tools MCP al modelo (inyección en system prompt vs tool_search)
- [ ] Límite de tools MCP por sesión (evitar context bloat)
- [ ] MCP server lifecycle management (start/stop/health-check)
- [ ] Mapping de permisos: tool MCP → nivel de permiso interno

---

## Integración con memory-management-mcp

Tu MCP existente se convierte en un servidor MCP que HOA consume:

```
HOA (client) ──SSE──→ memory-management-mcp (server)
                              │
                              ├── queryMemory
                              ├── queryCode
                              ├── querySkills
                              └── setupProject
```

Alternativa: integrar la lógica de memoria directamente como módulo interno (sin red). Decisión pendiente.

---

## Propuesta: CodeGraph como MCP externo

**Repo:** https://github.com/colbymchenry/codegraph  
**Qué es:** Grafo de conocimiento de código (tree-sitter + SQLite). Extrae símbolos, relaciones (calls, imports, extends), y expone búsqueda + impact analysis via MCP.  
**Resultado:** ~35% menos costo, ~70% menos tool calls — el agente consulta el grafo en vez de explorar archivos.

### Por qué nos sirve

| Memoria vectorial (nuestro MCP) | CodeGraph |
|---------------------------------|-----------|
| Decisiones históricas (por qué) | Estructura actual (qué/cómo) |
| Busca por semántica | Busca por símbolo/relación |
| Contexto temporal (commits) | Contexto estructural (call graph) |

Son **complementarios**: memoria dice "decidimos usar Strategy pattern", CodeGraph dice "ProviderRouter tiene 3 callees concretos".

### Tools que expone

| Tool | Uso en HOA |
|------|-------------------|
| `codegraph_context` | Pre-inyectar estructura relevante antes de cada turno |
| `codegraph_search` | Buscar símbolos por nombre (rápido, sin grep) |
| `codegraph_callers` / `codegraph_callees` | Trazar flujo antes de editar |
| `codegraph_impact` | Validar qué se rompe antes de aplicar cambios |

### Integración propuesta

```
HOA ──stdio──→ codegraph serve --mcp
                       │
                       └── .codegraph/codegraph.db (SQLite local)
```

- Levantar como proceso hijo (stdio transport)
- El harness lo consulta automáticamente en paso de "context building"
- No requiere aprobación del usuario (read-only, local)

### Consideraciones

- Dependencia en Node.js (el server es JS)
- 19+ lenguajes soportados (incluye Java)
- Auto-sync con file watcher — el grafo se actualiza solo
- A futuro: evaluar reimplementar las partes críticas en Java si queremos eliminar la dependencia de Node
