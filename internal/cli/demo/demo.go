package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/client"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/renderer"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/state"
)

var (
	stylePrompt  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	styleCommand = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Bold(true)
	styleSection = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true).Underline(true)
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type step struct {
	section string // printed as a header if non-empty
	cmd     string // raw command line
}

var script = []step{
	{section: "── Basics ──────────────────────────────────"},
	{cmd: "PING"},
	{cmd: "ECHO hello"},
	{cmd: "SET user:1 alice EX 3600"},
	{cmd: "GET user:1"},
	{cmd: "SET counter 10"},
	{cmd: "INCR counter"},
	{cmd: "DECR counter"},

	{section: "── Keys & Types ────────────────────────────"},
	{cmd: "KEYS *"},
	{cmd: "TYPE user:1"},

	{section: "── Lists ───────────────────────────────────"},
	{cmd: "RPUSH fruits apple banana cherry"},
	{cmd: "LRANGE fruits 0 -1"},
	{cmd: "LLEN fruits"},
	{cmd: "LPOP fruits"},

	{section: "── Sorted Sets ─────────────────────────────"},
	{cmd: "ZADD leaderboard 100 alice 200 bob 150 carol"},
	{cmd: "ZRANGE leaderboard 0 -1"},
	{cmd: "ZRANK leaderboard alice"},
	{cmd: "ZSCORE leaderboard bob"},

	{section: "── Streams ─────────────────────────────────"},
	{cmd: "XADD events * user alice action login"},
	{cmd: "XADD events * user bob action signup"},
	{cmd: "XRANGE events - +"},

	{section: "── GEO ─────────────────────────────────────"},
	{cmd: "GEOADD cities 13.361389 38.115556 Palermo"},
	{cmd: "GEOADD cities 15.087269 37.502669 Catania"},
	{cmd: "GEODIST cities Palermo Catania km"},
	{cmd: "GEOPOS cities Palermo"},

	{section: "── Transactions ────────────────────────────"},
	{cmd: "MULTI"},
	{cmd: "SET tx:balance 1000"},
	{cmd: "INCR tx:balance"},
	{cmd: "GET tx:balance"},
	{cmd: "EXEC"},
}

// Run executes the scripted demo against the given client.
func Run(c *client.Client, sess *state.Session) {
	prompt := stylePrompt.Render("redis") + " " +
		styleDim.Render(fmt.Sprintf("%s:%d", sess.Host, sess.Port)) + " › "

	sleep := func() { time.Sleep(500 * time.Millisecond) }

	for _, s := range script {
		if s.section != "" {
			fmt.Println()
			fmt.Println(styleSection.Render(s.section))
			sleep()
			continue
		}

		// Print the command as if typed
		fmt.Printf("%s%s\n", prompt, styleCommand.Render(s.cmd))

		tokens := strings.Fields(s.cmd)
		cmd := strings.ToUpper(tokens[0])

		// Handle TX state so EXEC renders correctly
		switch cmd {
		case "MULTI":
			sess.EnterTx()
		case "EXEC", "DISCARD":
			defer sess.ExitTx()
		}

		resp, err := c.Do(tokens...)
		if err != nil {
			fmt.Println(renderer.Render(&client.RESPValue{Type: '-', Value: err.Error()}, cmd))
		} else {
			renderCmd := cmd
			upper := strings.ToUpper(s.cmd)
			if cmd == "ZRANGE" && strings.Contains(upper, "WITHSCORES") {
				renderCmd = "ZRANGEWITHSCORES"
			}
			fmt.Println(renderer.Render(resp, renderCmd))
		}

		sleep()
	}

	fmt.Println()
	fmt.Println(styleDim.Render("── demo complete ──"))
	fmt.Println()
}
