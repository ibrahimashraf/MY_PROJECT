package photogrammetry

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func samplePointCloud() []Point3D {
	return []Point3D{
		{X: 0.0, Y: 0.0, Z: 10.0},
		{X: 10.0, Y: 0.0, Z: 10.5},
		{X: 10.0, Y: 10.0, Z: 11.0},
		{X: 0.0, Y: 10.0, Z: 10.2},
		{X: 5.0, Y: 5.0, Z: 12.0}, // Peak
	}
}

func TestComputeVolumetricSurvey_Success(t *testing.T) {
	pts := samplePointCloud()
	res, err := ComputeVolumetricSurvey("did:integin:asset:stockpile-01", pts, 10.0, 5.0)
	if err != nil {
		t.Fatalf("unexpected survey error: %v", err)
	}

	if res.PointCount != 5 {
		t.Fatalf("expected 5 points, got %d", res.PointCount)
	}

	if res.ApproximateVolumeM3 <= 0 {
		t.Fatalf("expected positive volume, got %f", res.ApproximateVolumeM3)
	}

	if res.MaxDeflectionMeters != 2.0 { // 12.0 - 10.0 = 2.0
		t.Fatalf("expected 2.0m max deflection, got %f", res.MaxDeflectionMeters)
	}
}

func TestComputeVolumetricSurvey_DeflectionBreach(t *testing.T) {
	pts := samplePointCloud()
	// Strict max permissible deflection of 1.0m (peak is 2.0m)
	res, err := ComputeVolumetricSurvey("did:integin:asset:stockpile-01", pts, 10.0, 1.0)
	if err != ErrDeflectionThreshold {
		t.Fatalf("expected ErrDeflectionThreshold, got %v", err)
	}
	if !res.DeflectionBreached {
		t.Fatalf("expected DeflectionBreached = true")
	}
}

func TestIssueSurveyReport(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	pts := samplePointCloud()
	res, err := ComputeVolumetricSurvey("did:integin:asset:stockpile-01", pts, 10.0, 5.0)
	if err != nil {
		t.Fatalf("survey failed: %v", err)
	}

	rpt, err := IssueSurveyReport(res, priv)
	if err != nil {
		t.Fatalf("report signing failed: %v", err)
	}

	if rpt.ReportDID == "" || rpt.SignatureHex == "" {
		t.Fatalf("report missing DID or signature: %+v", rpt)
	}
}

func BenchmarkComputeVolumetricSurvey_HotPath(b *testing.B) {
	pts := samplePointCloud()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = ComputeVolumetricSurvey("did:integin:asset:bench", pts, 10.0, 5.0)
	}
}
