package rigging

import "math"

// AngleLimits defines rotational limits (min/max radians) along X, Y, Z.
type AngleLimits struct {
	MinX, MaxX float64
	MinY, MaxY float64
	MinZ, MaxZ float64
}

// DefaultLimits allows full 360 degree rotation.
func DefaultLimits() AngleLimits {
	return AngleLimits{
		MinX: -math.Pi, MaxX: math.Pi,
		MinY: -math.Pi, MaxY: math.Pi,
		MinZ: -math.Pi, MaxZ: math.Pi,
	}
}

// Bone represents an individual skeletal joint node.
type Bone struct {
	ID              int         `json:"id"`
	Name            string      `json:"name"`
	ParentID        int         `json:"parent_id"` // -1 indicates root bone
	LocalPos        [3]float64  `json:"local_pos"`
	LocalRot        [3]float64  `json:"local_rot"` // Euler angles in radians (X, Y, Z)
	GlobalTransform Mat4        `json:"global_transform"`
	Limits          AngleLimits `json:"limits"`
}

// Skeleton encapsulates a hierarchical bone tree.
type Skeleton struct {
	Bones []Bone `json:"bones"`
}

// NewSkeleton initializes an empty skeleton.
func NewSkeleton() *Skeleton {
	return &Skeleton{
		Bones: make([]Bone, 0, 16),
	}
}

// AddBone appends a bone and returns its assigned ID.
func (s *Skeleton) AddBone(name string, parentID int, localPos [3]float64, limits AngleLimits) int {
	id := len(s.Bones)
	s.Bones = append(s.Bones, Bone{
		ID:              id,
		Name:            name,
		ParentID:        parentID,
		LocalPos:        localPos,
		LocalRot:        [3]float64{0, 0, 0},
		GlobalTransform: IdentityMat4(),
		Limits:          limits,
	})
	return id
}

// ComputeFK recursively solves Forward Kinematics across the bone hierarchy.
func (s *Skeleton) ComputeFK() {
	for i := range s.Bones {
		b := &s.Bones[i]

		// Local transform = Translation * RotZ * RotY * RotX
		t := TranslationMat4(b.LocalPos[0], b.LocalPos[1], b.LocalPos[2])
		rx := RotationX(b.LocalRot[0])
		ry := RotationY(b.LocalRot[1])
		rz := RotationZ(b.LocalRot[2])

		localM := t.Mul(rz.Mul(ry.Mul(rx)))

		if b.ParentID < 0 || b.ParentID >= len(s.Bones) {
			b.GlobalTransform = localM
		} else {
			parentGlobal := s.Bones[b.ParentID].GlobalTransform
			b.GlobalTransform = parentGlobal.Mul(localM)
		}
	}
}

// GetEndEffectorPos returns the world coordinate of a bone tip given bone length along local X.
func (s *Skeleton) GetEndEffectorPos(boneID int, boneLength float64) [3]float64 {
	if boneID < 0 || boneID >= len(s.Bones) {
		return [3]float64{0, 0, 0}
	}
	return s.Bones[boneID].GlobalTransform.TransformPoint([3]float64{boneLength, 0, 0})
}
