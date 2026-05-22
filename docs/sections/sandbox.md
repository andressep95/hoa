# Sandbox — Aislamiento de Ejecución

[← INDEX](../INDEX.md)

> ❌ **Estado: Descartado como área independiente** — La restricción se maneja via el Permission Gate de Tools.
> El aislamiento real se logra con: path validation en write tools + working dir lock en bash + confirmación humana para operaciones destructivas.

---

## Concepto

El sandbox restringe al agente a operar **solo dentro del workspace del proyecto**. Previene escrituras accidentales fuera del directorio de trabajo, ejecución de comandos peligrosos, y acceso a archivos sensibles.

---

## Niveles de Aislamiento

| Nivel | Restricción | Complejidad | Seguridad |
|-------|-------------|-------------|-----------|
| **Path validation** | Solo verificar que paths estén dentro del workspace | Baja | Media |
| **chroot-like** | Redirigir todas las operaciones FS al workspace | Media | Alta |
| **Container** | Ejecutar el agente dentro de un Docker container | Alta | Muy alta |
| **VM** | Máquina virtual completa | Muy alta | Máxima |

---

## Path Validation (mínimo viable)

```java
public class WorkspaceSandbox {
    private final Path workspaceRoot;

    public boolean isAllowed(Path target) {
        return target.toRealPath().startsWith(workspaceRoot.toRealPath());
    }
}
```

Esto cubre:
- ✅ Prevenir `write_file("/etc/passwd", ...)`
- ✅ Prevenir path traversal (`../../.ssh/id_rsa`)
- ❌ No previene que `bash` haga lo que quiera

---

## Sandbox para Bash

El problema real es `bash`. Opciones:

| Estrategia | Cómo | Trade-off |
|------------|------|-----------|
| **Allowlist de comandos** | Solo permitir comandos aprobados | Muy restrictivo, rompe flujo |
| **Denylist** | Bloquear `rm -rf /`, `sudo`, etc. | Fácil de evadir |
| **Working dir lock** | `bash` siempre ejecuta con `cwd` = workspace | Razonable para mayoría de casos |
| **Docker execution** | Cada `bash` corre en container efímero | Seguro pero lento |
| **Análisis pre-ejecución** | Parsear comando antes de ejecutar | Complejo, falsos positivos |

---

## ¿Vale la pena?

| A favor | En contra |
|---------|-----------|
| Previene desastres accidentales | Agrega latencia y complejidad |
| Seguridad si el agente "alucina" comandos | Path validation cubre 90% de los casos |
| Necesario si múltiples usuarios usan el agente | Para uso personal, overkill |
| Requerido para modo "auto" sin supervisión | Con confirmación humana, menos crítico |

---

## Recomendación Tentativa

**Fase 1:** Path validation + working dir lock para bash. Cubre el 90% sin complejidad.  
**Fase 2 (si se necesita):** Docker execution para bash en modo "auto" sin supervisión.

---

## Temas a Definir

- [ ] ¿Sandbox obligatorio o configurable por proyecto?
- [ ] ¿Qué pasa con tools que necesitan acceso fuera del workspace? (git global config, npm global)
- [ ] Network sandbox: ¿restringir acceso a internet del agente?
- [ ] Filesystem snapshot antes de cada sesión (para rollback completo)
- [ ] Integración con el permission gate de Tools
