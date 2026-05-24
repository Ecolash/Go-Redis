package completer

import (
	"strings"

	goprompt "github.com/c-bata/go-prompt"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/registry"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/state"
)

// Completer implements go-prompt autocomplete.
type Completer struct {
	sess *state.Session
}

func New(sess *state.Session) *Completer {
	return &Completer{sess: sess}
}

// Complete is the go-prompt Completer callback.
func (c *Completer) Complete(doc goprompt.Document) []goprompt.Suggest {
	if c.sess.InPubSub {
		return nil
	}
	text := doc.Text
	tokens := strings.Fields(text)
	if len(tokens) == 0 || (len(tokens) == 1 && !strings.HasSuffix(text, " ")) {
		return c.suggestCommands(text)
	}
	return c.suggestArgs(tokens)
}

func (c *Completer) suggestCommands(prefix string) []goprompt.Suggest {
	upper := strings.ToUpper(prefix)
	var out []goprompt.Suggest
	for _, def := range registry.All() {
		if strings.HasPrefix(def.Name, upper) {
			label := "[" + def.Category + "]"
			if c.sess.InTx {
				label = "⬡ QUEUED " + label
			}
			out = append(out, goprompt.Suggest{
				Text:        def.Name,
				Description: label + " " + def.Synopsis,
			})
		}
	}
	return out
}

func (c *Completer) suggestArgs(tokens []string) []goprompt.Suggest {
	def := registry.Find(tokens[0])
	if def == nil || len(def.Args) == 0 {
		return nil
	}
	argIdx := len(tokens) - 2 // 0-indexed into Args (tokens[0] is the command)
	if argIdx < 0 || argIdx >= len(def.Args) {
		return nil
	}
	arg := def.Args[argIdx]
	return []goprompt.Suggest{{Text: "<" + arg.Name + ">", Description: arg.Hint}}
}

// SuggestTypo returns the closest registered command name to the given typo.
func (c *Completer) SuggestTypo(input string) string {
	upper := strings.ToUpper(input)
	best := ""
	bestDist := 1<<31 - 1
	for _, def := range registry.All() {
		d := levenshtein(upper, def.Name)
		if d < bestDist {
			bestDist = d
			best = def.Name
		}
	}
	return best
}

func levenshtein(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	la := len(ra)
	lb := len(rb)
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			dp[i][j] = min3(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}
	return dp[la][lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
