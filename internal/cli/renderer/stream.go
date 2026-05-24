package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/client"
)

var (
	styleStreamID   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	styleFieldKey   = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styleFieldValue = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	styleRule       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func renderStreamEntries(rv *client.RESPValue) string {
	var sb strings.Builder
	for i, item := range rv.Items {
		if i > 0 {
			sb.WriteString(styleRule.Render(strings.Repeat("─", 40)) + "\n")
		}
		if item.Type != '*' || len(item.Items) < 2 {
			sb.WriteString(renderItem(item) + "\n")
			continue
		}
		id := styleStreamID.Render("⏱ " + item.Items[0].Value)
		sb.WriteString(id + "\n")
		fields := item.Items[1]
		if fields.Type == '*' {
			for j := 0; j+1 < len(fields.Items); j += 2 {
				k := styleFieldKey.Render(fields.Items[j].Value)
				val := styleFieldValue.Render(fields.Items[j+1].Value)
				sb.WriteString(fmt.Sprintf("  %s: %s\n", k, val))
			}
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderXRead(rv *client.RESPValue) string {
	var sb strings.Builder
	for _, item := range rv.Items {
		if item.Type != '*' || len(item.Items) < 2 {
			continue
		}
		streamKey := styleStreamID.Render("▶ stream: " + item.Items[0].Value)
		sb.WriteString(streamKey + "\n")
		sb.WriteString(renderStreamEntries(item.Items[1]) + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
