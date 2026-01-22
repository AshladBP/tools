// Package convexopt provides a proxy to the Python Convex Optimizer service.
package convexopt

// DistributionType represents supported probability distribution types.
type DistributionType string

const (
	DistLogNormal   DistributionType = "log_normal"
	DistGaussian    DistributionType = "gaussian"
	DistExponential DistributionType = "exponential"
)

// DistributionParams holds parameters for a probability distribution.
type DistributionParams struct {
	Type  DistributionType `json:"type"`
	Mode  *float64         `json:"mode,omitempty"`
	Std   *float64         `json:"std,omitempty"`
	Mean  *float64         `json:"mean,omitempty"`
	Power *float64         `json:"power,omitempty"`
	Scale float64          `json:"scale"`
}

// CriteriaConfig defines an optimization criteria (e.g., basegame, freegame).
type CriteriaConfig struct {
	Name            string              `json:"name"`
	RTP             float64             `json:"rtp"`
	HitRate         float64             `json:"hit_rate"`
	AverageWin      *float64            `json:"average_win,omitempty"`
	Distribution    DistributionParams  `json:"distribution"`
	MixDistribution *DistributionParams `json:"mix_distribution,omitempty"`
	MixWeight       float64             `json:"mix_weight"`
}

// OptimizerSettings holds convex optimizer tuning parameters.
type OptimizerSettings struct {
	KLDivergenceWeight float64 `json:"kl_divergence_weight"`
	SmoothnessWeight   float64 `json:"smoothness_weight"`
}

// ConvexOptimizeRequest is the full request for convex optimization.
type ConvexOptimizeRequest struct {
	Mode              string             `json:"mode"`
	Cost              float64            `json:"cost"`
	Criteria          []CriteriaConfig   `json:"criteria"`
	OptimizerSettings []OptimizerSettings `json:"optimizer_settings"`
	WeightScale       int                `json:"weight_scale"`
	LookupFile        string             `json:"lookup_file"`
	SegmentedFile     string             `json:"segmented_file"`
	WinStepSize       float64            `json:"win_step_size"`
	ExcludedPayouts   []float64          `json:"excluded_payouts"`
	SaveToFile        bool               `json:"save_to_file"`
	CreateBackup      bool               `json:"create_backup"`
}

// HitRateRange represents hit rate for a payout range.
type HitRateRange struct {
	RangeStart float64 `json:"range_start"`
	RangeEnd   float64 `json:"range_end"`
	HitRate    float64 `json:"hit_rate"`
}

// PlotPoint represents a single point on a chart.
type PlotPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// PlotData contains data for distribution visualization.
type PlotData struct {
	ActualPoints     []PlotPoint `json:"actual_points"`
	TheoreticalCurve []PlotPoint `json:"theoretical_curve"`
	SolutionCurve    []PlotPoint `json:"solution_curve"`
	XLabel           string      `json:"x_label"`
	YLabel           string      `json:"y_label"`
	XMin             float64     `json:"x_min"`
	XMax             float64     `json:"x_max"`
	YMin             float64     `json:"y_min"`
	YMax             float64     `json:"y_max"`
}

// CriteriaSolution represents the solution for a single criteria.
type CriteriaSolution struct {
	Name              string             `json:"name"`
	TargetRTP         float64            `json:"target_rtp"`
	AchievedRTP       float64            `json:"achieved_rtp"`
	TargetHitRate     float64            `json:"target_hit_rate"`
	AchievedHitRate   float64            `json:"achieved_hit_rate"`
	SolvedWeights     []float64          `json:"solved_weights"`
	UniquePayoutCount int                `json:"unique_payout_count"`
	DistributionType  string             `json:"distribution_type"`
	HitRateRanges     []HitRateRange     `json:"hit_rate_ranges"`
	SolutionMetrics   map[string]float64 `json:"solution_metrics"`
	PlotData          *PlotData          `json:"plot_data,omitempty"`
}

// LookupEntry represents a single entry in the optimized lookup table.
type LookupEntry struct {
	SimID  int `json:"sim_id"`
	Weight int `json:"weight"`
	Payout int `json:"payout"`
}

// SaveResult contains information about saved files.
type SaveResult struct {
	Saved       bool    `json:"saved"`
	LookupPath  *string `json:"lookup_path,omitempty"`
	HitratePath *string `json:"hitrate_path,omitempty"`
	BackupPath  *string `json:"backup_path,omitempty"`
}

// ConvexOptimizeResponse is the full response from convex optimization.
type ConvexOptimizeResponse struct {
	Success             bool               `json:"success"`
	Mode                string             `json:"mode"`
	OriginalRTP         float64            `json:"original_rtp"`
	FinalRTP            float64            `json:"final_rtp"`
	CriteriaSolutions   []CriteriaSolution `json:"criteria_solutions"`
	FinalLookup         []LookupEntry      `json:"final_lookup"`
	HitRateSummary      []HitRateRange     `json:"hit_rate_summary"`
	ZeroWeightProb      float64            `json:"zero_weight_probability"`
	TotalLookupLength   int                `json:"total_lookup_length"`
	Warnings            []string           `json:"warnings"`
	SaveResult          *SaveResult        `json:"save_result,omitempty"`
}

// HealthResponse is the health check response.
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

// ModeInfoResponse contains mode information for the frontend.
type ModeInfoResponse struct {
	Mode          string   `json:"mode"`
	Cost          float64  `json:"cost"`
	CriteriaNames []string `json:"criteria_names"`
	LookupFile    string   `json:"lookup_file"`
	SegmentedFile string   `json:"segmented_file"`
	IsBonusMode   bool     `json:"is_bonus_mode"`
}
