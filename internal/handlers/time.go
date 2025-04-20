package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

type CurrentTime struct {
	CurrentTime time.Time `json:"current_time"`
}

func CurrentTimeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return nil, fmt.Errorf("could not load location: %w", err)
	}

	resultJson, err := json.Marshal(CurrentTime{
		CurrentTime: time.Now().In(loc),
	})
	if err != nil {
		return nil, fmt.Errorf("could not marshal result: %w", err)
	}

	return mcp.NewToolResultText(string(resultJson)), nil
}
