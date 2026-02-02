package backtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateConsoleReport formats metrics for terminal output
func GenerateConsoleReport(result AggregatedResult) string {
	var builder strings.Builder
	builder.WriteString("Backtest Report\n")
	builder.WriteString("================\n")
	builder.WriteString(fmt.Sprintf("Composite Score: %.2f\n", result.CompositeScore))
	builder.WriteString(fmt.Sprintf("Recommendation: %s\n", result.Recommendation))
	builder.WriteString(fmt.Sprintf("Total Return: %.2f%%\n", result.HistoricalReplayMetrics.TotalReturn*100))
	builder.WriteString(fmt.Sprintf("Sharpe Ratio: %.2f\n", result.HistoricalReplayMetrics.SharpeRatio))
	builder.WriteString(fmt.Sprintf("Max Drawdown: %.2f%%\n", result.HistoricalReplayMetrics.MaxDrawdown*100))
	builder.WriteString(fmt.Sprintf("Win Rate: %.2f%%\n", result.HistoricalReplayMetrics.WinRate*100))
	builder.WriteString(fmt.Sprintf("Profit Factor: %.2f\n", result.HistoricalReplayMetrics.ProfitFactor))
	return builder.String()
}

// GenerateHTMLReport creates a simple HTML report
func GenerateHTMLReport(result AggregatedResult, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Backtest Report</title></head>
<body>
<h1>Backtest Report</h1>
<p><strong>Composite Score:</strong> %.2f</p>
<p><strong>Recommendation:</strong> %s</p>
<p><strong>Total Return:</strong> %.2f%%</p>
<p><strong>Sharpe Ratio:</strong> %.2f</p>
<p><strong>Max Drawdown:</strong> %.2f%%</p>
<p><strong>Win Rate:</strong> %.2f%%</p>
<p><strong>Profit Factor:</strong> %.2f</p>
</body>
</html>`,
		result.CompositeScore,
		result.Recommendation,
		result.HistoricalReplayMetrics.TotalReturn*100,
		result.HistoricalReplayMetrics.SharpeRatio,
		result.HistoricalReplayMetrics.MaxDrawdown*100,
		result.HistoricalReplayMetrics.WinRate*100,
		result.HistoricalReplayMetrics.ProfitFactor,
	)

	return os.WriteFile(outputPath, []byte(html), 0o644)
}

// GenerateCSVExport exports key metrics for spreadsheets
func GenerateCSVExport(result AggregatedResult, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	csv := "metric,value\n" +
		fmt.Sprintf("composite_score,%.4f\n", result.CompositeScore) +
		fmt.Sprintf("total_return,%.4f\n", result.HistoricalReplayMetrics.TotalReturn) +
		fmt.Sprintf("sharpe_ratio,%.4f\n", result.HistoricalReplayMetrics.SharpeRatio) +
		fmt.Sprintf("max_drawdown,%.4f\n", result.HistoricalReplayMetrics.MaxDrawdown) +
		fmt.Sprintf("win_rate,%.4f\n", result.HistoricalReplayMetrics.WinRate) +
		fmt.Sprintf("profit_factor,%.4f\n", result.HistoricalReplayMetrics.ProfitFactor) +
		fmt.Sprintf("recommendation,%s\n", result.Recommendation)
	return os.WriteFile(outputPath, []byte(csv), 0o644)
}
