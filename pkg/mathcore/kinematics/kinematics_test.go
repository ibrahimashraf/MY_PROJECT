package kinematics

import (
	"math"
	"testing"
)

func TestForwardKinematics(t *testing.T) {
	// Chain: Base (0) -> Boom1 (1) -> Boom2 (2)
	nodes := []KinematicNode{
		{
			ID:       0,
			ParentID: -1,
			Local:    TranslationMat4(10, 0, 0),
		},
		{
			ID:       1,
			ParentID: 0,
			Local:    TranslationMat4(0, 5, 0),
		},
		{
			ID:       2,
			ParentID: 1,
			Local:    TranslationMat4(0, 0, 20),
		},
	}

	ForwardKinematics(nodes)

	origin := Vec3{0, 0, 0}
	tip := nodes[2].World.TransformVec3(origin)

	expected := Vec3{10, 5, 20}
	for i := 0; i < 3; i++ {
		if math.Abs(tip[i]-expected[i]) > 1e-6 {
			t.Errorf("tip[%d] = %f, want %f", i, tip[i], expected[i])
		}
	}
}

func TestBoussinesqStress(t *testing.T) {
	// P = 1000 kN, at r = 0, z = 2m
	// sigma_z = (3 * 1000) / (2 * pi * 4) = 3000 / (8 * pi) ~ 119.366 kN/m^2
	P := 1000.0
	r := 0.0
	z := 2.0

	expected := (3.0 * P) / (2.0 * math.Pi * z * z)
	val := BoussinesqStress(P, r, z)

	if math.Abs(val-expected) > 1e-4 {
		t.Errorf("BoussinesqStress = %f, want %f", val, expected)
	}
}

// BenchmarkForwardKinematics_10000Nodes benchmarks 10,000 articulated links
func BenchmarkForwardKinematics_10000Nodes(b *testing.B) {
	const n = 10000
	nodes := make([]KinematicNode, n)
	for i := 0; i < n; i++ {
		nodes[i] = KinematicNode{
			ID:       i,
			ParentID: i - 1,
			Local:    TranslationMat4(0.1, 0.05, 0.2),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ForwardKinematics(nodes)
	}
}

// BenchmarkBoussinesqGrid_10000Points benchmarks 100x100 soil pressure grid
func BenchmarkBoussinesqGrid_10000Points(b *testing.B) {
	const (
		nx = 100
		ny = 100
	)
	out := make([]float64, nx*ny)
	P := 250.0 // 250 tonnes outrigger load

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		EvaluateGroundBearingPressureGrid(P, 5.0, 5.0, 1.5, 0.1, nx, ny, out)
	}
}

// BenchmarkTransformPointcloud_10000Vertices benchmarks 10,000 3D spatial points
func BenchmarkTransformPointcloud_10000Vertices(b *testing.B) {
	const n = 10000
	vertices := make([]Vec3, n)
	for i := 0; i < n; i++ {
		vertices[i] = Vec3{float64(i) * 0.1, float64(i) * 0.2, float64(i) * 0.3}
	}
	m := TranslationMat4(15, 25, 35).Mul(RotationZMat4(math.Pi / 4))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < n; j++ {
			vertices[j] = m.TransformVec3(vertices[j])
		}
	}
}
