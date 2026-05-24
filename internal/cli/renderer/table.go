package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/client"
)

var (
	styleHeader  = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true).Underline(true)
	styleCell    = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleScore   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleGeoName = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
)

// renderZRangeWithScores renders alternating member/score pairs as a table.
func renderZRangeWithScores(rv *client.RESPValue) string {
	if len(rv.Items)%2 != 0 {
		return renderGenericArray(rv)
	}
	var sb strings.Builder
	header := fmt.Sprintf("%-30s  %s",
		styleHeader.Render("member"),
		styleHeader.Render("score"))
	sb.WriteString(header + "\n")
	sb.WriteString(styleDim.Render(strings.Repeat("─", 44)) + "\n")
	for i := 0; i < len(rv.Items); i += 2 {
		member := styleCell.Render(fmt.Sprintf("%-30s", rv.Items[i].Value))
		score := styleScore.Render(rv.Items[i+1].Value)
		sb.WriteString(fmt.Sprintf("%s  %s\n", member, score))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// renderGeoResult renders GEO command output as a labelled table.
func renderGeoResult(rv *client.RESPValue, cmd string) string {
	var sb strings.Builder
	switch cmd {
	case "GEOPOS":
		sb.WriteString(styleHeader.Render("longitude") + "  " + styleHeader.Render("latitude") + "\n")
		sb.WriteString(styleDim.Render(strings.Repeat("─", 36)) + "\n")
		for _, item := range rv.Items {
			if item == nil || item.IsNull {
				sb.WriteString(styleNil.Render("(nil)") + "\n")
				continue
			}
			if item.Type == '*' && len(item.Items) == 2 {
				lon := item.Items[0].Value
				lat := item.Items[1].Value
				sb.WriteString(fmt.Sprintf("%s  %s\n",
					styleGeoName.Render(fmt.Sprintf("%-18s", lon)),
					styleCell.Render(lat)))
			}
		}
	default:
		return renderGenericArray(rv)
	}
	return strings.TrimRight(sb.String(), "\n")
}
