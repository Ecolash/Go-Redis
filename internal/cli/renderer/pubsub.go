package renderer

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/client"
)

var (
	styleTS      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleChan    = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)
	styleMsg     = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	stylePattern = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	styleFeedHdr = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
)

type msgReceived struct{ msg client.Message }
type feedDone struct{}

type feedModel struct {
	lines  []string
	ch     <-chan client.Message
	cancel func()
	done   bool
}

func newFeedModel(ch <-chan client.Message, cancel func()) feedModel {
	return feedModel{ch: ch, cancel: cancel}
}

func (m feedModel) Init() tea.Cmd {
	return m.waitForMsg()
}

func (m feedModel) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.ch
		if !ok {
			return feedDone{}
		}
		return msgReceived{msg}
	}
}

func (m feedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgReceived:
		ts := styleTS.Render(time.Now().Format("15:04:05"))
		ch := styleChan.Render(msg.msg.Channel)
		payload := styleMsg.Render(msg.msg.Payload)
		line := fmt.Sprintf("🔔 [%s] %s › %s", ts, ch, payload)
		if msg.msg.Kind == "pmessage" {
			pat := stylePattern.Render("(" + msg.msg.Pattern + ")")
			line = fmt.Sprintf("🔔 [%s] %s %s › %s", ts, ch, pat, payload)
		}
		m.lines = append(m.lines, line)
		return m, m.waitForMsg()
	case feedDone:
		m.done = true
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.cancel()
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m feedModel) View() string {
	if m.done {
		return ""
	}
	hdr := styleFeedHdr.Render("── Pub/Sub live feed (Ctrl+C to exit) ──")
	return hdr + "\n" + strings.Join(m.lines, "\n") + "\n"
}

// RunPubSubFeed starts the Bubble Tea program for live Pub/Sub display.
// Blocks until the user presses Ctrl+C or the channel closes.
func RunPubSubFeed(ch <-chan client.Message, cancel func()) error {
	p := tea.NewProgram(newFeedModel(ch, cancel))
	_, err := p.Run()
	return err
}
