package repl

import (
	"fmt"
	"strings"

	goprompt "github.com/c-bata/go-prompt"
	"github.com/charmbracelet/lipgloss"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/client"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/completer"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/registry"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/renderer"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/state"
)

var (
	stylePromptNormal = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	stylePromptTX     = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	styleQueued       = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	styleLatency      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleTip          = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleWarn         = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleCmdName      = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	styleCat          = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
)

// Run starts the interactive REPL. Blocks until the user quits.
func Run(c *client.Client, sess *state.Session) {
	comp := completer.New(sess)

	executor := func(input string) {
		input = strings.TrimSpace(input)
		if input == "" {
			return
		}
		handleInput(input, c, sess, comp)
	}

	p := goprompt.New(
		executor,
		comp.Complete,
		goprompt.OptionPrefix("redis> "),
		goprompt.OptionLivePrefix(func() (string, bool) { return buildPrompt(sess) }),
		goprompt.OptionTitle("redis-cli"),
		goprompt.OptionPrefixTextColor(goprompt.Red),
		goprompt.OptionPreviewSuggestionTextColor(goprompt.Blue),
		goprompt.OptionSelectedSuggestionBGColor(goprompt.LightGray),
		goprompt.OptionSuggestionBGColor(goprompt.DarkGray),
	)
	p.Run()
}

func buildPrompt(sess *state.Session) (string, bool) {
	lat := styleLatency.Render(fmt.Sprintf("[%dms]", sess.Latency.Milliseconds()))
	addr := fmt.Sprintf("%s:%d", sess.Host, sess.Port)
	if sess.InTx {
		return stylePromptTX.Render(fmt.Sprintf("TX› %s %s ", addr, lat)), true
	}
	return stylePromptNormal.Render(fmt.Sprintf("redis %s %s › ", addr, lat)), true
}

func handleInput(input string, c *client.Client, sess *state.Session, comp *completer.Completer) {
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return
	}
	cmd := strings.ToUpper(tokens[0])

	// Meta commands handled locally
	switch cmd {
	case "HELP":
		printHelp(tokens)
		return
	case "QUIT", "EXIT", "BYE":
		fmt.Println(styleTip.Render("Goodbye! 👋"))
		c.Close()
		return
	}

	// Pub/Sub entry — hands off to Bubble Tea feed
	if cmd == "SUBSCRIBE" || cmd == "PSUBSCRIBE" {
		channels := tokens[1:]
		if len(channels) == 0 {
			fmt.Println(styleWarn.Render("✗ Usage: SUBSCRIBE channel [channel ...]"))
			return
		}
		ch, cancel, err := c.Subscribe(channels)
		if err != nil {
			fmt.Println(renderer.Render(&client.RESPValue{Type: '-', Value: err.Error()}, cmd))
			return
		}
		for _, channel := range channels {
			sess.Subscribe(channel)
		}
		if err := renderer.RunPubSubFeed(ch, cancel); err != nil {
			fmt.Println(styleWarn.Render("✗ PubSub feed error: " + err.Error()))
		}
		for _, channel := range channels {
			sess.Unsubscribe(channel)
		}
		return
	}

	// TX state machine
	switch cmd {
	case "MULTI":
		sess.EnterTx()
	case "EXEC", "DISCARD":
		defer sess.ExitTx()
	}

	// Show queued indicator while inside a transaction
	if sess.InTx && cmd != "MULTI" && cmd != "EXEC" && cmd != "DISCARD" {
		sess.QueueCmd(input)
		fmt.Println(styleQueued.Render("  ⬡ QUEUED"))
	}

	resp, err := c.Do(tokens...)
	if err != nil {
		fmt.Println(renderer.Render(&client.RESPValue{Type: '-', Value: err.Error()}, cmd))
		return
	}
	sess.UpdateLatency(c.Latency())

	// Typo hint on unknown command errors
	if resp.Type == '-' && strings.Contains(resp.Value, "unknown command") {
		if suggestion := comp.SuggestTypo(cmd); suggestion != cmd {
			fmt.Println(styleWarn.Render(fmt.Sprintf("  Did you mean: %s?", suggestion)))
		}
	}

	fmt.Println(renderer.Render(resp, buildRenderCmd(cmd, tokens)))
}

// buildRenderCmd returns a context key for the renderer.
func buildRenderCmd(cmd string, tokens []string) string {
	upper := strings.ToUpper(strings.Join(tokens, " "))
	if cmd == "ZRANGE" && strings.Contains(upper, "WITHSCORES") {
		return "ZRANGEWITHSCORES"
	}
	return cmd
}

func printHelp(tokens []string) {
	fmt.Println(styleTip.Render("─── Redis CLI Help ───"))
	if len(tokens) > 1 {
		def := registry.Find(tokens[1])
		if def == nil {
			fmt.Println(styleWarn.Render("✗ Unknown command: " + strings.ToUpper(tokens[1])))
			return
		}
		fmt.Printf("  %s\n  Synopsis: %s\n  Example:  %s\n",
			styleCmdName.Render(def.Name),
			def.Synopsis,
			def.Example)
		return
	}
	categories := map[string][]string{}
	for _, def := range registry.All() {
		categories[def.Category] = append(categories[def.Category], def.Name)
	}
	order := []string{"basic", "string", "list", "stream", "sorted-set", "geo", "pubsub", "tx", "server", "replication", "meta"}
	for _, cat := range order {
		cmds, ok := categories[cat]
		if !ok {
			continue
		}
		fmt.Printf("  %s: %s\n",
			styleCat.Render(cat),
			strings.Join(cmds, "  "))
	}
}
