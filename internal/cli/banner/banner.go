package banner

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleLogo   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	styleInfo   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleOnline = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	stylePing   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

const logo = `
 ██████╗ ███████╗██████╗ ██╗███████╗      ██████╗██╗     ██╗
 ██╔══██╗██╔════╝██╔══██╗██║██╔════╝     ██╔════╝██║     ██║
 ██████╔╝█████╗  ██║  ██║██║███████╗     ██║     ██║     ██║
 ██╔══██╗██╔══╝  ██║  ██║██║╚════██║     ██║     ██║     ██║
 ██║  ██║███████╗██████╔╝██║███████║     ╚██████╗███████╗██║
 ╚═╝  ╚═╝╚══════╝╚═════╝ ╚═╝╚══════╝      ╚═════╝╚══════╝╚═╝`

// Print prints the startup banner to stdout.
func Print(host string, port int, latencyMs int64, version string) {
	fmt.Println(styleLogo.Render(logo))
	fmt.Println()
	status := styleOnline.Render("● connected")
	addr := styleInfo.Render(fmt.Sprintf("%s:%d", host, port))
	lat := stylePing.Render(fmt.Sprintf("%dms", latencyMs))
	ver := styleInfo.Render(version)
	fmt.Printf("  %s  %s  ping %s  redis %s\n\n", status, addr, lat, ver)
}
