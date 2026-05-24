package renderer

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleOK   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	styleErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	styleNil  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	styleInt  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleBulk = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	styleRaw  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func renderSimpleString(s string) string {
	if s == "OK" {
		return styleOK.Render("✓ OK")
	}
	return styleOK.Render("✓ " + s)
}

func renderErr(s string) string {
	return styleErr.Render("✗ " + s)
}

func renderNull() string {
	return styleNil.Render("(nil)")
}

func renderInteger(s string) string {
	return styleInt.Render(fmt.Sprintf("(integer) %s", s))
}

func renderBulk(s string) string {
	return styleBulk.Render(s)
}

func renderRaw(v interface{}) string {
	return styleRaw.Render(fmt.Sprintf("⚠ Unexpected response: %v", v))
}
