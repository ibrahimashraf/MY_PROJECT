package equipment

import (
	"errors"
)

type AssetCustomFields map[string]any

type AssetSummary struct {
	ID              string
	AssetID         string
	SerialNumber    string
	Description     string
	BranchID        string
	AreaID          string
	ZoneID          string
	EquipmentTypeID string
	Status          string
}

var ErrInvalidActor = errors.New("equipment actor context is invalid")
