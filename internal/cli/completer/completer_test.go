package completer_test

import (
	"testing"

	goprompt "github.com/c-bata/go-prompt"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/completer"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/state"
)

func TestCompleteCommandPrefix(t *testing.T) {
	s := state.New("127.0.0.1", 6379)
	c := completer.New(s)
	doc := goprompt.Document{Text: "GE"}
	suggestions := c.Complete(doc)
	names := make([]string, len(suggestions))
	for i, sg := range suggestions {
		names[i] = sg.Text
	}
	if !contains(names, "GET") {
		t.Errorf("expected GET in suggestions for 'GE', got %v", names)
	}
}

func TestNoCompleteInPubSub(t *testing.T) {
	s := state.New("127.0.0.1", 6379)
	s.Subscribe("ch")
	c := completer.New(s)
	doc := goprompt.Document{Text: "G"}
	if len(c.Complete(doc)) != 0 {
		t.Error("expected no suggestions while in PubSub mode")
	}
}

func TestTypoSuggestion(t *testing.T) {
	s := state.New("127.0.0.1", 6379)
	c := completer.New(s)
	got := c.SuggestTypo("GETT")
	if got != "GET" {
		t.Errorf("expected GET for typo GETT, got %q", got)
	}
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
