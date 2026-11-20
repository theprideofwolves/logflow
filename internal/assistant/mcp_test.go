package assistant

import (
	"context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"testing"
	"time"
)

func TestMCPReadsOnlySelectedSnapshot(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s := fixture(t)
	left, right := mcp.NewInMemoryTransports()
	server, err := MCPServer(s).Connect(ctx, left, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "regression", Version: "1"}, nil)
	session, err := client.Connect(ctx, right, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 3 {
		t.Fatal(tools, err)
	}
	for _, tool := range tools.Tools {
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Fatal(tool)
		}
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_evidence", Arguments: map[string]string{"id": s.Alert.ID}})
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "get_runbook_section", Arguments: map[string]string{"id": "../../.env"}})
	if err == nil && !result.IsError {
		t.Fatal("read escaped approved corpus")
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "send_jira", Arguments: map[string]string{"id": s.Alert.ID}})
	if err == nil && !result.IsError {
		t.Fatal("write tool exposed")
	}
}
