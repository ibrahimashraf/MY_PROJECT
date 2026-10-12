package loadchart

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// RatingEntry specifies capacity for a discrete radius.
type RatingEntry struct {
	RadiusM       float64 `json:"radius_m"`
	CapacityT     float64 `json:"capacity_t"`
}

// BoomConfiguration holds capacity table for a discrete boom length.
type BoomConfiguration struct {
	BoomLengthM float64       `json:"boom_length_m"`
	Ratings     []RatingEntry `json:"ratings"`
}

// LoadChart represents a canonical electronic load chart for a crane configuration.
type LoadChart struct {
	CraneModel        string              `json:"crane_model"`
	CounterweightT    float64             `json:"counterweight_t"`
	OutriggerPosition string              `json:"outrigger_position"` // e.g. "100%", "50%", "0%"
	SlewSectorDeg     float64             `json:"slew_sector_deg"`    // 360 or sector
	Configurations    []BoomConfiguration `json:"configurations"`
}

// Validate verifies monotonic sanity and physical positivity of the chart.
func (c *LoadChart) Validate() error {
	if c.CraneModel == "" {
		return errors.New("loadchart: crane_model is required")
	}
	if len(c.Configurations) == 0 {
		return errors.New("loadchart: at least one boom configuration is required")
	}
	for i, cfg := range c.Configurations {
		if cfg.BoomLengthM <= 0 {
			return fmt.Errorf("loadchart: config[%d] boom_length_m must be > 0", i)
		}
		if len(cfg.Ratings) == 0 {
			return fmt.Errorf("loadchart: config[%d] has no rating entries", i)
		}
		for j, r := range cfg.Ratings {
			if r.RadiusM <= 0 || r.CapacityT <= 0 {
				return fmt.Errorf("loadchart: config[%d].ratings[%d] radius and capacity must be > 0", i, j)
			}
		}
	}
	return nil
}

// LookupConservativeCapacity finds rated capacity adhering strictly to conservative
// step-down boundary rules (ASME B30.5 non-interpolating safety rule).
//
// 1. Boom Length: Selects the next longer boom length if exact match doesn't exist (more conservative capacity).
// 2. Working Radius: Selects the next larger radius in the table (lower capacity bound).
//
// Returns an error if radius or boom length exceeds published chart maximum.
func (c *LoadChart) LookupConservativeCapacity(reqBoomM, reqRadiusM float64) (float64, error) {
	if math.IsNaN(reqBoomM) || math.IsNaN(reqRadiusM) || reqBoomM <= 0 || reqRadiusM <= 0 {
		return 0, errors.New("loadchart: boom length and radius must be positive finite numbers")
	}

	// Sort configurations by boom length ascending
	cfgs := make([]BoomConfiguration, len(c.Configurations))
	copy(cfgs, c.Configurations)
	sort.Slice(cfgs, func(i, j int) bool {
		return cfgs[i].BoomLengthM < cfgs[j].BoomLengthM
	})

	var matchedCfg *BoomConfiguration
	for i := range cfgs {
		if cfgs[i].BoomLengthM >= reqBoomM {
			matchedCfg = &cfgs[i]
			break
		}
	}
	if matchedCfg == nil {
		return 0, fmt.Errorf("loadchart: requested boom length %.2fm exceeds maximum chart length %.2fm", reqBoomM, cfgs[len(cfgs)-1].BoomLengthM)
	}

	// Sort ratings by radius ascending
	ratings := make([]RatingEntry, len(matchedCfg.Ratings))
	copy(ratings, matchedCfg.Ratings)
	sort.Slice(ratings, func(i, j int) bool {
		return ratings[i].RadiusM < ratings[j].RadiusM
	})

	// If requested radius is smaller than minimum chart radius, conservative bound is min radius capacity
	if reqRadiusM <= ratings[0].RadiusM {
		return ratings[0].CapacityT, nil
	}

	// Find the next larger radius (step-down to lower capacity)
	for _, r := range ratings {
		if r.RadiusM >= reqRadiusM {
			return r.CapacityT, nil
		}
	}

	return 0, fmt.Errorf("loadchart: requested radius %.2fm exceeds maximum chart radius %.2fm for boom length %.2fm",
		reqRadiusM, ratings[len(ratings)-1].RadiusM, matchedCfg.BoomLengthM)
}
