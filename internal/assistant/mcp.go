package assistant

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ToolArgs struct {
	ID string `json:"id"`
}

// ServeMCP exposes only reads from one operator-selected snapshot over stdio.
func MCPServer(s *Session) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "logflow-evidence", Version: "1.0.0"}, nil)
	for _, name := range []string{"get_finding", "get_evidence", "get_runbook_section"} {
		tool := name
		mcp.AddTool(server, &mcp.Tool{Name: tool, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}, Description: "Read bounded evidence or an approved project runbook section. No writes or external requests."}, func(ctx context.Context, req *mcp.CallToolRequest, args ToolArgs) (*mcp.CallToolResult, any, error) {
			b, _ := json.Marshal(args)
			value, err := s.Call(tool, b)
			slog.New(slog.NewJSONHandler(os.Stderr, nil)).Info("mcp_read", "tool", tool, "snapshot", s.Snapshot, "id", args.ID, "allowed", err == nil)
			if err != nil {
				return nil, nil, err
			}
			var decoded any
			err = json.Unmarshal(value, &decoded)
			return nil, decoded, err
		})
	}
	return server
}

func ServeMCP(ctx context.Context, s *Session) error {
	return MCPServer(s).Run(ctx, &mcp.StdioTransport{})
}
