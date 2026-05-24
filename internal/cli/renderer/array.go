package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/client"
)

var (
	styleBullet = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styleIndex  = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
)

func renderArray(v *client.RESPValue, cmd string) string {
	if v == nil || v.IsNull {
		return renderNull()
	}
	if len(v.Items) == 0 {
		return styleNil.Render("(empty array)")
	}

	switch cmd {
	case "EXEC":
		return renderExec(v)
	case "ZRANGE", "ZRANGEBYSCORE", "ZRANGEWITHSCORES":
		return renderZRange(v)
	case "GEOPOS", "GEODIST", "GEOSEARCH":
		return renderGeo(v, cmd)
	case "XRANGE", "XREAD":
		return renderStream(v, cmd)
	default:
		return renderGenericArray(v)
	}
}

func renderGenericArray(rv *client.RESPValue) string {
	var sb strings.Builder
	for i, item := range rv.Items {
		bullet := styleBullet.Render("▸")
		sb.WriteString(fmt.Sprintf("%s %d) %s\n", bullet, i+1, renderItem(item)))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderExec(rv *client.RESPValue) string {
	var sb strings.Builder
	for i, item := range rv.Items {
		idx := styleIndex.Render(fmt.Sprintf("[%d]", i+1))
		sb.WriteString(fmt.Sprintf("%s %s\n", idx, Render(item, "")))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderItem(v *client.RESPValue) string {
	if v == nil || v.IsNull {
		return renderNull()
	}
	switch v.Type {
	case '+':
		return renderSimpleString(v.Value)
	case '-':
		return renderErr(v.Value)
	case ':':
		return renderInteger(v.Value)
	case '$':
		return renderBulk(v.Value)
	case '*':
		return renderGenericArray(v)
	}
	return ""
}

func renderZRange(rv *client.RESPValue) string   { return renderZRangeWithScores(rv) }
func renderGeo(rv *client.RESPValue, cmd string) string { return renderGeoResult(rv, cmd) }
func renderStream(rv *client.RESPValue, cmd string) string {
	if cmd == "XREAD" {
		return renderXRead(rv)
	}
	return renderStreamEntries(rv)
}
