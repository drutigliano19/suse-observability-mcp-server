package tools

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/drutigliano19/suse-observability-mcp/internal/stackstate/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListMetricsParams struct {
	Filter string `json:"filter,omitempty" jsonschema:"Optional regex filter to search for specific metrics"`
}

type QueryMetricParams struct {
	Query string `json:"query" jsonschema:"The PromQL query to execute"`
}

type QueryRangeMetricParams struct {
	Query string `json:"query" jsonschema:"The PromQL query to execute"`
	Start string `json:"start" jsonschema:"Start time: 'now' or duration (e.g. '1h')"`
	End   string `json:"end" jsonschema:"End time: 'now' or duration (e.g. '1h')"`
	Step  string `json:"step" jsonschema:"Query resolution step width in duration format or float number of seconds"`
}

type MetricPoint struct {
	Timestamp int64   `json:"t"`
	Value     float64 `json:"v"`
}

type MetricSeries struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Points []MetricPoint     `json:"points"`
}

// ListMetrics lists all available metrics
func (t tool) ListMetrics(ctx context.Context, request *mcp.CallToolRequest, params ListMetricsParams) (*mcp.CallToolResult, any, error) {
	end := time.Now()
	start := end.Add(-1 * time.Hour)
	metrics, err := t.client.ListMetrics(ctx, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list metrics: %w", err)
	}

	// Filter metrics if filter param is provided
	var filteredMetrics []string
	if params.Filter != "" {
		re, err := regexp.Compile(params.Filter)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid regex filter: %w", err)
		}

		for _, m := range metrics {
			if re.MatchString(m) {
				filteredMetrics = append(filteredMetrics, m)
			}
		}
	} else {
		filteredMetrics = metrics
	}

	// Format as a comma-separated list
	output := strings.Join(filteredMetrics, ", ")
	if len(output) == 0 {
		output = "No metrics found."
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output,
			},
		},
	}, nil, nil
}

// QueryMetric queries a single metric
func (t tool) QueryMetric(ctx context.Context, request *mcp.CallToolRequest, params QueryMetricParams) (*mcp.CallToolResult, any, error) {
	// Default to now if time is not provided or invalid
	at := time.Now()
	timeout := "30s"

	result, err := t.client.QueryMetric(ctx, params.Query, at, timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query metric: %w", err)
	}

	series := simplifyMetricResponse(result.Data)
	table := formatMetricsTable(series)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: table,
			},
		},
	}, nil, nil
}

// QueryRangeMetric queries a metric over a range of time
func (t tool) QueryRangeMetric(ctx context.Context, request *mcp.CallToolRequest, params QueryRangeMetricParams) (*mcp.CallToolResult, any, error) {
	start, err := parseTime(params.Start)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse start time: %w", err)
	}

	end, err := parseTime(params.End)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse end time: %w", err)
	}

	step := params.Step
	if step == "" {
		step = "1m"
	}
	timeout := "30s"

	result, err := t.client.QueryRangeMetric(ctx, params.Query, start, end, step, timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query range metric: %w", err)
	}

	series := simplifyMetricResponse(result.Data)
	output := formatMetricsWithChart(series)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output,
			},
		},
	}, nil, nil
}

func formatMetricsWithChart(series []MetricSeries) string {
	if len(series) == 0 {
		return "No data found."
	}

	var sb strings.Builder
	for _, s := range series {
		sb.WriteString(renderAsciiChart(s))
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderAsciiChart(series MetricSeries) string {
	if len(series.Points) < 2 {
		return fmt.Sprintf("Not enough data points to plot for %s", series.Name)
	}

	// Constants for chart dimensions
	const (
		height      = 15
		width       = 80
		yAxisWidth  = 10
		xAxisHeight = 2
		chartWidth  = width - yAxisWidth
		chartHeight = height - xAxisHeight
	)

	points := series.Points
	minVal := points[0].Value
	maxVal := points[0].Value
	minTime := points[0].Timestamp
	maxTime := points[len(points)-1].Timestamp

	for _, p := range points {
		if p.Value < minVal {
			minVal = p.Value
		}
		if p.Value > maxVal {
			maxVal = p.Value
		}
	}

	// Normalize values
	rangeVal := maxVal - minVal
	if rangeVal == 0 {
		rangeVal = 1
	}
	rangeTime := maxTime - minTime
	if rangeTime == 0 {
		rangeTime = 1
	}

	// Initialize grid
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Draw Box Border
	for x := 0; x < width; x++ {
		grid[0][x] = '─'
		grid[height-1][x] = '─'
	}
	for y := 0; y < height; y++ {
		grid[y][0] = '│'
		grid[y][width-1] = '│'
	}
	grid[0][0] = '┌'
	grid[0][width-1] = '┐'
	grid[height-1][0] = '└'
	grid[height-1][width-1] = '┘'

	// Title
	title := " Metrics "
	titleStart := (width - len(title)) / 2
	for i, r := range title {
		grid[0][titleStart+i] = r
	}

	// Y-Axis Line
	for y := 1; y < height-xAxisHeight; y++ {
		grid[y][yAxisWidth] = '│'
	}
	grid[height-xAxisHeight][yAxisWidth] = '└'

	// X-Axis Line
	for x := yAxisWidth + 1; x < width-1; x++ {
		grid[height-xAxisHeight][x] = '─'
	}

	// Plot Points
	for _, p := range points {
		// Calculate X position
		// Time is linear on X axis
		xRatio := float64(p.Timestamp-minTime) / float64(rangeTime)
		x := int(xRatio*float64(chartWidth-2)) + yAxisWidth + 1

		if x >= width-1 {
			x = width - 2
		}

		// Calculate Y position
		// Value is linear on Y axis
		yRatio := (p.Value - minVal) / rangeVal
		y := int(yRatio*float64(chartHeight-2)) + 1

		// Invert Y (0 is top in grid, but max val is top in chart)
		// chartHeight-2 is the available drawing height
		// We want y=1 to be maxVal, y=chartHeight-2 to be minVal
		// So row = (chartHeight - 1) - y
		// Wait, let's be precise.
		// Drawing area Y range: 1 to height-xAxisHeight-1
		// Let's say drawing area is rows 1 to 12 (if height=15, xAxisHeight=2 -> axis at 13)

		drawY := (height - xAxisHeight - 1) - y
		if drawY < 1 {
			drawY = 1
		}

		grid[drawY][x] = '•'
	}

	// Y-Axis Labels
	sMax := fmt.Sprintf("%.2f", maxVal)
	sMin := fmt.Sprintf("%.2f", minVal)

	// Draw Max
	for i, r := range sMax {
		if i < yAxisWidth-1 {
			grid[1][i+1] = r
		}
	}
	// Draw Min
	for i, r := range sMin {
		if i < yAxisWidth-1 {
			grid[height-xAxisHeight-1][i+1] = r
		}
	}

	// Label "Value"
	valLabel := "Value"
	for i, r := range valLabel {
		if i < yAxisWidth-1 {
			grid[2][i+1] = r
		}
	}

	// X-Axis Labels
	tMax := time.Unix(maxTime, 0).Format("15:04:05")
	tMin := time.Unix(minTime, 0).Format("15:04:05")

	// Draw Min Time
	for i, r := range tMin {
		if yAxisWidth+1+i < width-1 {
			grid[height-2][yAxisWidth+1+i] = r
		}
	}

	// Draw Max Time
	for i, r := range tMax {
		if width-2-len(tMax)+i > yAxisWidth {
			grid[height-2][width-2-len(tMax)+i] = r
		}
	}

	// Legend Box
	// Top right corner of chart area
	legend := fmt.Sprintf(" %s ", series.Name)
	if len(legend) > 30 {
		legend = legend[:27] + "..."
	}
	legendX := width - len(legend) - 3
	legendY := 2

	// Draw legend box
	for x := legendX; x < legendX+len(legend)+2; x++ {
		grid[legendY][x] = '─'
		grid[legendY+2][x] = '─'
	}
	for y := legendY; y <= legendY+2; y++ {
		grid[y][legendX] = '│'
		grid[y][legendX+len(legend)+1] = '│'
	}
	grid[legendY][legendX] = '┌'
	grid[legendY][legendX+len(legend)+1] = '┐'
	grid[legendY+2][legendX] = '└'
	grid[legendY+2][legendX+len(legend)+1] = '┘'

	for i, r := range legend {
		grid[legendY+1][legendX+1+i] = r
	}

	// Render to string
	var sb strings.Builder
	for _, row := range grid {
		sb.WriteString(string(row))
		sb.WriteString("\n")
	}

	return sb.String()
}

func parseTime(s string) (time.Time, error) {
	if s == "now" {
		return time.Now(), nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("invalid time format: %s (expected 'now' or duration like '1h')", s)
}

// Helper to simplify metric response
func simplifyMetricResponse(data api.MetricData) []MetricSeries {
	var series []MetricSeries
	for _, res := range data.Result {
		var points []MetricPoint
		for _, p := range res.Points {
			points = append(points, MetricPoint{
				Timestamp: p.Timestamp,
				Value:     p.Value,
			})
		}

		name := res.Labels["__name__"]
		if name == "" {
			name = "metric"
		}

		series = append(series, MetricSeries{
			Name:   name,
			Labels: res.Labels,
			Points: points,
		})
	}
	return series
}

func formatMetricsTable(series []MetricSeries) string {
	if len(series) == 0 {
		return "No data found."
	}

	// Collect all unique label keys
	labelKeysMap := make(map[string]bool)
	for _, s := range series {
		for k := range s.Labels {
			if k != "__name__" { // Exclude __name__ as it is already the Name column
				labelKeysMap[k] = true
			}
		}
	}

	var labelKeys []string
	for k := range labelKeysMap {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)

	var sb strings.Builder

	// Header
	sb.WriteString("| Name | Timestamp | Value |")
	for _, k := range labelKeys {
		sb.WriteString(fmt.Sprintf(" %s |", k))
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString("|---|---|---|")
	for range labelKeys {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")

	// Rows
	for _, s := range series {
		for _, p := range s.Points {
			ts := time.Unix(p.Timestamp, 0).Format(time.RFC3339)
			sb.WriteString(fmt.Sprintf("| %s | %s | %f |", s.Name, ts, p.Value))

			for _, k := range labelKeys {
				val := s.Labels[k]
				sb.WriteString(fmt.Sprintf(" %s |", val))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
