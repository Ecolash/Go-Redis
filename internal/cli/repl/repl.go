package repl

import (
	"fmt"
	"os"
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
	styleQueued  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	styleTip     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleCmdName = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	styleCat     = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
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

	// prefixColor switches to Yellow in TX mode so the user can see state change.
	prefixColor := func() goprompt.Color {
		if sess.InTx {
			return goprompt.Yellow
		}
		return goprompt.Red
	}

	p := goprompt.New(
		executor,
		comp.Complete,
		goprompt.OptionPrefix("redis> "),
		goprompt.OptionLivePrefix(func() (string, bool) { return buildPrompt(sess) }),
		goprompt.OptionTitle("redis-cli"),
		goprompt.OptionPrefixTextColor(prefixColor()),
		goprompt.OptionPreviewSuggestionTextColor(goprompt.Blue),
		goprompt.OptionSelectedSuggestionBGColor(goprompt.LightGray),
		goprompt.OptionSuggestionBGColor(goprompt.DarkGray),
	)
	p.Run()
}

func buildPrompt(sess *state.Session) (string, bool) {
	addr := fmt.Sprintf("%s:%d", sess.Host, sess.Port)
	lat := fmt.Sprintf("[%dms]", sess.Latency.Milliseconds())
	if sess.InTx {
		return fmt.Sprintf("TX› %s %s ", addr, lat), true
	}
	return fmt.Sprintf("redis %s %s › ", addr, lat), true
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
		os.Exit(0)
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
		// Drain ch until the goroutine exits and closes it.
		// This guarantees the read deadline has been reset before we send UNSUBSCRIBE.
		for range ch {}
		unsubArgs := append([]string{"UNSUBSCRIBE"}, channels...)
		c.Do(unsubArgs...) //nolint:errcheck
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
