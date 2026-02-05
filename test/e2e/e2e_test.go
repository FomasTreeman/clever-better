//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/clever-better/internal/backtest"
	"github.com/yourusername/clever-better/internal/betfair"
	"github.com/yourusername/clever-better/internal/bot"
	"github.com/yourusername/clever-better/internal/config"
	"github.com/yourusername/clever-better/internal/database"
	"github.com/yourusername/clever-better/internal/datasource"
	"github.com/yourusername/clever-better/internal/ml"
	mlpb "github.com/yourusername/clever-better/internal/ml/mlpb"
	"github.com/yourusername/clever-better/internal/models"
	"github.com/yourusername/clever-better/internal/repository"
	"github.com/yourusername/clever-better/internal/strategy"
	"google.golang.org/grpc"
)

const (
	sampleMarketID = "1.198765432"
	skipE2E        = "Skipping E2E test in short mode"
)

type raceFixture struct {
	ID          string         `json:"id"`
	MarketID    string         `json:"market_id"`
	Venue       string         `json:"venue"`
	RaceName    string         `json:"race_name"`
	RaceType    string         `json:"race_type"`
	Distance    int            `json:"distance"`
	RaceClass   string         `json:"race_class"`
	StartTime   string         `json:"start_time"`
	Runners     []runnerFixture `json:"runners"`
}

type runnerFixture struct {
	ID          string `json:"id"`
	SelectionID uint64 `json:"selection_id"`
	Name        string `json:"name"`
	Trainer     string `json:"trainer"`
	Age         int    `json:"age"`
	Weight      int    `json:"weight"`
	Stall       int    `json:"stall"`
	Form        string `json:"form"`
}

type oddsFixture struct {
	MarketID string            `json:"market_id"`
	Timestamp string           `json:"timestamp"`
	Runners  []oddsRunnerFixture `json:"runners"`
}

type oddsRunnerFixture struct {
	SelectionID uint64       `json:"selection_id"`
	BackPrices  []priceSize  `json:"back_prices"`
	LayPrices   []priceSize  `json:"lay_prices"`
	LTP         float64      `json:"last_price_traded"`
	TotalMatched float64     `json:"total_matched"`
}

type priceSize struct {
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

type e2eStrategy struct{}

func (e e2eStrategy) Name() string { return "e2e" }
func (e e2eStrategy) Evaluate(ctx context.Context, strategyCtx strategy.Context) ([]strategy.Signal, error) {
	if len(strategyCtx.Runners) == 0 {
		return nil, nil
	}
	runner := strategyCtx.Runners[0]
	return []strategy.Signal{{
		RunnerID:      runner.ID,
		Side:          models.BetSideBack,
		Odds:          3.5,
		Stake:         50,
		Confidence:    0.7,
		ExpectedValue: 0.1,
		Reasoning:     "e2e",
	}}, nil
}
func (e e2eStrategy) ShouldBet(signal strategy.Signal) bool { return true }
func (e e2eStrategy) CalculateStake(signal strategy.Signal, bankroll float64) float64 {
	if bankroll < signal.Stake {
		return bankroll
	}
	return signal.Stake
}
func (e e2eStrategy) GetParameters() map[string]interface{} { return map[string]interface{}{} }

type mockMLService struct {
	mlpb.UnimplementedMLServiceServer
}

func (m *mockMLService) GetPrediction(ctx context.Context, req *mlpb.PredictionRequest) (*mlpb.PredictionResponse, error) {
	return &mlpb.PredictionResponse{
		RaceId:               req.RaceId,
		RunnerId:             req.RunnerId,
		PredictedProbability: 0.65,
		Confidence:           0.75,
		Recommendation:       "BACK",
		ModelVersion:         "v1",
	}, nil
}

func (m *mockMLService) EvaluateStrategy(ctx context.Context, req *mlpb.StrategyRequest) (*mlpb.StrategyResponse, error) {
	return &mlpb.StrategyResponse{CompositeScore: 0.8, Recommendation: "INCREASE"}, nil
}

func (m *mockMLService) SubmitBacktestFeedback(ctx context.Context, req *mlpb.BacktestFeedbackRequest) (*mlpb.BacktestFeedbackResponse, error) {
	return &mlpb.BacktestFeedbackResponse{Success: true, Message: "ok"}, nil
}

func (m *mockMLService) GenerateStrategy(ctx context.Context, req *mlpb.StrategyGenerationRequest) (*mlpb.StrategyGenerationResponse, error) {
	return &mlpb.StrategyGenerationResponse{}, nil
}

func (m *mockMLService) BatchPredict(ctx context.Context, req *mlpb.BatchPredictionRequest) (*mlpb.BatchPredictionResponse, error) {
	return &mlpb.BatchPredictionResponse{}, nil
}

func startMockMLServer(t *testing.T) (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	mlpb.RegisterMLServiceServer(grpcServer, &mockMLService{})

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	cleanup := func() {
		grpcServer.Stop()
		_ = listener.Close()
	}

	return listener.Addr().String(), cleanup
}

func setupBetfairServer(t *testing.T) (*betfair.BetfairClient, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req betfair.JSONRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")

		switch req.Method {
		case "placeOrders":
			resp := betfair.PlaceOrdersResponse{
				MarketID: sampleMarketID,
				Status:   "SUCCESS",
				InstructionReports: []betfair.InstructionReport{{
					Status:              "SUCCESS",
					OrderStatus:         "EXECUTION_COMPLETE",
					BetID:               "bf-bet-123",
					AveragePriceMatched: 3.5,
					SizeMatched:         100,
				}},
			}

			payload, err := json.Marshal(resp)
			require.NoError(t, err)
			json.NewEncoder(w).Encode(betfair.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: payload})
			return
		case "listMarketCatalogue":
			payload, err := json.Marshal([]map[string]interface{}{{"marketId": sampleMarketID}})
			require.NoError(t, err)
			json.NewEncoder(w).Encode(betfair.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: payload})
			return
		default:
			json.NewEncoder(w).Encode(betfair.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &betfair.JSONRPCError{Code: 400, Message: "UNKNOWN"}})
		}
	}))

	httpClient := datasource.NewRateLimitedHTTPClient(datasource.HTTPClientConfig{
		Timeout:           2 * time.Second,
		MaxRetries:        0,
		RetryWaitMin:      10 * time.Millisecond,
		RetryWaitMax:      20 * time.Millisecond,
		RateLimit:         100,
		CircuitBreakerMax: 10,
	}, nil)

	cfg := &config.BetfairConfig{
		APIURL:    server.URL,
		StreamURL: server.URL,
		AppKey:    "test-key",
		Username:  "test-user",
		Password:  "test-pass",
	}

	client := betfair.NewBetfairClient(cfg, httpClient, nil)
	client.SetSessionToken("test-session", time.Now().Add(1*time.Hour))

	cleanup := func() {
		server.Close()
	}

	return client, cleanup
}

func loadRaceFixtures(t *testing.T) []raceFixture {
	data, err := os.ReadFile("test/e2e/fixtures/sample_race_data.json")
	require.NoError(t, err)

	var races []raceFixture
	err = json.Unmarshal(data, &races)
	require.NoError(t, err)
	return races
}

func loadOddsFixtures(t *testing.T) []oddsFixture {
	data, err := os.ReadFile("test/e2e/fixtures/sample_odds_data.json")
	require.NoError(t, err)

	var odds []oddsFixture
	err = json.Unmarshal(data, &odds)
	require.NoError(t, err)
	return odds
}

func seedRacesAndRunners(t *testing.T, ctx context.Context, repos *repository.Repositories, fixtures []raceFixture) (map[string]uuid.UUID, map[uint64]uuid.UUID) {
	marketToRace := make(map[string]uuid.UUID)
	selectionToRunner := make(map[uint64]uuid.UUID)

	for _, fixture := range fixtures {
		raceID := uuid.MustParse(fixture.ID)
		scheduled, err := time.Parse(time.RFC3339, fixture.StartTime)
		require.NoError(t, err)

			race := &models.Race{
			ID:             raceID,
			ScheduledStart: scheduled,
			Track:          fixture.Venue,
			RaceType:       fixture.RaceType,
			Distance:       fixture.Distance,
			Grade:          fixture.RaceClass,
				Conditions:     json.RawMessage(`{"race_name":"` + fixture.RaceName + `"}`),
			Status:         "scheduled",
		}
		require.NoError(t, repos.Race.Create(ctx, race))
		marketToRace[fixture.MarketID] = raceID

		for _, runnerFixture := range fixture.Runners {
			runnerID := uuid.MustParse(runnerFixture.ID)
			weight := float64(runnerFixture.Weight)
			runner := &models.Runner{
				ID:         runnerID,
				RaceID:     raceID,
				TrapNumber: runnerFixture.Stall,
				Name:       runnerFixture.Name,
				Trainer:    runnerFixture.Trainer,
				Weight:     &weight,
				Metadata:   json.RawMessage(fmt.Sprintf(`{"selection_id":%d,"form":"%s"}`, runnerFixture.SelectionID, runnerFixture.Form)),
			}
			require.NoError(t, repos.Runner.Create(ctx, runner))
			selectionToRunner[runnerFixture.SelectionID] = runnerID
		}
	}

	return marketToRace, selectionToRunner
}

func seedOddsSnapshots(t *testing.T, ctx context.Context, repos *repository.Repositories, fixtures []oddsFixture, marketToRace map[string]uuid.UUID, selectionToRunner map[uint64]uuid.UUID) {
	var snapshots []*models.OddsSnapshot

	for _, fixture := range fixtures {
		raceID, ok := marketToRace[fixture.MarketID]
		require.True(t, ok, "race not found for market %s", fixture.MarketID)

		at, err := time.Parse(time.RFC3339, fixture.Timestamp)
		require.NoError(t, err)

		for _, runner := range fixture.Runners {
			runnerID, ok := selectionToRunner[runner.SelectionID]
			require.True(t, ok, "runner not found for selection %d", runner.SelectionID)

			var backPrice, backSize, layPrice, laySize *float64
			if len(runner.BackPrices) > 0 {
				backPrice = &runner.BackPrices[0].Price
				backSize = &runner.BackPrices[0].Size
			}
			if len(runner.LayPrices) > 0 {
				layPrice = &runner.LayPrices[0].Price
				laySize = &runner.LayPrices[0].Size
			}
			ltp := runner.LTP
			total := runner.TotalMatched

			snapshots = append(snapshots, &models.OddsSnapshot{
				Time:        at,
				RaceID:      raceID,
				RunnerID:    runnerID,
				BackPrice:   backPrice,
				BackSize:    backSize,
				LayPrice:    layPrice,
				LaySize:     laySize,
				LTP:         &ltp,
				TotalVolume: &total,
			})
		}
	}

	require.NoError(t, repos.Odds.InsertBatch(ctx, snapshots))
}

// TestCompleteWorkflow validates end-to-end workflow across backtest, ML, Betfair, and persistence
func TestCompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip(skipE2E)
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Setup database
	db := database.SetupTestDB(t)
	defer database.TeardownTestDB(t, db)

	repos, err := repository.NewRepositories(db)
	require.NoError(t, err)

	// Seed fixtures
	raceFixtures := loadRaceFixtures(t)
	oddsFixtures := loadOddsFixtures(t)

	marketToRace, selectionToRunner := seedRacesAndRunners(t, ctx, repos, raceFixtures)
	seedOddsSnapshots(t, ctx, repos, oddsFixtures, marketToRace, selectionToRunner)

	// Start mock ML service
	grpcAddr, mlCleanup := startMockMLServer(t)
	defer mlCleanup()

	mlClient, err := ml.NewMLClient(&config.MLServiceConfig{URL: "http://localhost", GRPCAddress: grpcAddr}, logger)
	require.NoError(t, err)

	// Start mock Betfair server
	betfairClient, betfairCleanup := setupBetfairServer(t)
	defer betfairCleanup()

	bettingService := betfair.NewBettingService(betfairClient, repos.Bet, betfair.BettingConfig{
		MaxStake:       500.0,
		MinStake:       10.0,
		CommissionRate: 0.05,
	}, nil)

	// Run backtest
	startDate := time.Now().Add(-24 * time.Hour)
	endDate := time.Now().Add(24 * time.Hour)

	engine, err := backtest.NewEngine(backtest.BacktestConfig{
		StartDate:       startDate,
		EndDate:         endDate,
		InitialBankroll: 10000,
		CommissionRate:  0.05,
		SlippageTicks:   1,
		MinLiquidity:    0,
		OutputPath:      "",
		MonteCarloIterations: 10,
	}, db, e2eStrategy{}, logger)
	require.NoError(t, err)

	state, metrics, err := engine.Run(ctx, startDate, endDate)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Greater(t, len(state.Bets), 0)

	// Persist backtest result
	strategyID := uuid.New()
	result := &models.BacktestResult{
		ID:             uuid.New(),
		StrategyID:     strategyID,
		RunDate:        time.Now(),
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: 10000,
		FinalCapital:   metrics.FinalBankroll,
		TotalReturn:    metrics.TotalReturn,
		SharpeRatio:    metrics.SharpeRatio,
		MaxDrawdown:    metrics.MaxDrawdown,
		TotalBets:      len(state.Bets),
		WinRate:        metrics.WinRate,
		ProfitFactor:   metrics.ProfitFactor,
		Method:         "e2e",
		CompositeScore: metrics.CompositeScore,
		Recommendation: "HOLD",
		MLFeatures:     json.RawMessage(`{"source":"e2e"}`),
		FullResults:    json.RawMessage(`{}`),
		CreatedAt:      time.Now(),
	}
	require.NoError(t, repos.BacktestResult.SaveResult(ctx, result))

	// Request ML prediction
	var sampleRunnerID uuid.UUID
	for _, id := range selectionToRunner {
		sampleRunnerID = id
		break
	}

	prediction, err := mlClient.GetPrediction(ctx, uuid.New(), sampleRunnerID, strategyID, []float64{1, 2}, "latest")
	require.NoError(t, err)
	assert.Equal(t, "BACK", prediction.Recommendation)

	// Place simulated bet via Betfair
	betID, err := bettingService.PlaceBet(ctx, sampleMarketID, 12345678, 3.5, 100.0, "BACK")
	require.NoError(t, err)
	assert.Equal(t, "bf-bet-123", betID)

	// Persist bet
	bet := &models.Bet{
		ID:         uuid.New(),
		BetID:      betID,
		MarketID:   sampleMarketID,
		RaceID:     marketToRace[sampleMarketID],
		RunnerID:   sampleRunnerID,
		StrategyID: strategyID,
		MarketType: models.MarketTypeWin,
		Side:       models.BetSideBack,
		Odds:       3.5,
		Stake:      100,
		Status:     models.BetStatusMatched,
		PlacedAt:   time.Now(),
	}
	require.NoError(t, repos.Bet.Create(ctx, bet))

	bets, err := repos.Bet.GetByRaceID(ctx, bet.RaceID)
	require.NoError(t, err)
	require.NotEmpty(t, bets)

	// Enforce risk limits
	riskMgr := bot.NewRiskManager(&config.TradingConfig{
		MaxStakePerBet: 100,
		MaxExposure:    200,
		MaxDailyLoss:   50,
	}, repos.Bet, logger)

	assert.Error(t, riskMgr.CheckRiskLimits(ctx, 150), "stake exceeds max stake")

	// Strategy activation/deactivation
	params, err := json.Marshal(map[string]interface{}{"min_value": 0.1})
	require.NoError(t, err)

	strategyRecord := &models.Strategy{
		ID:          strategyID,
		Name:        "E2E Strategy",
		Description: "e2e validation",
		Parameters:  params,
		Active:      false,
	}
	require.NoError(t, repos.Strategy.Create(ctx, strategyRecord))

	strategyRecord.Active = true
	require.NoError(t, repos.Strategy.Update(ctx, strategyRecord))

	updated, err := repos.Strategy.GetByID(ctx, strategyID)
	require.NoError(t, err)
	assert.True(t, updated.Active)

	strategyRecord.Active = false
	require.NoError(t, repos.Strategy.Update(ctx, strategyRecord))
}
