# Story 05 — TUI con Bubble Tea

## Como usuario
Quiero una interfaz de terminal con input estilizado, historial de comandos, y output formateado — no un scanner de stdin crudo.

## Criterios de Aceptación

- [ ] Bubble Tea program como loop principal (reemplaza stdin scanner)
- [ ] Input box con historial (flecha arriba/abajo)
- [ ] Banner de inicio con nombre + modelo activo
- [ ] Output del agente renderizado en viewport scrollable
- [ ] Spinner mientras el modelo piensa
- [ ] Ctrl+C / `/exit` sale limpiamente
- [ ] stdout del agente y tools se redirige al viewport (no se mezcla con TUI)

## Archivos a Crear

```
internal/ui/program.go    # Bubble Tea Model + Update + View
internal/ui/input.go      # Input component con historial
internal/ui/banner.go     # Banner de inicio
internal/ui/spinner.go    # Spinner de loading
internal/ui/styles.go     # Lipgloss styles
```

## Definición de Done

- El binario muestra banner al arrancar
- Se puede escribir, enviar con Enter, ver respuesta formateada
- Historial funciona con flechas
- Spinner visible mientras espera respuesta del modelo
