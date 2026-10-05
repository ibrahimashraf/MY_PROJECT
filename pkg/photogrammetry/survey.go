package photogrammetry

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"
)

var (
	ErrZeroPoints             = errors.New("photogrammetry: point cloud requires at least 4 non-coplanar points")
	ErrDeflectionThreshold    = errors.New("photogrammetry: critical structural deflection threshold breached")
	ErrNegativeVolume         = errors.New("photogrammetry: computed volumetric reconciliation is non-positive")
	ErrFlightEnvelopeBreach   = errors.New("photogrammetry: telemetry points violate permitted aerial flight envelope")
	ErrZeroAssetDID           = errors.New("photogrammetry: target asset DID cannot be empty")
)

// Point3D represents a georeferenced spatial coordinate (X, Y, Z in meters or UTM).
type Point3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// BoundingBox3D defines volumetric extent.
type BoundingBox3D struct {
	Min Point3D `json:"min"`
	Max Point3D `json:"max"`
}

// VolumetricAnalysisResult encapsulates computed geometric survey metrics.
type VolumetricAnalysisResult struct {
	AssetDID             string        `json:"asset_did"`
	PointCount           int           `json:"point_count"`
	BoundingBox          BoundingBox3D `json:"bounding_box"`
	ApproximateVolumeM3  float64       `json:"approximate_volume_m3"` // Volume in cubic meters
	MaxDeflectionMeters  float64       `json:"max_deflection_meters"` // Max deviation from baseline plane/mesh
	DeflectionBreached   bool          `json:"deflection_breached"`
	EvaluatedAt          time.Time     `json:"evaluated_at"`
}

// ComputeVolumetricSurvey calculates convex bounding metrics and structural deflection.
func ComputeVolumetricSurvey(
	assetDID string,
	points []Point3D,
	baselinePlaneZ float64,
	maxPermissibleDeflection float64,
) (VolumetricAnalysisResult, error) {
	if assetDID == "" {
		return VolumetricAnalysisResult{}, ErrZeroAssetDID
	}
	if len(points) < 4 {
		return VolumetricAnalysisResult{}, ErrZeroPoints
	}

	minPt := points[0]
	maxPt := points[0]

	var sumZ float64
	maxDeflection := 0.0

	for _, pt := range points {
		if pt.X < minPt.X {
			minPt.X = pt.X
		}
		if pt.Y < minPt.Y {
			minPt.Y = pt.Y
		}
		if pt.Z < minPt.Z {
			minPt.Z = pt.Z
		}

		if pt.X > maxPt.X {
			maxPt.X = pt.X
		}
		if pt.Y > maxPt.Y {
			maxPt.Y = pt.Y
		}
		if pt.Z > maxPt.Z {
			maxPt.Z = pt.Z
		}

		sumZ += pt.Z

		deflection := math.Abs(pt.Z - baselinePlaneZ)
		if deflection > maxDeflection {
			maxDeflection = deflection
		}
	}

	dx := maxPt.X - minPt.X
	dy := maxPt.Y - minPt.Y
	dz := maxPt.Z - minPt.Z

	if dx <= 0 || dy <= 0 || dz <= 0 {
		return VolumetricAnalysisResult{}, ErrNegativeVolume
	}

	// 2.5D prism integration: Footprint Area * Average Height relative to min Z
	footprintArea := dx * dy
	avgHeight := (sumZ / float64(len(points))) - minPt.Z
	if avgHeight < 0 {
		avgHeight = 0
	}
	volume := footprintArea * avgHeight

	breached := maxPermissibleDeflection > 0 && maxDeflection > maxPermissibleDeflection

	res := VolumetricAnalysisResult{
		AssetDID:   assetDID,
		PointCount: len(points),
		BoundingBox: BoundingBox3D{
			Min: minPt,
			Max: maxPt,
		},
		ApproximateVolumeM3: volume,
		MaxDeflectionMeters: maxDeflection,
		DeflectionBreached:  breached,
		EvaluatedAt:         time.Now().UTC(),
	}

	if breached {
		return res, ErrDeflectionThreshold
	}

	return res, nil
}

// CertifiedPhotogrammetryReport represents an Ed25519-signed survey certificate.
type CertifiedPhotogrammetryReport struct {
	ReportDID       string    `json:"report_did"` // did:integin:survey:<uuid>
	AssetDID        string    `json:"asset_did"`
	PointCount      int       `json:"point_count"`
	VolumeM3        float64   `json:"volume_m3"`
	MaxDeflectionM  float64   `json:"max_deflection_m"`
	IssuedAt        time.Time `json:"issued_at"`
	Digest          string    `json:"digest"`
	SignatureHex    string    `json:"signature_hex"`
}

// IssueSurveyReport signs an authoritative photogrammetric inspection record.
func IssueSurveyReport(
	res VolumetricAnalysisResult,
	privKey ed25519.PrivateKey,
) (*CertifiedPhotogrammetryReport, error) {
	if res.DeflectionBreached {
		return nil, ErrDeflectionThreshold
	}

	reportDID := fmt.Sprintf("did:integin:survey:%s", hex.EncodeToString([]byte(fmt.Sprintf("%s-%d", res.AssetDID, res.EvaluatedAt.UnixNano()))))
	now := time.Now().UTC()

	rpt := &CertifiedPhotogrammetryReport{
		ReportDID:      reportDID,
		AssetDID:       res.AssetDID,
		PointCount:     res.PointCount,
		VolumeM3:       res.ApproximateVolumeM3,
		MaxDeflectionM: res.MaxDeflectionMeters,
		IssuedAt:       now,
	}

	payload := fmt.Sprintf("%s|%s|%d|%.4f|%.4f|%d",
		rpt.ReportDID, rpt.AssetDID, rpt.PointCount,
		rpt.VolumeM3, rpt.MaxDeflectionM, rpt.IssuedAt.Unix(),
	)
	digest := sha256.Sum256([]byte(payload))
	rpt.Digest = hex.EncodeToString(digest[:])

	sig := ed25519.Sign(privKey, digest[:])
	rpt.SignatureHex = hex.EncodeToString(sig)

	return rpt, nil
}
