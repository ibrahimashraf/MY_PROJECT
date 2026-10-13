package planetary

// Camera64 represents a 64-bit precision camera in ECEF space
type Camera64 struct {
	X, Y, Z float64 // Camera ECEF position
}

// RTCPatch holds Relative-To-Center offset data for rendering precision
type RTCPatch struct {
	CenterECEF [3]float64
	RTCOffset  [3]float32
}

// ComputeRTCOffset calculates the 32-bit RTC offset for a given patch relative to the camera.
func ComputeRTCOffset(cam Camera64, patchCenter [3]float64) [3]float32 {
	return [3]float32{
		float32(patchCenter[0] - cam.X),
		float32(patchCenter[1] - cam.Y),
		float32(patchCenter[2] - cam.Z),
	}
}
