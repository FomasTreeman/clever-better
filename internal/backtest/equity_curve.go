package backtest

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"time"
)

// EquityPoint represents a point in the equity curve
type EquityPoint struct {
	Time     time.Time `json:"time"`
	Value    float64   `json:"value"`
	Drawdown float64   `json:"drawdown"`
	DailyPnL float64   `json:"daily_pnl"`
}

// EquityCurve represents a time-series of equity points
type EquityCurve []EquityPoint

// GetReturns calculates periodic returns from equity curve
func (e EquityCurve) GetReturns() []float64 {
	if len(e) < 2 {
		return []float64{}
	}
	returns := make([]float64, 0, len(e)-1)
	for i := 1; i < len(e); i++ {
		prev := e[i-1].Value
		curr := e[i].Value
		if prev == 0 {
			returns = append(returns, 0)
			continue
		}
		returns = append(returns, (curr-prev)/prev)
	}
	return returns
}

// GetVolatility calculates standard deviation of returns
func (e EquityCurve) GetVolatility() float64 {
	returns := e.GetReturns()
	if len(returns) == 0 {
		return 0
	}
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))
	return math.Sqrt(variance)
}

// GetDownsideDeviation calculates downside deviation of returns
func (e EquityCurve) GetDownsideDeviation() float64 {
	returns := e.GetReturns()
	if len(returns) == 0 {
		return 0
	}
	variance := 0.0
	count := 0
	for _, r := range returns {
		if r < 0 {
			variance += r * r
			count++
		}
	}
	if count == 0 {
		return 0
	}
	variance /= float64(count)
	return math.Sqrt(variance)
}

// ToCSV exports equity curve to CSV string
func (e EquityCurve) ToCSV() string {
	var buf bytes.Buffer
	buf.WriteString("time,value,drawdown,daily_pnl\n")
	for _, point := range e {
		buf.WriteString(point.Time.Format(time.RFC3339))
		buf.WriteString(",")
		buf.WriteString(formatFloat(point.Value))
		buf.WriteString(",")
		buf.WriteString(formatFloat(point.Drawdown))
		buf.WriteString(",")
		buf.WriteString(formatFloat(point.DailyPnL))
		buf.WriteString("\n")
	}
	return buf.String()
}

// ToJSON exports equity curve to JSON string
func (e EquityCurve) ToJSON() string {
	data, _ := json.Marshal(e)
	return string(data)
}

func formatFloat(v float64) string {
	return strconvFormat(v, 6)
}

func strconvFormat(v float64, prec int) string {
	return strconv.FormatFloat(v, 'f', prec, 64)
}
