# SUSE Observability MCP Server

## Description
The SUSE Observability Model Context Protocol (MCP) Server is a specialized, middle-tier API designed to translate the complex, high-cardinality observability data from StackState (topology, metrics, and events) into highly structured, contextually rich, and LLM-ready snippets.

This MCP Server abstracts the StackState APIs. Its primary function is to serve as a Tool/Function Calling target for AI agents. When an AI receives an alert or a user query (e.g., "What caused the outage?"), the AI calls an MCP Server endpoint. The server then fetches the relevant operational facts, summarizes them, normalizes technical identifiers (like URNs and raw metric names) into natural language concepts, and returns a concise JSON or YAML payload. This payload is then injected directly into the LLM's prompt, ensuring the final diagnosis or action is grounded in real-time, accurate SUSE Observability data, effectively minimizing hallucinations.

## Goals
- **Grounding AI Responses**: Ensure that all AI diagnoses, root cause analyses, and action recommendations are strictly based on verifiable, real-time data retrieved from the SUSE Observability StackState platform.
- **Simplifying Data Access**: Abstract the complexity of StackState's native APIs (e.g., Time Travel, 4T Data Model) into simple, semantic functions that can be easily invoked by LLM tool-calling mechanisms.
- **Data Normalization**: Convert complex, technical identifiers (like component URNs, raw metric names, and proprietary health states) into standardized, natural language terms that an LLM can easily reason over.
- **Enabling Automated Remediation**: Define clear, action-oriented MCP endpoints that allow the AI agent to initiate automated operational workflows.

## Available Tools

The server currently exposes the following tools for AI agents:

### Metrics Tools

-   **`listMetrics`**: Lists bound metrics for a specific component.
    -   Arguments: `component_id` (integer, required): The ID of the component to list bound metrics for (from topology queries)
    -   Returns: A markdown table showing the bound metrics with their names, units, and query expressions

-   **`getMetrics`**: Query metrics from SUSE Observability over a range of time.
    -   Arguments: 
        - `query` (string, required): The PromQL query to execute
        - `start` (string, required): Start time for the query (e.g., 'now', '1h')
        - `end` (string, required): End time for the query (e.g., 'now', '1h')
        - `step` (string, optional): Query resolution step width (e.g., '15s', '1m', defaults to '1m')
    -   Returns: A markdown table with the visual representation of the query result

### Monitors Tools

-   **`listMonitors`**: Lists monitors for a specific component.
    -   Arguments: `component_id` (integer, required): The ID of the component to list monitors for (from topology queries)
    -   Returns: A markdown table showing monitors associated with the specified component and their current states

### Topology Tools

-   **`getComponents`**: Searches for topology components using STQL filters.
    -   Arguments (all support comma-separated values for multiple items):
        - `names` (string, optional): Component names to match exactly (comma-separated, e.g., 'checkout-service,redis-master')
        - `types` (string, optional): Component types (comma-separated, e.g., 'pod,service,deployment')
        - `healthstates` (string, optional): Health states (comma-separated, e.g., 'CRITICAL,DEVIATING'). Particularly useful to query multiple states at once
        - `with_neighbors` (boolean, optional): Include connected components using withNeighborsOf
        - `with_neighbors_levels` (string, optional): Number of levels (1-14) or 'all' (default: 1)
        - `with_neighbors_direction` (string, optional): 'up', 'down', or 'both' (default: 'both')
    -   Note: At least one filter must be provided. All filters use STQL IN operator for efficient multi-value queries
    -   Returns: A markdown table of matching components with their IDs and identifiers

## Build and Run

### Prerequisites
-   Go 1.23 or later

### Build
To build the server, run:
```bash
go build -o suse-observability-mcp-server cmd/server/main.go
```

### Run
To run the server, you need to provide the SUSE Observability API details. You can run it using stdio (default) or HTTP.

**Using Stdio (for MCP clients):**
```bash
./suse-observability-mcp-server \
  -url "https://your-instance.suse.observability.com" \
  -token "YOUR_API_TOKEN" \
  -apitoken
```

**Using HTTP:**
```bash
./suse-observability-mcp-server \
  -http ":8080" \
  -url "https://your-instance.suse.observability.com" \
  -token "YOUR_API_TOKEN" \
  -apitoken
```

### Configuration Flags
-   `-http`: Address for HTTP transport (e.g., ":8080"). If empty, defaults to stdio.
-   `-url`: SUSE Observability API URL
-   `-token`: SUSE Observability API Token
-   `-apitoken`: Use SUSE Observability API Token instead of a Service Token (boolean)

## Resources
*   [Honeycomb: End of Observability](https://www.honeycomb.io/blog/its-the-end-of-observability-as-we-know-it-and-i-feel-fine)
*   [Datadog Remote MCP Server](https://www.datadoghq.com/blog/datadog-remote-mcp-server)
*   [Model Context Protocol Specification](https://modelcontextprotocol.io/specification/2025-06-18/index)
