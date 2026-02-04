package loadbalancer

import (
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"orchids-api/internal/store"
)

// Strategy defines the load balancing strategy
type Strategy string

const (
	// StrategyWeightedRandom uses weighted random selection (default)
	StrategyWeightedRandom Strategy = "weighted_random"
	// StrategyRoundRobin distributes requests evenly across accounts
	StrategyRoundRobin Strategy = "round_robin"
	// StrategyLeastConnections selects the account with fewest active requests
	StrategyLeastConnections Strategy = "least_connections"
)

// AccountHealth tracks the health status of an account
type AccountHealth struct {
	ID              int64
	FailureCount    int
	LastFailure     time.Time
	LastSuccess     time.Time
	IsHealthy       bool
	ConsecutiveFails int
	TotalRequests   int64
	TotalFailures   int64
}

// RateLimitConfig defines rate limiting settings per account
type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
	BurstSize         int
}

// LoadBalancer manages account selection with various strategies
type LoadBalancer struct {
	store              *store.Store
	mu                 sync.RWMutex
	strategy           Strategy
	roundRobinIndex    int
	accountHealth      map[int64]*AccountHealth
	rateLimitCounters  map[int64]*rateLimitCounter
	rateLimitConfig    RateLimitConfig
	healthCheckEnabled bool
	maxConsecutiveFails int
	failureCooldown    time.Duration
}

// rateLimitCounter tracks request counts for rate limiting
type rateLimitCounter struct {
	minuteCount   int
	hourCount     int
	minuteStart   time.Time
	hourStart     time.Time
	mu            sync.Mutex
}

// Options configures the load balancer
type Options struct {
	Strategy            Strategy
	RateLimitConfig     RateLimitConfig
	HealthCheckEnabled  bool
	MaxConsecutiveFails int
	FailureCooldown     time.Duration
}

// DefaultOptions returns default load balancer options
func DefaultOptions() Options {
	return Options{
		Strategy: StrategyWeightedRandom,
		RateLimitConfig: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstSize:         10,
		},
		HealthCheckEnabled:  true,
		MaxConsecutiveFails: 3,
		FailureCooldown:     5 * time.Minute,
	}
}

// New creates a new LoadBalancer with default options
func New(s *store.Store) *LoadBalancer {
	return NewWithOptions(s, DefaultOptions())
}

// NewWithOptions creates a new LoadBalancer with custom options
func NewWithOptions(s *store.Store, opts Options) *LoadBalancer {
	return &LoadBalancer{
		store:               s,
		strategy:            opts.Strategy,
		roundRobinIndex:     0,
		accountHealth:       make(map[int64]*AccountHealth),
		rateLimitCounters:   make(map[int64]*rateLimitCounter),
		rateLimitConfig:     opts.RateLimitConfig,
		healthCheckEnabled:  opts.HealthCheckEnabled,
		maxConsecutiveFails: opts.MaxConsecutiveFails,
		failureCooldown:     opts.FailureCooldown,
	}
}

func (lb *LoadBalancer) GetNextAccount() (*store.Account, error) {
	return lb.GetNextAccountExcluding(nil)
}

func (lb *LoadBalancer) GetNextAccountExcluding(excludeIDs []int64) (*store.Account, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	accounts, err := lb.store.GetEnabledAccounts()
	if err != nil {
		return nil, err
	}

	if len(excludeIDs) > 0 {
		excludeSet := make(map[int64]bool)
		for _, id := range excludeIDs {
			excludeSet[id] = true
		}
		var filtered []*store.Account
		for _, acc := range accounts {
			if !excludeSet[acc.ID] {
				filtered = append(filtered, acc)
			}
		}
		accounts = filtered
	}

	// Filter out unhealthy accounts if health check is enabled
	if lb.healthCheckEnabled {
		accounts = lb.filterHealthyAccounts(accounts)
	}

	// Filter out rate-limited accounts
	accounts = lb.filterRateLimitedAccounts(accounts)

	if len(accounts) == 0 {
		return nil, errors.New("no enabled accounts available")
	}

	account := lb.selectAccount(accounts)

	if err := lb.store.IncrementRequestCount(account.ID); err != nil {
		return nil, err
	}

	// Update rate limit counter
	lb.incrementRateLimit(account.ID)

	return account, nil
}

// filterHealthyAccounts returns only accounts that are considered healthy
func (lb *LoadBalancer) filterHealthyAccounts(accounts []*store.Account) []*store.Account {
	if !lb.healthCheckEnabled {
		return accounts
	}

	now := time.Now()
	var healthy []*store.Account

	for _, acc := range accounts {
		health, exists := lb.accountHealth[acc.ID]
		if !exists {
			// New account, assume healthy
			lb.accountHealth[acc.ID] = &AccountHealth{
				ID:        acc.ID,
				IsHealthy: true,
			}
			healthy = append(healthy, acc)
			continue
		}

		// Check if cooldown period has passed for unhealthy accounts
		if !health.IsHealthy && now.Sub(health.LastFailure) > lb.failureCooldown {
			health.IsHealthy = true
			health.ConsecutiveFails = 0
			log.Printf("Account %d health restored after cooldown", acc.ID)
		}

		if health.IsHealthy {
			healthy = append(healthy, acc)
		}
	}

	// If all accounts are unhealthy, return all accounts
	if len(healthy) == 0 {
		log.Println("All accounts unhealthy, returning all accounts")
		return accounts
	}

	return healthy
}

// filterRateLimitedAccounts removes accounts that have exceeded rate limits
func (lb *LoadBalancer) filterRateLimitedAccounts(accounts []*store.Account) []*store.Account {
	now := time.Now()
	var available []*store.Account

	for _, acc := range accounts {
		counter, exists := lb.rateLimitCounters[acc.ID]
		if !exists {
			lb.rateLimitCounters[acc.ID] = &rateLimitCounter{
				minuteStart: now,
				hourStart:   now,
			}
			available = append(available, acc)
			continue
		}

		counter.mu.Lock()
		// Reset minute counter if needed
		if now.Sub(counter.minuteStart) > time.Minute {
			counter.minuteCount = 0
			counter.minuteStart = now
		}
		// Reset hour counter if needed
		if now.Sub(counter.hourStart) > time.Hour {
			counter.hourCount = 0
			counter.hourStart = now
		}

		withinLimits := counter.minuteCount < lb.rateLimitConfig.RequestsPerMinute &&
			counter.hourCount < lb.rateLimitConfig.RequestsPerHour
		counter.mu.Unlock()

		if withinLimits {
			available = append(available, acc)
		}
	}

	// If all accounts are rate-limited, return all accounts
	if len(available) == 0 {
		log.Println("All accounts rate-limited, returning all accounts")
		return accounts
	}

	return available
}

// incrementRateLimit updates rate limit counters for an account
func (lb *LoadBalancer) incrementRateLimit(accountID int64) {
	counter, exists := lb.rateLimitCounters[accountID]
	if !exists {
		lb.rateLimitCounters[accountID] = &rateLimitCounter{
			minuteCount: 1,
			hourCount:   1,
			minuteStart: time.Now(),
			hourStart:   time.Now(),
		}
		return
	}

	counter.mu.Lock()
	defer counter.mu.Unlock()
	counter.minuteCount++
	counter.hourCount++
}

func (lb *LoadBalancer) selectAccount(accounts []*store.Account) *store.Account {
	if len(accounts) == 1 {
		return accounts[0]
	}

	switch lb.strategy {
	case StrategyRoundRobin:
		return lb.selectRoundRobin(accounts)
	case StrategyLeastConnections:
		return lb.selectLeastConnections(accounts)
	default:
		return lb.selectWeightedRandom(accounts)
	}
}

// selectWeightedRandom uses weighted random selection
func (lb *LoadBalancer) selectWeightedRandom(accounts []*store.Account) *store.Account {
	var totalWeight int
	for _, acc := range accounts {
		totalWeight += acc.Weight
	}

	randomWeight := rand.Intn(totalWeight)
	currentWeight := 0

	for _, acc := range accounts {
		currentWeight += acc.Weight
		if currentWeight > randomWeight {
			return acc
		}
	}

	return accounts[0]
}

// selectRoundRobin selects accounts in round-robin fashion
func (lb *LoadBalancer) selectRoundRobin(accounts []*store.Account) *store.Account {
	lb.roundRobinIndex = lb.roundRobinIndex % len(accounts)
	account := accounts[lb.roundRobinIndex]
	lb.roundRobinIndex++
	return account
}

// selectLeastConnections selects the account with the least request count
func (lb *LoadBalancer) selectLeastConnections(accounts []*store.Account) *store.Account {
	if len(accounts) == 0 {
		return nil
	}

	selected := accounts[0]
	for _, acc := range accounts[1:] {
		if acc.RequestCount < selected.RequestCount {
			selected = acc
		}
	}
	return selected
}

// ReportSuccess reports a successful request for an account
func (lb *LoadBalancer) ReportSuccess(accountID int64) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	health, exists := lb.accountHealth[accountID]
	if !exists {
		lb.accountHealth[accountID] = &AccountHealth{
			ID:            accountID,
			IsHealthy:     true,
			LastSuccess:   time.Now(),
			TotalRequests: 1,
		}
		return
	}

	health.LastSuccess = time.Now()
	health.ConsecutiveFails = 0
	health.IsHealthy = true
	health.TotalRequests++
}

// ReportFailure reports a failed request for an account
func (lb *LoadBalancer) ReportFailure(accountID int64) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	health, exists := lb.accountHealth[accountID]
	if !exists {
		lb.accountHealth[accountID] = &AccountHealth{
			ID:              accountID,
			IsHealthy:       true,
			LastFailure:     time.Now(),
			FailureCount:    1,
			ConsecutiveFails: 1,
			TotalFailures:   1,
		}
		return
	}

	health.LastFailure = time.Now()
	health.FailureCount++
	health.ConsecutiveFails++
	health.TotalFailures++

	if health.ConsecutiveFails >= lb.maxConsecutiveFails {
		health.IsHealthy = false
		log.Printf("Account %d marked unhealthy after %d consecutive failures", accountID, health.ConsecutiveFails)
	}
}

// GetAccountHealth returns health status for an account
func (lb *LoadBalancer) GetAccountHealth(accountID int64) *AccountHealth {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if health, exists := lb.accountHealth[accountID]; exists {
		return health
	}
	return nil
}

// GetAllAccountHealth returns health status for all accounts
func (lb *LoadBalancer) GetAllAccountHealth() map[int64]*AccountHealth {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make(map[int64]*AccountHealth)
	for id, health := range lb.accountHealth {
		result[id] = health
	}
	return result
}

// SetStrategy changes the load balancing strategy
func (lb *LoadBalancer) SetStrategy(strategy Strategy) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.strategy = strategy
}

// GetStrategy returns the current load balancing strategy
func (lb *LoadBalancer) GetStrategy() Strategy {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.strategy
}

// GetStats returns load balancer statistics
func (lb *LoadBalancer) GetStats() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	healthyCount := 0
	unhealthyCount := 0
	for _, health := range lb.accountHealth {
		if health.IsHealthy {
			healthyCount++
		} else {
			unhealthyCount++
		}
	}

	return map[string]interface{}{
		"strategy":        string(lb.strategy),
		"healthy_count":   healthyCount,
		"unhealthy_count": unhealthyCount,
		"health_enabled":  lb.healthCheckEnabled,
	}
}
