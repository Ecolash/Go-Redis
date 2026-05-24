package renderer_test

import (
	"strings"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/cli/client"
	"github.com/codecrafters-io/redis-starter-go/internal/cli/renderer"
)

func TestRenderOK(t *testing.T) {
	v := &client.RESPValue{Type: '+', Value: "OK"}
	out := renderer.Render(v, "SET")
	if !strings.Contains(out, "OK") {
		t.Errorf("expected OK in output, got %q", out)
	}
	if !strings.Contains(out, "✓") {
		t.Errorf("expected checkmark in OK output, got %q", out)
	}
}

func TestRenderError(t *testing.T) {
	v := &client.RESPValue{Type: '-', Value: "ERR no such key"}
	out := renderer.Render(v, "GET")
	if !strings.Contains(out, "✗") {
		t.Errorf("expected ✗ in error output, got %q", out)
	}
	if !strings.Contains(out, "ERR no such key") {
		t.Errorf("expected error message in output, got %q", out)
	}
}

func TestRenderNull(t *testing.T) {
	v := &client.RESPValue{Type: '$', IsNull: true}
	out := renderer.Render(v, "GET")
	if !strings.Contains(out, "nil") {
		t.Errorf("expected nil in null output, got %q", out)
	}
}

func TestRenderInteger(t *testing.T) {
	v := &client.RESPValue{Type: ':', Value: "42"}
	out := renderer.Render(v, "INCR")
	if !strings.Contains(out, "42") {
		t.Errorf("expected 42 in integer output, got %q", out)
	}
}

func TestRenderArray(t *testing.T) {
	v := &client.RESPValue{
		Type: '*',
		Items: []*client.RESPValue{
			{Type: '$', Value: "alpha"},
			{Type: '$', Value: "beta"},
		},
	}
	out := renderer.Render(v, "KEYS")
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Errorf("expected array items in output, got %q", out)
	}
	if !strings.Contains(out, "▸") {
		t.Errorf("expected bullet ▸ in array output, got %q", out)
	}
}

func TestRenderNullArray(t *testing.T) {
	v := &client.RESPValue{Type: '*', IsNull: true}
	out := renderer.Render(v, "KEYS")
	if !strings.Contains(out, "nil") {
		t.Errorf("expected nil for null array, got %q", out)
	}
}

func TestRenderExecResult(t *testing.T) {
	v := &client.RESPValue{
		Type: '*',
		Items: []*client.RESPValue{
			{Type: '+', Value: "OK"},
			{Type: ':', Value: "1"},
		},
	}
	out := renderer.Render(v, "EXEC")
	if !strings.Contains(out, "[1]") || !strings.Contains(out, "[2]") {
		t.Errorf("expected indexed results for EXEC, got %q", out)
	}
}

func TestRenderZRangeWithScores(t *testing.T) {
	v := &client.RESPValue{
		Type: '*',
		Items: []*client.RESPValue{
			{Type: '$', Value: "alice"},
			{Type: '$', Value: "100"},
			{Type: '$', Value: "bob"},
			{Type: '$', Value: "200"},
		},
	}
	out := renderer.Render(v, "ZRANGEWITHSCORES")
	if !strings.Contains(out, "alice") || !strings.Contains(out, "100") {
		t.Errorf("expected member+score in table, got %q", out)
	}
}

func TestRenderStreamEntry(t *testing.T) {
	entry := &client.RESPValue{
		Type: '*',
		Items: []*client.RESPValue{
			{Type: '$', Value: "1-0"},
			{Type: '*', Items: []*client.RESPValue{
				{Type: '$', Value: "name"},
				{Type: '$', Value: "alice"},
			}},
		},
	}
	v := &client.RESPValue{Type: '*', Items: []*client.RESPValue{entry}}
	out := renderer.Render(v, "XRANGE")
	if !strings.Contains(out, "1-0") {
		t.Errorf("expected stream ID in output, got %q", out)
	}
	if !strings.Contains(out, "name") || !strings.Contains(out, "alice") {
		t.Errorf("expected field/value in output, got %q", out)
	}
}
