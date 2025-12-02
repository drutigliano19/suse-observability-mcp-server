package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/drutigliano19/suse-observability-mcp/internal/stackstate"
	"github.com/drutigliano19/suse-observability-mcp/internal/tools"
)

func main() {
	// Stackstate flags
	stsApiURL := flag.String("url", "", "SUSE Observability API URL")
	stsApiKey := flag.String("key", "", "SUSE Observability API Key")
	stsApiToken := flag.String("token", "", "SUSE Observability API Token")
	stsApiTokenType := flag.String("tokentype", "", "SUSE Observability API Token type")
	stsLegacyApi := flag.Bool("legacy", false, "")

	// MCP server flags
	listenAddr := flag.String("http", "", "address for http transport, defaults to stdio")
	flag.Parse()

	server := mcp.NewServer(&mcp.Implementation{Name: "SUSE Observability MCP server", Version: "v0.0.1"}, nil)

	tools = tools.NewTools(&stackstate.StackState{
		ApiUrl:       *stsApiURL,
		ApiKey:       *stsApiKey,
		ApiToken:     *stsApiToken,
		ApiTokenType: *stsApiTokenType,
		LegacyApi:    *stsLegacyApi,
	})


	if *listenAddr == "" {
		// Run the server on the stdio transport.
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			slog.Error("Server failed", "error", err)
		}
	} else {
		// Create a streamable HTTP handler.
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)

		// Run the server on the HTTP transport.
		slog.Info("Server listening", "address", *listenAddr)
		if err := http.ListenAndServe(*listenAddr, handler); err != nil {
			slog.Error("Server failed", "error", err)
		}
	}
}