// Package units provides type-safe SI unit primitives preventing cross-domain confusion.
package units

import "math"

// --- Length ---
type Meters float64
type Kilometers float64

func (m Meters) ToKilometers() Kilometers { return Kilometers(m / 1000.0) }
func (k Kilometers) ToMeters() Meters     { return Meters(k * 1000.0) }

// --- Mass ---
type Kilograms float64
type Tonnes float64

func (kg Kilograms) ToTonnes() Tonnes   { return Tonnes(kg / 1000.0) }
func (t Tonnes) ToKilograms() Kilograms { return Kilograms(t * 1000.0) }

// --- Force ---
type Newtons float64
type Kilonewtons float64

func (n Newtons) ToKilonewtons() Kilonewtons { return Kilonewtons(n / 1000.0) }
func (kn Kilonewtons) ToNewtons() Newtons    { return Newtons(kn * 1000.0) }

// --- Pressure / Stress ---
type Pascals float64
type Megapascals float64
type Gigapascals float64

func (p Pascals) ToMegapascals() Megapascals { return Megapascals(p / 1e6) }
func (p Pascals) ToGigapascals() Gigapascals { return Gigapascals(p / 1e9) }
func (mp Megapascals) ToPascals() Pascals    { return Pascals(mp * 1e6) }
func (gp Gigapascals) ToPascals() Pascals    { return Pascals(gp * 1e9) }

// --- Temperature ---
type Kelvin float64
type Celsius float64

func (k Kelvin) ToCelsius() Celsius { return Celsius(k - 273.15) }
func (c Celsius) ToKelvin() Kelvin  { return Kelvin(c + 273.15) }

// --- Angles ---
type Radians float64
type Degrees float64

func (r Radians) ToDegrees() Degrees { return Degrees(r * 180.0 / math.Pi) }
func (d Degrees) ToRadians() Radians { return Radians(d * math.Pi / 180.0) }

// --- Energy ---
type Joules float64
type Kilojoules float64
type Watts float64

func (j Joules) ToKilojoules() Kilojoules { return Kilojoules(j / 1000.0) }

// --- Time ---
type Seconds float64
type Milliseconds float64
type Hours float64

func (s Seconds) ToMilliseconds() Milliseconds { return Milliseconds(s * 1000.0) }
func (s Seconds) ToHours() Hours               { return Hours(s / 3600.0) }
func (ms Milliseconds) ToSeconds() Seconds     { return Seconds(ms / 1000.0) }

// --- Frequency ---
type Hertz float64
type Kilohertz float64

func (hz Hertz) ToKilohertz() Kilohertz { return Kilohertz(hz / 1000.0) }
func (khz Kilohertz) ToHertz() Hertz    { return Hertz(khz * 1000.0) }

// --- Velocity ---
type MetersPerSecond float64

// MachNumber computes dimensionless Mach number given speed of sound.
func (v MetersPerSecond) MachNumber(speedOfSoundMs MetersPerSecond) float64 {
	if speedOfSoundMs == 0 {
		return math.Inf(1)
	}
	return float64(v) / float64(speedOfSoundMs)
}
