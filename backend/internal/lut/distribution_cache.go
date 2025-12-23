package lut

import (
	"fmt"
	"sort"
	"sync"

	"stakergs"
)

// DistributionCache caches pre-computed distribution data per mode.
type DistributionCache struct {
	mu    sync.RWMutex
	cache map[string]*CachedDistribution

	// Track which modes are being generated
	generating   map[string]bool
	generatingMu sync.Mutex
}

// CachedDistribution holds pre-computed distribution for a mode.
type CachedDistribution struct {
	// Full distribution sorted by payout descending
	Items []DistributionItem

	// Items grouped by bucket key (range_start-range_end)
	ByBucket map[string][]DistributionItem

	// Bucket definitions
	Buckets []PayoutBucket

	// Total weight for odds calculation
	TotalWeight uint64

	// Max payout for bucket boundary detection
	MaxPayout float64

	// Ready flag - true when generation is complete
	Ready bool
}

// NewDistributionCache creates a new distribution cache.
func NewDistributionCache() *DistributionCache {
	return &DistributionCache{
		cache:      make(map[string]*CachedDistribution),
		generating: make(map[string]bool),
	}
}

// Get returns cached distribution for a mode, or nil if not cached.
func (c *DistributionCache) Get(mode string) *CachedDistribution {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache[mode]
}

// GetBucketItems returns items for a specific bucket with pagination.
// Returns nil if not cached or bucket not found.
func (c *DistributionCache) GetBucketItems(mode string, rangeStart, rangeEnd float64, offset, limit int) *BucketDistributionResponse {
	c.mu.RLock()
	cached := c.cache[mode]
	c.mu.RUnlock()

	if cached == nil || !cached.Ready {
		return nil
	}

	key := bucketKey(rangeStart, rangeEnd)
	items, ok := cached.ByBucket[key]
	if !ok {
		// Return empty response for unknown bucket
		return &BucketDistributionResponse{
			RangeStart: rangeStart,
			RangeEnd:   rangeEnd,
			Items:      []DistributionItem{},
			Total:      0,
			Offset:     offset,
			Limit:      limit,
			HasMore:    false,
		}
	}

	total := len(items)

	// Apply pagination
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	end := offset + limit
	if end > total {
		end = total
	}

	paginatedItems := items[offset:end]

	return &BucketDistributionResponse{
		RangeStart: rangeStart,
		RangeEnd:   rangeEnd,
		Items:      paginatedItems,
		Total:      total,
		Offset:     offset,
		Limit:      limit,
		HasMore:    end < total,
	}
}

// IsGenerating returns true if distribution is being generated for a mode.
func (c *DistributionCache) IsGenerating(mode string) bool {
	c.generatingMu.Lock()
	defer c.generatingMu.Unlock()
	return c.generating[mode]
}

// StartGenerating marks a mode as being generated. Returns false if already generating.
func (c *DistributionCache) StartGenerating(mode string) bool {
	c.generatingMu.Lock()
	defer c.generatingMu.Unlock()

	if c.generating[mode] {
		return false
	}
	c.generating[mode] = true
	return true
}

// FinishGenerating marks generation as complete for a mode.
func (c *DistributionCache) FinishGenerating(mode string) {
	c.generatingMu.Lock()
	defer c.generatingMu.Unlock()
	delete(c.generating, mode)
}

// GenerateAsync starts background generation of distribution for a mode.
func (c *DistributionCache) GenerateAsync(mode string, lut *stakergs.LookupTable, buckets []PayoutBucket) {
	if !c.StartGenerating(mode) {
		return // Already generating
	}

	go func() {
		defer c.FinishGenerating(mode)
		c.Generate(mode, lut, buckets)
	}()
}

// Generate computes and caches distribution for a mode.
func (c *DistributionCache) Generate(mode string, lut *stakergs.LookupTable, buckets []PayoutBucket) {
	totalWeight := lut.TotalWeight()
	if totalWeight == 0 || len(lut.Outcomes) == 0 {
		return
	}

	maxPayout := float64(lut.MaxPayout()) / 100.0

	// Create initial cache entry (not ready yet)
	cached := &CachedDistribution{
		TotalWeight: totalWeight,
		MaxPayout:   maxPayout,
		Buckets:     buckets,
		ByBucket:    make(map[string][]DistributionItem),
		Ready:       false,
	}

	// Group outcomes by payout value
	type payoutData struct {
		weight uint64
		simIDs []int
	}
	payoutMap := make(map[uint]*payoutData)

	for _, o := range lut.Outcomes {
		if payoutMap[o.Payout] == nil {
			payoutMap[o.Payout] = &payoutData{}
		}
		payoutMap[o.Payout].weight += o.Weight
		payoutMap[o.Payout].simIDs = append(payoutMap[o.Payout].simIDs, o.SimID)
	}

	// Convert to DistributionItem slice
	items := make([]DistributionItem, 0, len(payoutMap))
	for payout, data := range payoutMap {
		odds := float64(totalWeight) / float64(data.weight)

		// Keep only first 10 sim_ids
		simIDs := data.simIDs
		if len(simIDs) > 10 {
			simIDs = simIDs[:10]
		}

		items = append(items, DistributionItem{
			Payout: round2(float64(payout) / 100.0),
			Weight: data.weight,
			Odds:   formatOdds(odds),
			Count:  len(data.simIDs),
			SimIDs: simIDs,
		})
	}

	// Sort by payout descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].Payout > items[j].Payout
	})

	cached.Items = items

	// Pre-compute by-bucket groupings
	maxRangeEnd := 0.0
	for _, b := range buckets {
		if b.RangeEnd > maxRangeEnd {
			maxRangeEnd = b.RangeEnd
		}
	}

	for _, bucket := range buckets {
		key := bucketKey(bucket.RangeStart, bucket.RangeEnd)
		bucketItems := make([]DistributionItem, 0)

		for _, item := range items {
			inBucket := false

			if bucket.RangeStart == 0 && bucket.RangeEnd == 0 {
				// Zero bucket: exact match
				inBucket = item.Payout == 0
			} else if bucket.RangeEnd >= maxRangeEnd*0.99 {
				// Last bucket: include all >= range_start
				inBucket = item.Payout >= bucket.RangeStart
			} else {
				// Normal bucket: [start, end)
				inBucket = item.Payout >= bucket.RangeStart && item.Payout < bucket.RangeEnd
			}

			if inBucket {
				bucketItems = append(bucketItems, item)
			}
		}

		cached.ByBucket[key] = bucketItems
	}

	cached.Ready = true

	// Store in cache
	c.mu.Lock()
	c.cache[mode] = cached
	c.mu.Unlock()
}

// Invalidate removes cached distribution for a mode.
func (c *DistributionCache) Invalidate(mode string) {
	c.mu.Lock()
	delete(c.cache, mode)
	c.mu.Unlock()
}

// InvalidateAll clears the entire cache.
func (c *DistributionCache) InvalidateAll() {
	c.mu.Lock()
	c.cache = make(map[string]*CachedDistribution)
	c.mu.Unlock()
}

func bucketKey(rangeStart, rangeEnd float64) string {
	return fmt.Sprintf("%.2f-%.2f", rangeStart, rangeEnd)
}
