package backtest

import (
	"time"

	"github.com/yourusername/clever-better/internal/models"
)

// BacktestState tracks current backtest state
type BacktestState struct {
	CurrentBankroll float64
	PeakBankroll    float64
	Bets            []*models.Bet
	EquityCurve     EquityCurve
	DailyPnL        map[time.Time]float64
}

// NewBacktestState initializes backtest state
func NewBacktestState(initialBankroll float64) *BacktestState {
	state := &BacktestState{
		CurrentBankroll: initialBankroll,
		PeakBankroll:    initialBankroll,
		Bets:            []*models.Bet{},
		EquityCurve:     EquityCurve{},
		DailyPnL:        make(map[time.Time]float64),
	}
	state.RecordEquityPoint(time.Now().UTC(), initialBankroll)
	return state
}

// UpdateState updates the bankroll and state with a settled bet
func (s *BacktestState) UpdateState(bet *models.Bet, pnl float64) {
	s.CurrentBankroll += pnl
	if s.CurrentBankroll > s.PeakBankroll {
		s.PeakBankroll = s.CurrentBankroll
	}
	s.Bets = append(s.Bets, bet)

	if bet.SettledAt != nil {
		day := time.Date(bet.SettledAt.Year(), bet.SettledAt.Month(), bet.SettledAt.Day(), 0, 0, 0, 0, bet.SettledAt.Location())
		s.DailyPnL[day] += pnl
	}
}

// GetCurrentDrawdown calculates peak-to-trough drawdown
func (s *BacktestState) GetCurrentDrawdown() float64 {
	if s.PeakBankroll == 0 {
		return 0
	}
	drawdown := (s.PeakBankroll - s.CurrentBankroll) / s.PeakBankroll
	if drawdown < 0 {
		return 0
	}
	return drawdown
}

// RecordEquityPoint adds an equity point to the curve
func (s *BacktestState) RecordEquityPoint(t time.Time, value float64) {
	drawdown := 0.0
	if value < s.PeakBankroll && s.PeakBankroll > 0 {
		drawdown = (s.PeakBankroll - value) / s.PeakBankroll
	}

	point := EquityPoint{
		Time:     t,
		Value:    value,
		Drawdown: drawdown,
		DailyPnL: 0,
	}
	s.EquityCurve = append(s.EquityCurve, point)
}
