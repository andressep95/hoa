package ui

import "github.com/charmbracelet/lipgloss"

var (
	StylePrompt   = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	StyleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	StyleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	StyleTool     = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	StyleTitle    = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	StyleSubtitle = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
)
