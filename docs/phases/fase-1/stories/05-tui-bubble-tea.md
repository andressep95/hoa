# Story 05 — TUI con Bubble Tea ✅

## Como usuario
Quiero una interfaz de terminal con input estilizado, historial de comandos, y output formateado — no un scanner de stdin crudo.

## Criterios de Aceptación

- [x] Bubble Tea program como loop principal (reemplaza stdin scanner)
- [x] Input box con historial (flecha arriba/abajo)
- [x] Banner de inicio dinámico (refleja provider/modelo/modo actual)
- [x] Output del agente renderizado en viewport scrollable (PageUp/PageDown)
- [x] Spinner mientras el modelo piensa
- [x] Ctrl+C / `/exit` sale limpiamente
- [x] stdout del agente y tools se redirige al viewport via channel (no se mezcla con TUI)
- [x] Menús interactivos inline (↑↓ navegar, Enter seleccionar, Esc cancelar)
- [x] Autocomplete dropdown filtrable al escribir `/`
- [x] Separador visual `⎿` entre comandos y resultados

## Archivos Implementados

```
internal/ui/program.go    # Bubble Tea Model + Update + View + menus + autocomplete + scroll
internal/ui/styles.go     # Lipgloss styles centralizados
internal/ui/textinput.go  # Input component (wizard)
internal/ui/selector.go   # Selector component (wizard)
```

## Definición de Done ✅

- Alt-screen con banner dinámico
- Input con historial de comandos
- Spinner visible mientras espera respuesta
- Menús interactivos para /model, /provider, /mode
- Autocomplete con dropdown que filtra al escribir
- PageUp/PageDown para scroll del historial
