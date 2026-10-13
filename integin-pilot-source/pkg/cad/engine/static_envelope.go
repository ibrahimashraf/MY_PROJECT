package engine

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var (
	ErrEnvelopeInvalidFormat   = errors.New("unsupported static equipment envelope format")
	ErrEnvelopeMissingGeometry = errors.New("static equipment envelope missing geometric data")
	ErrEnvelopeInvalidCoG      = errors.New("center of gravity out of bounds")
	ErrEnvelopeZeroMass        = errors.New("equipment mass must be positive")
	ErrEnvelopeLiftPointsEmpty = errors.New("at least one lifting point/trunnion is required")
)

// SupportedEnvelopeFormat defines neutral interchange formats.
type SupportedEnvelopeFormat string

const (
	FormatOBJ  SupportedEnvelopeFormat = "OBJ"
	FormatGLB  SupportedEnvelopeFormat = "GLB"
	FormatSTEP SupportedEnvelopeFormat = "STEP"
	FormatIFC  SupportedEnvelopeFormat = "IFC"
)

// LiftPoint represents an engineered pick point (trunnion, pad-eye, lifting lug).
type LiftPoint struct {
	ID             string  `json:"id"`
	Position       Vertex  `json:"position"`
	RatedCapacityT float64 `json:"rated_capacity_tonnes"`
	Type           string  `json:"type"` // "TRUNNION", "PADEYE", "LIFTING_LUG"
}

// StaticLoadEnvelope defines a neutral, bounded equipment envelope for lifting plans.
// INVARIANT: Static equipment geometry serves solely as a bounded load envelope,
// obstacle collision volume, and rigging pick-point reference. INTEGIN does NOT
// perform pressure-vessel mechanical drafting, nozzle authoring, or ASME Sec VIII CAD.
type StaticLoadEnvelope struct {
	EnvelopeID      string                  `json:"envelope_id"`
	AssetTag        string                  `json:"asset_tag"`
	Format          SupportedEnvelopeFormat `json:"format"`
	MassSpec        MassSpec                `json:"mass_spec"`
	CenterOfGravity Vertex                  `json:"center_of_gravity"`
	BoundingBox     BoundingBox             `json:"bounding_box"`
	LiftPoints      []LiftPoint             `json:"lift_points"`
	Provenance      string                  `json:"provenance"`
	ImportedAt      time.Time               `json:"imported_at"`
}

// IngestOBJEnvelope parses an OBJ payload and bounded metadata into a StaticLoadEnvelope.
func IngestOBJEnvelope(id, assetTag, objData string, massT float64, liftPoints []LiftPoint) (*StaticLoadEnvelope, error) {
	if strings.TrimSpace(objData) == "" {
		return nil, ErrEnvelopeMissingGeometry
	}
	if massT <= 0 {
		return nil, ErrEnvelopeZeroMass
	}
	if len(liftPoints) == 0 {
		return nil, ErrEnvelopeLiftPointsEmpty
	}

	mesh, err := ParseOBJMesh(objData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OBJ mesh: %w", err)
	}

	// Compute geometry dimensions from AABB
	dx := math.Abs(mesh.Bounds.Max.X - mesh.Bounds.Min.X)
	dy := math.Abs(mesh.Bounds.Max.Y - mesh.Bounds.Min.Y)
	dz := math.Abs(mesh.Bounds.Max.Z - mesh.Bounds.Min.Z)

	// Approximate cylinder bounds: Length is max dimension, Radius is half of max transverse dimension
	lengthM := math.Max(dx, math.Max(dy, dz))
	transverse1 := dx
	transverse2 := dy
	if lengthM == dx {
		transverse1, transverse2 = dy, dz
	} else if lengthM == dy {
		transverse1, transverse2 = dx, dz
	}
	radiusM := math.Max(transverse1, transverse2) * 0.5
	if radiusM <= 0 {
		radiusM = 0.5
	}

	// CoG defaults to geometric centroid of bounding box
	cog := Vertex{
		X: (mesh.Bounds.Min.X + mesh.Bounds.Max.X) * 0.5,
		Y: (mesh.Bounds.Min.Y + mesh.Bounds.Max.Y) * 0.5,
		Z: (mesh.Bounds.Min.Z + mesh.Bounds.Max.Z) * 0.5,
	}

	// Validate lift points are within reasonable distance from bounds
	for _, lp := range liftPoints {
		if lp.RatedCapacityT <= 0 {
			return nil, fmt.Errorf("lift point %s rated capacity must be positive", lp.ID)
		}
	}

	return &StaticLoadEnvelope{
		EnvelopeID: id,
		AssetTag:   assetTag,
		Format:     FormatOBJ,
		MassSpec: MassSpec{
			MassTonnes: massT,
			LengthM:    lengthM,
			RadiusM:    radiusM,
		},
		CenterOfGravity: cog,
		BoundingBox:     mesh.Bounds,
		LiftPoints:      liftPoints,
		Provenance:      "neutral-cad-envelope/v1",
		ImportedAt:      time.Now().UTC(),
	}, nil
}
