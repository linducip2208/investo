package service

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type FeatureFlag struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	Enabled      bool   `json:"enabled" db:"enabled"`
	Description  string `json:"description" db:"description"`
	PlanRequired string `json:"plan_required" db:"plan_required"`
}

type FeatureFlagService struct {
	DB *sqlx.DB
}

func (s *FeatureFlagService) IsEnabled(name string) bool {
	var flag FeatureFlag
	err := s.DB.Get(&flag, "SELECT * FROM feature_flags WHERE name = ?", name)
	if err != nil {
		return false
	}
	return flag.Enabled
}

func (s *FeatureFlagService) IsEnabledForUser(name string, userID int64) bool {
	flag, err := s.getFlag(name)
	if err != nil {
		return false
	}
	if !flag.Enabled {
		return false
	}
	return true
}

func (s *FeatureFlagService) CanAccess(userID int64, flagName string) bool {
	flag, err := s.getFlag(flagName)
	if err != nil {
		return false
	}
	if !flag.Enabled {
		return false
	}

	var userPlan string
	err = s.DB.Get(&userPlan, "SELECT COALESCE(plan, 'free') FROM users WHERE id = ?", userID)
	if err != nil {
		return false
	}

	return s.planMeetsRequirement(userPlan, flag.PlanRequired)
}

func (s *FeatureFlagService) planMeetsRequirement(userPlan, required string) bool {
	planLevels := map[string]int{
		"free":       1,
		"basic":      2,
		"pro":        3,
		"whitelabel": 4,
	}

	userLevel, userOk := planLevels[userPlan]
	requiredLevel, reqOk := planLevels[required]

	if !userOk || !reqOk {
		return false
	}
	return userLevel >= requiredLevel
}

func (s *FeatureFlagService) getFlag(name string) (*FeatureFlag, error) {
	var flag FeatureFlag
	err := s.DB.Get(&flag, "SELECT * FROM feature_flags WHERE name = ?", name)
	if err != nil {
		return nil, fmt.Errorf("feature flag %s not found: %w", name, err)
	}
	return &flag, nil
}

func (s *FeatureFlagService) SetFlag(name string, enabled bool) error {
	result, err := s.DB.Exec("UPDATE feature_flags SET enabled = ? WHERE name = ?", enabled, name)
	if err != nil {
		return fmt.Errorf("SetFlag update: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("feature flag %s not found", name)
	}
	return nil
}

func (s *FeatureFlagService) GetAllFlags() ([]FeatureFlag, error) {
	var flags []FeatureFlag
	err := s.DB.Select(&flags, "SELECT * FROM feature_flags ORDER BY id ASC")
	if err != nil {
		return nil, fmt.Errorf("GetAllFlags: %w", err)
	}
	if flags == nil {
		flags = []FeatureFlag{}
	}
	return flags, nil
}

func (s *FeatureFlagService) SetAllFlags(flags []FeatureFlag) error {
	for _, f := range flags {
		_, err := s.DB.Exec("UPDATE feature_flags SET enabled = ? WHERE id = ?", f.Enabled, f.ID)
		if err != nil {
			return fmt.Errorf("SetAllFlags update id %d: %w", f.ID, err)
		}
	}
	return nil
}
