package rulesengine

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"
)

// BLE GATT live telemetry bridge.
//
// A load-cell sensor on the field tablet advertises force readings over BLE
// GATT notifications. Every characteristic payload is 20 bytes — the standard
// GATT maximum — carrying sequence number, nanosecond hardware timestamp, raw
// force and scaled load, and is authenticated by a trailing enclave HMAC-SHA256
// (the full MTU-sized notification). The bridge decodes the notification,
// verifies the enclave binding, rejects replay/regression fail-closed, and
// feeds the resulting telemetric stream into the harmonic-jitter verifier.

const bleFrameSize = 54 // 2 magic + 20-byte payload + 32-byte enclave signature

var (
	ErrBLEFrameInvalid        = errors.New("jitter: invalid BLE GATT telemetry frame")
	ErrBLESequenceReplay      = errors.New("jitter: BLE telemetry sequence replay detected")
	ErrBLETimestampRegression = errors.New("jitter: BLE telemetry timestamp regression detected")
)

// BLEGATTFrame is one decoded load-cell notification. Signature is the raw
// 32-byte enclave HMAC-SHA256 over the 20-byte canonical payload and must be
// verified before the frame is trusted.
type BLEGATTFrame struct {
	Seq         uint32  `json:"seq"`
	TimestampNs int64   `json:"timestamp_ns"`
	RawLoad     int32   `json:"raw_load"`
	LoadKN      float32 `json:"load_kn"`
	Signature   []byte  `json:"signature"`
}

// payload returns the 20-byte signed payload: seq, timestamp, raw load, load.
func (f BLEGATTFrame) payload() ([]byte, error) {
	buf := make([]byte, 20)
	binary.LittleEndian.PutUint32(buf[0:4], f.Seq)
	binary.LittleEndian.PutUint64(buf[4:12], uint64(f.TimestampNs))
	binary.LittleEndian.PutUint32(buf[12:16], uint32(f.RawLoad))
	binary.LittleEndian.PutUint32(buf[16:20], math.Float32bits(f.LoadKN))
	return buf, nil
}

// ParseBLEFrame decodes a raw GATT notification byte stream. Frame layout
// (all little-endian):
//
//	[0:2]   magic 0x49 0x01
//	[2:6]   sequence number (uint32)
//	[6:14]  nanosecond hardware timestamp (int64)
//	[14:18] raw force/load (int32)
//	[18:22] scaled load (float32, kN)
//	[22:54] enclave HMAC-SHA256 signature (32 bytes, full MTU notification)
//
// Frames of any other length, with the wrong magic, or yielding NaN/Inf load
// values fail closed.
func ParseBLEFrame(raw []byte) (BLEGATTFrame, error) {
	var f BLEGATTFrame
	if len(raw) != bleFrameSize {
		return f, fmt.Errorf("%w: got %d bytes, want %d", ErrBLEFrameInvalid, len(raw), bleFrameSize)
	}
	if raw[0] != 0x49 || raw[1] != 0x01 {
		return f, fmt.Errorf("%w: bad magic", ErrBLEFrameInvalid)
	}
	f.Seq = binary.LittleEndian.Uint32(raw[2:6])
	f.TimestampNs = int64(binary.LittleEndian.Uint64(raw[6:14]))
	f.RawLoad = int32(binary.LittleEndian.Uint32(raw[14:18]))
	f.LoadKN = math.Float32frombits(binary.LittleEndian.Uint32(raw[18:22]))
	if math.IsNaN(float64(f.LoadKN)) || math.IsInf(float64(f.LoadKN), 0) {
		return f, fmt.Errorf("%w: NaN/Inf load reading", ErrBLEFrameInvalid)
	}
	f.Signature = append([]byte(nil), raw[22:54]...)
	return f, nil
}

// BLEGATTBridge ingests raw GATT notifications into an authenticated
// telemetric sample stream. Each packet is enclave-bound at ingest (fails
// closed on bad signature) and replay/regression is rejected fail-closed.
type BLEGATTBridge struct {
	key      []byte
	mu       sync.Mutex
	frames   []BLEGATTFrame
	lastSeq  uint32
	lastTsNs int64
	seen     bool
}

// NewBLEGATTBridge binds the bridge to the sensor enclave pairing key
// (>=16 bytes, matching VerifySensorBLEBinding).
func NewBLEGATTBridge(bleDeviceKey []byte) (*BLEGATTBridge, error) {
	if len(bleDeviceKey) < 16 {
		return nil, errors.New("jitter: BLE enclave pairing key must be at least 16 bytes")
	}
	return &BLEGATTBridge{key: append([]byte(nil), bleDeviceKey...)}, nil
}

// IngestBLEPacket parses, authenticates, and records one raw GATT notification
// frame. It fails closed on: malformed frames, invalid enclave signatures,
// sequence replay, and timestamp regression.
func (b *BLEGATTBridge) IngestBLEPacket(raw []byte) (*BLEGATTFrame, error) {
	frame, err := ParseBLEFrame(raw)
	if err != nil {
		return nil, err
	}
	payload, err := frame.payload()
	if err != nil {
		return nil, err
	}
	if err := VerifySensorBLEBinding(payload, b.key, hex.EncodeToString(frame.Signature)); err != nil {
		return nil, err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.seen {
		if frame.Seq <= b.lastSeq {
			return nil, ErrBLESequenceReplay
		}
		if frame.TimestampNs < b.lastTsNs {
			return nil, ErrBLETimestampRegression
		}
	}
	b.frames = append(b.frames, frame)
	b.lastSeq = frame.Seq
	b.lastTsNs = frame.TimestampNs
	b.seen = true
	return &frame, nil
}

// Samples returns the authenticated telemetric stream (copies, safe to read
// concurrently with IngestBLEPacket).
func (b *BLEGATTBridge) Samples() []TelemetricSample {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]TelemetricSample, 0, len(b.frames))
	for _, f := range b.frames {
		out = append(out, TelemetricSample{TimestampNs: f.TimestampNs, Value: float64(f.LoadKN)})
	}
	return out
}

// Frames returns copies of the authenticated frames in ingestion order.
func (b *BLEGATTBridge) Frames() []BLEGATTFrame {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]BLEGATTFrame, len(b.frames))
	for i, f := range b.frames {
		out[i] = f
		out[i].Signature = append([]byte(nil), f.Signature...)
	}
	return out
}

// VerifyAuthenticJitter feeds the decoded stream into VerifyHarmonicJitter and
// re-asserts the enclave binding of every packet via VerifySensorBLEBinding —
// so a tampered frame cannot enter the harmonic-analysis verdict undetected.
func (b *BLEGATTBridge) VerifyAuthenticJitter(minEntropy float64) (*JitterProfile, error) {
	frames := b.Frames()
	for _, f := range frames {
		payload, err := f.payload()
		if err != nil {
			return nil, err
		}
		if err := VerifySensorBLEBinding(payload, b.key, hex.EncodeToString(f.Signature)); err != nil {
			return nil, fmt.Errorf("%w: frame seq %d", err, f.Seq)
		}
	}
	samples := make([]TelemetricSample, len(frames))
	for i, f := range frames {
		samples[i] = TelemetricSample{TimestampNs: f.TimestampNs, Value: float64(f.LoadKN)}
	}
	return VerifyHarmonicJitter(samples, minEntropy)
}
