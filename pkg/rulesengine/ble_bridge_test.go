package rulesengine

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

var bleTestKey = []byte("INTEGIN-BLE-ENCLAVE-KEY-0123456789ABCD")

func buildBLEFrame(key []byte, seq uint32, tsNanos int64, raw int32, load float32) []byte {
	buf := make([]byte, bleFrameSize)
	buf[0], buf[1] = 0x49, 0x01
	binary.LittleEndian.PutUint32(buf[2:6], seq)
	binary.LittleEndian.PutUint64(buf[6:14], uint64(tsNanos))
	binary.LittleEndian.PutUint32(buf[14:18], uint32(raw))
	binary.LittleEndian.PutUint32(buf[18:22], math.Float32bits(load))
	mac := hmac.New(sha256.New, key)
	mac.Write(buf[2:22]) // 20-byte standard GATT payload bytes
	copy(buf[22:54], mac.Sum(nil))
	return buf
}

func TestParseBLEFrameDecodes(t *testing.T) {
	raw := buildBLEFrame(bleTestKey, 42, 1_700_000_000_000, 98_000, 98.0)
	frame, err := ParseBLEFrame(raw)
	if err != nil {
		t.Fatalf("ParseBLEFrame: %v", err)
	}
	if frame.Seq != 42 || frame.TimestampNs != 1_700_000_000_000 || frame.RawLoad != 98_000 {
		t.Fatalf("frame = %+v", frame)
	}
	if math.Abs(float64(frame.LoadKN)-98.0) > 1e-4 {
		t.Fatalf("load = %v, want 98.0", frame.LoadKN)
	}
	if len(frame.Signature) != 32 {
		t.Fatalf("signature bytes = %d, want 32", len(frame.Signature))
	}
}

func TestParseBLEFrameRejectsBadInput(t *testing.T) {
	raw := buildBLEFrame(bleTestKey, 1, 1, 1, 1.0)
	if _, err := ParseBLEFrame(raw[:20]); !errors.Is(err, ErrBLEFrameInvalid) {
		t.Fatalf("short frame err = %v", err)
	}
	bad := append([]byte(nil), raw...)
	bad[0] = 0x00
	if _, err := ParseBLEFrame(bad); !errors.Is(err, ErrBLEFrameInvalid) {
		t.Fatalf("bad magic err = %v", err)
	}
	nan := append([]byte(nil), raw...)
	binary.LittleEndian.PutUint32(nan[18:22], math.Float32bits(float32(math.NaN())))
	if _, err := ParseBLEFrame(nan); !errors.Is(err, ErrBLEFrameInvalid) {
		t.Fatalf("NaN load err = %v", err)
	}
}

func TestBLEBridgeIngestFeedsJitterStream(t *testing.T) {
	bridge, err := NewBLEGATTBridge(bleTestKey)
	if err != nil {
		t.Fatal(err)
	}
	const n = 12
	base := int64(2_000_000_000_000)
	interval := int64(50_000_000) // 20 Hz
	for i := 0; i < n; i++ {
		ripple := float32(0.3 * math.Sin(2*math.Pi*float64(i)/6))
		raw := buildBLEFrame(bleTestKey, uint32(i+1), base+int64(i)*interval, 98_000+int32(ripple*1000), 98.0+ripple)
		if _, err := bridge.IngestBLEPacket(raw); err != nil {
			t.Fatalf("ingest frame %d: %v", i, err)
		}
	}

	samples := bridge.Samples()
	if len(samples) != n {
		t.Fatalf("samples = %d, want %d", len(samples), n)
	}

	profile, err := bridge.VerifyAuthenticJitter(0)
	if err != nil {
		t.Fatalf("VerifyAuthenticJitter: %v", err)
	}
	if !profile.IsAuthentic {
		t.Fatal("stream not marked authentic")
	}
}

func TestBLEBridgeRejectsSequenceReplay(t *testing.T) {
	bridge, err := NewBLEGATTBridge(bleTestKey)
	if err != nil {
		t.Fatal(err)
	}
	raw := buildBLEFrame(bleTestKey, 7, 1_000, 10, 1.0)
	if _, err := bridge.IngestBLEPacket(raw); err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	if _, err := bridge.IngestBLEPacket(raw); !errors.Is(err, ErrBLESequenceReplay) {
		t.Fatalf("replay err = %v, want ErrBLESequenceReplay", err)
	}
	frame7less := buildBLEFrame(bleTestKey, 5, 3_000, 10, 1.0)
	if _, err := bridge.IngestBLEPacket(frame7less); !errors.Is(err, ErrBLESequenceReplay) {
		t.Fatalf("stale-sequence err = %v, want ErrBLESequenceReplay", err)
	}
}

func TestBLEBridgeRejectsTimestampRegression(t *testing.T) {
	bridge, err := NewBLEGATTBridge(bleTestKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.IngestBLEPacket(buildBLEFrame(bleTestKey, 1, 5_000, 10, 1.0)); err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	regressed := buildBLEFrame(bleTestKey, 2, 4_000, 10, 1.0) // seq advances but clock regresses
	if _, err := bridge.IngestBLEPacket(regressed); !errors.Is(err, ErrBLETimestampRegression) {
		t.Fatalf("regression err = %v, want ErrBLETimestampRegression", err)
	}
}

func TestBLEBridgeRejectsTamperedSignature(t *testing.T) {
	bridge, err := NewBLEGATTBridge(bleTestKey)
	if err != nil {
		t.Fatal(err)
	}
	raw := buildBLEFrame(bleTestKey, 1, 1_000, 10, 1.0)
	raw[22] ^= 0xFF
	if _, err := bridge.IngestBLEPacket(raw); !errors.Is(err, ErrInvalidSensorSignature) {
		t.Fatalf("tampered signature err = %v, want ErrInvalidSensorSignature", err)
	}
	if len(bridge.Frames()) != 0 {
		t.Fatal("tampered frame recorded")
	}
}

func TestBLEBridgeVerifyAuthenticJitterCatchesTamperedValueAtRest(t *testing.T) {
	bridge, err := NewBLEGATTBridge(bleTestKey)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		raw := buildBLEFrame(bleTestKey, uint32(i+1), int64(i)*50_000_000, 98_000, 98.0+float32(0.1*math.Sin(float64(i))))
		if _, err := bridge.IngestBLEPacket(raw); err != nil {
			t.Fatalf("ingest frame %d: %v", i, err)
		}
	}
	bridge.mu.Lock()
	bridge.frames[3].LoadKN = 4_000_000 // tampered load at rest
	bridge.mu.Unlock()

	if _, err := bridge.VerifyAuthenticJitter(0); err == nil {
		t.Fatal("tampered at-rest value passed stream verification")
	}
}
