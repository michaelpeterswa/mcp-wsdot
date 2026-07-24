package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"alpineworks.io/wsdot"
	"alpineworks.io/wsdot/vessels"
	"github.com/mark3labs/mcp-go/mcp"
)

type WSDOTHandlerClient struct {
	wsdotClient   *wsdot.WSDOTClient
	vesselsClient *vessels.VesselsClient
}

func NewWSDOTHandlerClient(w *wsdot.WSDOTClient) (*WSDOTHandlerClient, error) {
	vesselsClient, err := vessels.NewVesselsClient(w)
	if err != nil {
		return nil, fmt.Errorf("could not create vessels client: %w", err)
	}

	return &WSDOTHandlerClient{
		wsdotClient:   w,
		vesselsClient: vesselsClient,
	}, nil
}

func (whc *WSDOTHandlerClient) GetRouteSchedulesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	routeSchedules, err := whc.vesselsClient.GetRouteSchedules()
	if err != nil {
		return nil, fmt.Errorf("could not get route schedules: %w", err)
	}

	resultJson, err := json.Marshal(routeSchedules)
	if err != nil {
		return nil, fmt.Errorf("could not marshal result: %w", err)
	}

	return mcp.NewToolResultText(string(resultJson)), nil
}

func (whc *WSDOTHandlerClient) GetSchedulesTodayByRouteIDHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	routeID := mcp.ParseInt(request, "routeID", -1)
	onlyRemainingTime := mcp.ParseBoolean(request, "onlyRemainingTime", false)

	schedulesToday, err := whc.vesselsClient.GetSchedulesTodayByRouteID(routeID, onlyRemainingTime)
	if err != nil {
		return nil, fmt.Errorf("could not get schedules today by route ID: %w", err)
	}

	resultJson, err := json.Marshal(schedulesToday)
	if err != nil {
		return nil, fmt.Errorf("could not marshal result: %w", err)
	}

	return mcp.NewToolResultText(string(resultJson)), nil
}
