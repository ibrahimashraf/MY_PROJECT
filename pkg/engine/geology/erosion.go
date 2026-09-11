package geology

import (
	"math"
	"math/rand"
)

// HydraulicDroplet simulates an individual rain particle transporting sediment across heightmap.
type HydraulicDroplet struct {
	PosX, PosY   float64
	DirX, DirY   float64
	Speed        float64
	Water        float64
	Sediment     float64
}

// HydraulicErosionConfig controls deposition and erosion dynamics.
type HydraulicErosionConfig struct {
	Inertia      float64 // Direction persistence [0, 1]
	Capacity     float64 // Max sediment coefficient
	MinCapacity  float64
	ErodeSpeed   float64
	DepositSpeed float64
	Evaporate    float64 // Water evaporation per step
	Gravity      float64
	MaxSteps     int
}

// DefaultErosionConfig yields realistic fluvial valleys.
func DefaultErosionConfig() HydraulicErosionConfig {
	return HydraulicErosionConfig{
		Inertia:      0.05,
		Capacity:     4.0,
		MinCapacity:  0.01,
		ErodeSpeed:   0.3,
		DepositSpeed: 0.3,
		Evaporate:    0.01,
		Gravity:      4.0,
		MaxSteps:     30,
	}
}

// Heightmap represents continuous 2D planetary elevation grid.
type Heightmap struct {
	Width, Height int
	Data          []float64
}

// NewHeightmap initializes grid.
func NewHeightmap(w, h int) *Heightmap {
	return &Heightmap{
		Width:  w,
		Height: h,
		Data:   make([]float64, w*h),
	}
}

func (hm *Heightmap) Get(x, y int) float64 {
	if x < 0 || x >= hm.Width || y < 0 || y >= hm.Height {
		return 0
	}
	return hm.Data[y*hm.Width+x]
}

func (hm *Heightmap) Set(x, y int, val float64) {
	if x >= 0 && x < hm.Width && y >= 0 && y < hm.Height {
		hm.Data[y*hm.Width+x] = val
	}
}

// BilinearGradient computes normal gradient (gx, gy) and height at continuous (x, y).
func (hm *Heightmap) BilinearGradient(x, y float64) (height, gx, gy float64) {
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := x0 + 1
	y1 := y0 + 1

	u := x - float64(x0)
	v := y - float64(y0)

	h00 := hm.Get(x0, y0)
	h10 := hm.Get(x1, y0)
	h01 := hm.Get(x0, y1)
	h11 := hm.Get(x1, y1)

	gx = (h10-h00)*(1.0-v) + (h11-h01)*v
	gy = (h01-h00)*(1.0-u) + (h11-h10)*u
	height = h00*(1.0-u)*(1.0-v) + h10*u*(1.0-v) + h01*(1.0-u)*v + h11*u*v
	return height, gx, gy
}

// SimulateHydraulicErosion runs N droplet iterations over the heightmap.
func (hm *Heightmap) SimulateHydraulicErosion(dropletCount int, cfg HydraulicErosionConfig, seed int64) {
	r := rand.New(rand.NewSource(seed))

	for iter := 0; iter < dropletCount; iter++ {
		d := HydraulicDroplet{
			PosX:     r.Float64() * float64(hm.Width-2),
			PosY:     r.Float64() * float64(hm.Height-2),
			Speed:    1.0,
			Water:    1.0,
			Sediment: 0.0,
		}

		for step := 0; step < cfg.MaxSteps; step++ {
			nodeX := int(math.Floor(d.PosX))
			nodeY := int(math.Floor(d.PosY))
			if nodeX < 0 || nodeX >= hm.Width-1 || nodeY < 0 || nodeY >= hm.Height-1 {
				break
			}

			h, gx, gy := hm.BilinearGradient(d.PosX, d.PosY)

			// Update direction with inertia
			d.DirX = (d.DirX * cfg.Inertia) - (gx * (1.0 - cfg.Inertia))
			d.DirY = (d.DirY * cfg.Inertia) - (gy * (1.0 - cfg.Inertia))

			lenDir := math.Hypot(d.DirX, d.DirY)
			if lenDir == 0 {
				break
			}
			d.DirX /= lenDir
			d.DirY /= lenDir

			newX := d.PosX + d.DirX
			newY := d.PosY + d.DirY
			newH, _, _ := hm.BilinearGradient(newX, newY)

			hDiff := newH - h

			// Sediment capacity
			sedimentCapacity := math.Max(-hDiff*d.Speed*d.Water*cfg.Capacity, cfg.MinCapacity)

			if d.Sediment > sedimentCapacity || hDiff > 0 {
				// Deposit sediment
				var depositAmount float64
				if hDiff > 0 {
					depositAmount = math.Min(hDiff, d.Sediment)
				} else {
					depositAmount = (d.Sediment - sedimentCapacity) * cfg.DepositSpeed
				}
				d.Sediment -= depositAmount
				hm.Set(nodeX, nodeY, hm.Get(nodeX, nodeY)+depositAmount)
			} else {
				// Erode terrain
				erodeAmount := math.Min((sedimentCapacity-d.Sediment)*cfg.ErodeSpeed, -hDiff)
				d.Sediment += erodeAmount
				hm.Set(nodeX, nodeY, hm.Get(nodeX, nodeY)-erodeAmount)
			}

			d.Speed = math.Sqrt(math.Max(0.0, d.Speed*d.Speed+hDiff*cfg.Gravity))
			d.Water *= (1.0 - cfg.Evaporate)
			d.PosX = newX
			d.PosY = newY
		}
	}
}
