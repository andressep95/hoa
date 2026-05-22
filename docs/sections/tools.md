# Tools — Registro, Permisos y Ejecución

[← INDEX](../INDEX.md)

---

## Concepto

Las tools son las acciones que el agente puede ejecutar en el mundo real. El harness controla **qué** tools están disponibles, **quién** puede usarlas, y **cómo** se ejecutan.

---

## Clasificación de Permisos

| Nivel | Alcance | Ejemplos | Requiere aprobación |
|-------|---------|----------|---------------------|
| **read-only** | Lectura de estado | `ls`, `cat`, `grep`, `git log`, `tree` | No |
| **workspace** | Escritura al proyecto | `write_file`, `edit_file`, `git commit` | Configurable |
| **full-access** | Operaciones destructivas | `rm -rf`, `git push --force`, `docker rm` | Siempre |

---

## Registro de Tools

Cada tool se define con:

```java
public record ToolDefinition(
    String name,
    String description,
    JsonSchema inputSchema,
    PermissionLevel permission,
    boolean requiresConfirmation
) {}
```

---

## Temas a Definir

- [ ] Tool discovery dinámico (como `tool_search` de Claude) vs catálogo estático
- [ ] Rate limiting por tool (evitar loops infinitos de bash)
- [ ] Timeout por ejecución de tool
- [ ] Output truncation (tools que devuelven demasiado texto)
- [ ] Tool pooling — reutilizar instancias costosas (LSP, Docker)
- [ ] Hooks pre/post ejecución (logging, validación, métricas)
- [ ] Tools built-in vs tools via MCP (dónde trazar la línea)

---

## Tools Built-in Candidatas

| Tool | Tipo | Inspiración |
|------|------|-------------|
| `bash` | full-access | Claude Code BashTool |
| `read_file` | read-only | Claude Code FileReadTool |
| `write_file` | workspace | Claude Code FileWriteTool |
| `edit_file` | workspace | Claude Code FileEditTool (str_replace) |
| `grep` | read-only | ripgrep wrapper |
| `glob` | read-only | file discovery |
| `web_search` | read-only | búsqueda web |
| `web_fetch` | read-only | fetch + extract de URLs |
| `ask_user` | read-only | pausa para input humano |
| `sub_agent` | workspace | fork de agente con tarea específica |
