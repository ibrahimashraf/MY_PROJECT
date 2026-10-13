package rigging

// SkinVertex represents a vertex bound to up to 4 influencing bones.
type SkinVertex struct {
	BasePos [3]float32 `json:"base_pos"`
	Joints  [4]uint16  `json:"joints"`
	Weights [4]float32 `json:"weights"`
}

// NormalizeWeights ensures the 4 bone influence weights sum to exactly 1.0.
func (sv *SkinVertex) NormalizeWeights() {
	sum := sv.Weights[0] + sv.Weights[1] + sv.Weights[2] + sv.Weights[3]
	if sum > 1e-6 {
		invSum := 1.0 / sum
		sv.Weights[0] *= invSum
		sv.Weights[1] *= invSum
		sv.Weights[2] *= invSum
		sv.Weights[3] *= invSum
	} else {
		// Default to 100% influence on first joint
		sv.Weights[0] = 1.0
		sv.Weights[1] = 0.0
		sv.Weights[2] = 0.0
		sv.Weights[3] = 0.0
	}
}

// LinearBlendSkinning applies joint transformations to a list of skinned vertices.
// boneTransforms: slice of 4x4 matrix transforms indexed by joint index.
// Deforms vertices in place with zero heap allocation.
func LinearBlendSkinning(
	vertices []SkinVertex,
	boneTransforms []Mat4,
	outDeformed [][3]float32,
) {
	numVerts := len(vertices)
	if len(outDeformed) < numVerts {
		return
	}

	for i := 0; i < numVerts; i++ {
		sv := &vertices[i]
		p := sv.BasePos

		var dx, dy, dz float64

		for j := 0; j < 4; j++ {
			w := float64(sv.Weights[j])
			if w <= 1e-6 {
				continue
			}

			jointIdx := int(sv.Joints[j])
			if jointIdx >= len(boneTransforms) {
				continue
			}

			m := boneTransforms[jointIdx]
			tx := m[0]*float64(p[0]) + m[4]*float64(p[1]) + m[8]*float64(p[2]) + m[12]
			ty := m[1]*float64(p[0]) + m[5]*float64(p[1]) + m[9]*float64(p[2]) + m[13]
			tz := m[2]*float64(p[0]) + m[6]*float64(p[1]) + m[10]*float64(p[2]) + m[14]

			dx += tx * w
			dy += ty * w
			dz += tz * w
		}

		outDeformed[i] = [3]float32{float32(dx), float32(dy), float32(dz)}
	}
}
