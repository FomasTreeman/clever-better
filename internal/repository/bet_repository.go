package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/models"
)

// PostgresBetRepository implements BetRepository for PostgreSQL
type PostgresBetRepository struct {
	db *database.DB
}

// NewPostgresBetRepository creates a new bet repository
func NewPostgresBetRepository(db *database.DB) BetRepository {
	return &PostgresBetRepository{db: db}
}

// Create inserts a new bet
func (b *PostgresBetRepository) Create(ctx context.Context, bet *models.Bet) error {
	query := `
		INSERT INTO bets (id, bet_id, market_id, race_id, runner_id, strategy_id, market_type, side, 
		                  odds, stake, matched_price, matched_size, status, placed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := b.db.GetPool().Exec(ctx, query,
		bet.ID, bet.BetID, bet.MarketID, bet.RaceID, bet.RunnerID, bet.StrategyID, bet.MarketType,
		bet.Side, bet.Odds, bet.Stake, bet.MatchedPrice, bet.MatchedSize, bet.Status, bet.PlacedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create bet: %w", err)
	}

	return nil
}

// GetByID retrieves a bet by ID
func (b *PostgresBetRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Bet, error) {
	query := `
		SELECT id, bet_id, market_id, race_id, runner_id, strategy_id, market_type, side, odds, stake,
		       matched_price, matched_size, status, placed_at, matched_at, settled_at, cancelled_at,
		       profit_loss, commission, created_at, updated_at
		FROM bets WHERE id = $1
	`

	bet := &models.Bet{}
	err := b.db.GetPool().QueryRow(ctx, query, id).Scan(
		&bet.ID, &bet.BetID, &bet.MarketID, &bet.RaceID, &bet.RunnerID, &bet.StrategyID, &bet.MarketType,
		&bet.Side, &bet.Odds, &bet.Stake, &bet.MatchedPrice, &bet.MatchedSize, &bet.Status, &bet.PlacedAt,
		&bet.MatchedAt, &bet.SettledAt, &bet.CancelledAt, &bet.ProfitLoss, &bet.Commission, &bet.CreatedAt, &bet.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bet: %w", err)
	}

	return bet, nil
}

// GetByRaceID retrieves all bets for a specific race
func (b *PostgresBetRepository) GetByRaceID(ctx context.Context, raceID uuid.UUID) ([]*models.Bet, error) {
	query := `
		SELECT id, bet_id, market_id, race_id, runner_id, strategy_id, market_type, side, odds, stake,
		       matched_price, matched_size, status, placed_at, matched_at, settled_at, cancelled_at,
		       profit_loss, commission, created_at, updated_at
		FROM bets
		WHERE race_id = $1
		ORDER BY placed_at DESC
	`

	rows, err := b.db.GetPool().Query(ctx, query, raceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query bets by race: %w", err)
	}
	defer rows.Close()

	var bets []*models.Bet
	for rows.Next() {
		bet := &models.Bet{}
		err := rows.Scan(
			&bet.ID, &bet.BetID, &bet.MarketID, &bet.RaceID, &bet.RunnerID, &bet.StrategyID, &bet.MarketType,
			&bet.Side, &bet.Odds, &bet.Stake, &bet.MatchedPrice, &bet.MatchedSize, &bet.Status, &bet.PlacedAt,
			&bet.MatchedAt, &bet.SettledAt, &bet.CancelledAt, &bet.ProfitLoss, &bet.Commission, &bet.CreatedAt, &bet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bet: %w", err)
		}
		bets = append(bets, bet)
	}

	return bets, rows.Err()
}

// GetByStrategyID retrieves all bets for a specific strategy within a date range
func (b *PostgresBetRepository) GetByStrategyID(ctx context.Context, strategyID uuid.UUID, start, end time.Time) ([]*models.Bet, error) {
	query := `
		SELECT id, bet_id, market_id, race_id, runner_id, strategy_id, market_type, side, odds, stake,
		       matched_price, matched_size, status, placed_at, matched_at, settled_at, cancelled_at,
		       profit_loss, commission, created_at, updated_at
		FROM bets
		WHERE strategy_id = $1 AND placed_at >= $2 AND placed_at <= $3
		ORDER BY placed_at DESC
	`

	rows, err := b.db.GetPool().Query(ctx, query, strategyID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query bets by strategy: %w", err)
	}
	defer rows.Close()

	var bets []*models.Bet
	for rows.Next() {
		bet := &models.Bet{}
		err := rows.Scan(
			&bet.ID, &bet.BetID, &bet.MarketID, &bet.RaceID, &bet.RunnerID, &bet.StrategyID, &bet.MarketType,
			&bet.Side, &bet.Odds, &bet.Stake, &bet.MatchedPrice, &bet.MatchedSize, &bet.Status, &bet.PlacedAt,
			&bet.MatchedAt, &bet.SettledAt, &bet.CancelledAt, &bet.ProfitLoss, &bet.Commission, &bet.CreatedAt, &bet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bet: %w", err)
		}
		bets = append(bets, bet)
	}

	return bets, rows.Err()
}

// Update updates an existing bet
func (b *PostgresBetRepository) Update(ctx context.Context, bet *models.Bet) error {
	query := `
		UPDATE bets SET
			bet_id = $2, market_id = $3, matched_price = $4, matched_size = $5,
			status = $6, matched_at = $7, settled_at = $8, cancelled_at = $9,
			profit_loss = $10, commission = $11, updated_at = NOW()
		WHERE id = $1
	`

	commandTag, err := b.db.GetPool().Exec(ctx, query,
		bet.ID, bet.BetID, bet.MarketID, bet.MatchedPrice, bet.MatchedSize,
		bet.Status, bet.MatchedAt, bet.SettledAt, bet.CancelledAt, bet.ProfitLoss, bet.Commission,
	)
	if err != nil {
		return fmt.Errorf("failed to update bet: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrNotFound
	}

	return nil
}

// GetPendingBets retrieves all pending bets
func (b *PostgresBetRepository) GetPendingBets(ctx context.Context) ([]*models.Bet, error) {
	query := `
		SELECT id, bet_id, market_id, race_id, runner_id, strategy_id, market_type, side, odds, stake,
		       matched_price, matched_size, status, placed_at, matched_at, settled_at, cancelled_at,
		       profit_loss, commission, created_at, updated_at
		FROM bets
		WHERE status = 'pending'
		ORDER BY placed_at ASC
	`

	rows, err := b.db.GetPool().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending bets: %w", err)
	}
	defer rows.Close()

	var bets []*models.Bet
	for rows.Next() {
		bet := &models.Bet{}
		err := rows.Scan(
			&bet.ID, &bet.BetID, &bet.MarketID, &bet.RaceID, &bet.RunnerID, &bet.StrategyID, &bet.MarketType,
			&bet.Side, &bet.Odds, &bet.Stake, &bet.MatchedPrice, &bet.MatchedSize, &bet.Status, &bet.PlacedAt,
			&bet.MatchedAt, &bet.SettledAt, &bet.CancelledAt, &bet.ProfitLoss, &bet.Commission, &bet.CreatedAt, &bet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bet: %w", err)
		}
		bets = append(bets, bet)
	}

	return bets, rows.Err()
}

// GetSettledBets retrieves all settled bets within a date range
func (b *PostgresBetRepository) GetSettledBets(ctx context.Context, start, end time.Time) ([]*models.Bet, error) {
	query := `
		SELECT id, bet_id, market_id, race_id, runner_id, strategy_id, market_type, side, odds, stake,
		       matched_price, matched_size, status, placed_at, matched_at, settled_at, cancelled_at,
		       profit_loss, commission, created_at, updated_at
		FROM bets
		WHERE status = 'settled' AND settled_at >= $1 AND settled_at <= $2
		ORDER BY settled_at DESC
	`

	rows, err := b.db.GetPool().Query(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query settled bets: %w", err)
	}
	defer rows.Close()

	var bets []*models.Bet
	for rows.Next() {
		bet := &models.Bet{}
		err := rows.Scan(
			&bet.ID, &bet.BetID, &bet.MarketID, &bet.RaceID, &bet.RunnerID, &bet.StrategyID, &bet.MarketType,
			&bet.Side, &bet.Odds, &bet.Stake, &bet.MatchedPrice, &bet.MatchedSize, &bet.Status, &bet.PlacedAt,
			&bet.MatchedAt, &bet.SettledAt, &bet.CancelledAt, &bet.ProfitLoss, &bet.Commission, &bet.CreatedAt, &bet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bet: %w", err)
		}
		bets = append(bets, bet)
	}

	return bets, rows.Err()
}

// GetByBetfairBetID retrieves a bet by Betfair bet ID
func (b *PostgresBetRepository) GetByBetfairBetID(ctx context.Context, betID string) (*models.Bet, error) {
	query := `
		SELECT id, bet_id, market_id, race_id, runner_id, strategy_id, market_type, side, odds, stake,
		       matched_price, matched_size, status, placed_at, matched_at, settled_at, cancelled_at,
		       profit_loss, commission, created_at, updated_at
		FROM bets WHERE bet_id = $1
	`

	bet := &models.Bet{}
	err := b.db.GetPool().QueryRow(ctx, query, betID).Scan(
		&bet.ID, &bet.BetID, &bet.MarketID, &bet.RaceID, &bet.RunnerID, &bet.StrategyID, &bet.MarketType,
		&bet.Side, &bet.Odds, &bet.Stake, &bet.MatchedPrice, &bet.MatchedSize, &bet.Status, &bet.PlacedAt,
		&bet.MatchedAt, &bet.SettledAt, &bet.CancelledAt, &bet.ProfitLoss, &bet.Commission, &bet.CreatedAt, &bet.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bet by betfair bet ID: %w", err)
	}

	return bet, nil
}
