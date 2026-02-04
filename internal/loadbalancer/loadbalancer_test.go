package loadbalancer

import (
	"testing"
	"time"
)

// mockStore implements a minimal store interface for testing
type mockStore struct {
	accounts       []*mockAccount
	enabledAccounts []*mockAccount
}

type mockAccount struct {
	ID           int64
	Name         string
	Weight       int
	Enabled      bool
	RequestCount int64
}

func TestLoadBalancerStrategies(t *testing.T) {
	t.Run("WeightedRandom", func(t *testing.T) {
		lb := &LoadBalancer{
			strategy:           StrategyWeightedRandom,
			accountHealth:      make(map[int64]*AccountHealth),
			rateLimitCounters:  make(map[int64]*rateLimitCounter),
			rateLimitConfig:    DefaultOptions().RateLimitConfig,
			healthCheckEnabled: false,
		}

		// The selectWeightedRandom function uses internal store.Account, so we test the config
		if lb.strategy != StrategyWeightedRandom {
			t.Errorf("Expected strategy WeightedRandom, got %v", lb.strategy)
		}
	})

	t.Run("SetStrategy", func(t *testing.T) {
		lb := &LoadBalancer{
			strategy: StrategyWeightedRandom,
		}

		lb.SetStrategy(StrategyRoundRobin)
		if lb.GetStrategy() != StrategyRoundRobin {
			t.Errorf("Expected strategy RoundRobin, got %v", lb.GetStrategy())
		}

		lb.SetStrategy(StrategyLeastConnections)
		if lb.GetStrategy() != StrategyLeastConnections {
			t.Errorf("Expected strategy LeastConnections, got %v", lb.GetStrategy())
		}
	})
}

func TestHealthTracking(t *testing.T) {
	lb := &LoadBalancer{
		accountHealth:       make(map[int64]*AccountHealth),
		healthCheckEnabled:  true,
		maxConsecutiveFails: 3,
		failureCooldown:     5 * time.Minute,
	}

	accountID := int64(1)

	// Report success
	lb.ReportSuccess(accountID)
	health := lb.GetAccountHealth(accountID)
	if health == nil {
		t.Fatal("Expected health record to exist")
	}
	if !health.IsHealthy {
		t.Error("Expected account to be healthy after success")
	}
	if health.TotalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", health.TotalRequests)
	}

	// Report failures
	for i := 0; i < 3; i++ {
		lb.ReportFailure(accountID)
	}
	health = lb.GetAccountHealth(accountID)
	if health.IsHealthy {
		t.Error("Expected account to be unhealthy after 3 consecutive failures")
	}
	if health.ConsecutiveFails != 3 {
		t.Errorf("Expected 3 consecutive failures, got %d", health.ConsecutiveFails)
	}

	// Report success should reset consecutive failures
	lb.ReportSuccess(accountID)
	health = lb.GetAccountHealth(accountID)
	if !health.IsHealthy {
		t.Error("Expected account to be healthy after success")
	}
	if health.ConsecutiveFails != 0 {
		t.Errorf("Expected 0 consecutive failures after success, got %d", health.ConsecutiveFails)
	}
}

func TestGetAllAccountHealth(t *testing.T) {
	lb := &LoadBalancer{
		accountHealth: make(map[int64]*AccountHealth),
	}

	// Add some health records
	lb.ReportSuccess(1)
	lb.ReportSuccess(2)
	lb.ReportFailure(3)

	allHealth := lb.GetAllAccountHealth()
	if len(allHealth) != 3 {
		t.Errorf("Expected 3 health records, got %d", len(allHealth))
	}
}

func TestGetStats(t *testing.T) {
	lb := &LoadBalancer{
		strategy:           StrategyRoundRobin,
		accountHealth:      make(map[int64]*AccountHealth),
		healthCheckEnabled: true,
	}

	// Add some health records
	lb.accountHealth[1] = &AccountHealth{ID: 1, IsHealthy: true}
	lb.accountHealth[2] = &AccountHealth{ID: 2, IsHealthy: true}
	lb.accountHealth[3] = &AccountHealth{ID: 3, IsHealthy: false}

	stats := lb.GetStats()
	
	if stats["strategy"] != "round_robin" {
		t.Errorf("Expected strategy 'round_robin', got %v", stats["strategy"])
	}
	if stats["healthy_count"] != 2 {
		t.Errorf("Expected healthy_count 2, got %v", stats["healthy_count"])
	}
	if stats["unhealthy_count"] != 1 {
		t.Errorf("Expected unhealthy_count 1, got %v", stats["unhealthy_count"])
	}
	if stats["health_enabled"] != true {
		t.Errorf("Expected health_enabled true, got %v", stats["health_enabled"])
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Strategy != StrategyWeightedRandom {
		t.Errorf("Expected default strategy WeightedRandom, got %v", opts.Strategy)
	}
	if opts.MaxConsecutiveFails != 3 {
		t.Errorf("Expected max consecutive fails 3, got %d", opts.MaxConsecutiveFails)
	}
	if opts.FailureCooldown != 5*time.Minute {
		t.Errorf("Expected failure cooldown 5m, got %v", opts.FailureCooldown)
	}
	if opts.RateLimitConfig.RequestsPerMinute != 60 {
		t.Errorf("Expected 60 requests per minute, got %d", opts.RateLimitConfig.RequestsPerMinute)
	}
}

func TestRateLimitCounter(t *testing.T) {
	lb := &LoadBalancer{
		rateLimitCounters: make(map[int64]*rateLimitCounter),
		rateLimitConfig: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstSize:         10,
		},
	}

	accountID := int64(1)

	// First increment creates counter
	lb.incrementRateLimit(accountID)
	counter := lb.rateLimitCounters[accountID]
	if counter == nil {
		t.Fatal("Expected counter to be created")
	}
	if counter.minuteCount != 1 {
		t.Errorf("Expected minute count 1, got %d", counter.minuteCount)
	}
	if counter.hourCount != 1 {
		t.Errorf("Expected hour count 1, got %d", counter.hourCount)
	}

	// Additional increments
	lb.incrementRateLimit(accountID)
	lb.incrementRateLimit(accountID)
	if counter.minuteCount != 3 {
		t.Errorf("Expected minute count 3, got %d", counter.minuteCount)
	}
}
