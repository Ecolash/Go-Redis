package renderer

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/cli/client"
)

// Render dispatches a RESPValue to the appropriate sub-renderer.
// cmd is the command that produced this response (used for context-aware rendering).
func Render(v *client.RESPValue, cmd string) string {
	if v == nil || v.IsNull {
		return renderNull()
	}
	upper := strings.ToUpper(cmd)
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
		return renderArray(v, upper)
	default:
		return renderRaw(v)
	}
}
