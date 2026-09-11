package advisory

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const sealGenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

type SealRecord struct {
	ModelID     string    `json:"model_id"`
	WeightsHash string    `json:"weights_hash"`
	PromptHash  string    `json:"prompt_hash"`
	TensorHash  string    `json:"tensor_hash"`
	EvalTime    time.Time `json:"eval_time"`
	PrevSeal    string    `json:"prev_seal"`
	SealHash    string    `json:"seal_hash"`
}

type LedgerAppendFunc func(SealRecord) error

func sealHash(b []byte) string {
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h)
}

func canonicalAppendLen(buf []byte, s string) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(len(s)))
	buf = append(buf, b...)
	buf = append(buf, []byte(s)...)
	return buf
}

func canonicalSealPayload(r SealRecord) []byte {
	var buf []byte
	buf = canonicalAppendLen(buf, r.ModelID)
	buf = canonicalAppendLen(buf, r.WeightsHash)
	buf = canonicalAppendLen(buf, r.PromptHash)
	buf = canonicalAppendLen(buf, r.TensorHash)
	tbuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(tbuf, uint64(r.EvalTime.UnixNano()))
	buf = append(buf, tbuf...)
	buf = canonicalAppendLen(buf, r.PrevSeal)
	return buf
}

func hashWeights(r io.Reader) (string, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", errors.New("sealer: weights must not be empty")
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func hashBytes(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("sealer: input must not be empty")
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h), nil
}

func Seal(weights io.Reader, prompt string, tensors []byte, modelID string, evalTime time.Time, prevSeal string) (SealRecord, error) {
	if weights == nil {
		return SealRecord{}, errors.New("sealer: weights reader is nil")
	}
	if strings.TrimSpace(prompt) == "" {
		return SealRecord{}, errors.New("sealer: prompt must not be empty")
	}
	if modelID == "" {
		return SealRecord{}, errors.New("sealer: model id must not be empty")
	}
	if evalTime.IsZero() {
		return SealRecord{}, errors.New("sealer: eval time must not be zero")
	}
	weightsHash, err := hashWeights(weights)
	if err != nil {
		return SealRecord{}, err
	}
	normalizedPrompt := strings.TrimSpace(prompt)
	promptHash, err := hashBytes([]byte(normalizedPrompt))
	if err != nil {
		return SealRecord{}, err
	}
	tensorHash, err := hashBytes(tensors)
	if err != nil {
		return SealRecord{}, err
	}
	if prevSeal == "" {
		prevSeal = sealGenesisHash
	}
	rec := SealRecord{
		ModelID:     modelID,
		WeightsHash: weightsHash,
		PromptHash:  promptHash,
		TensorHash:  tensorHash,
		EvalTime:    evalTime,
		PrevSeal:    prevSeal,
	}
	rec.SealHash = sealHash(canonicalSealPayload(rec))
	return rec, nil
}

func VerifySeal(rec SealRecord) error {
	if rec.ModelID == "" {
		return errors.New("sealer: model id must not be empty")
	}
	if rec.WeightsHash == "" {
		return errors.New("sealer: weights hash must not be empty")
	}
	if rec.PromptHash == "" {
		return errors.New("sealer: prompt hash must not be empty")
	}
	if rec.TensorHash == "" {
		return errors.New("sealer: tensor hash must not be empty")
	}
	if rec.EvalTime.IsZero() {
		return errors.New("sealer: eval time must not be zero")
	}
	if rec.PrevSeal == "" {
		return errors.New("sealer: prev seal must not be empty")
	}
	if rec.SealHash == "" {
		return errors.New("sealer: seal hash must not be empty")
	}
	expected := sealHash(canonicalSealPayload(rec))
	if rec.SealHash != expected {
		return errors.New("sealer: seal hash mismatch — tampered record")
	}
	return nil
}

func VerifyChain(records []SealRecord) error {
	if len(records) == 0 {
		return errors.New("sealer: chain must not be empty")
	}
	for i := range records {
		if err := VerifySeal(records[i]); err != nil {
			return fmt.Errorf("sealer: record %d: %w", i, err)
		}
		if i == 0 {
			if records[i].PrevSeal != sealGenesisHash {
				return errors.New("sealer: first record must commit to genesis hash")
			}
			continue
		}
		if records[i].PrevSeal != records[i-1].SealHash {
			return fmt.Errorf("sealer: record %d prev seal does not match record %d seal hash", i, i-1)
		}
	}
	return nil
}

func VerifyArtifacts(rec SealRecord, weights io.Reader, prompt string, tensors []byte) error {
	if weights == nil {
		return errors.New("sealer: weights reader is nil")
	}
	weightsHash, err := hashWeights(weights)
	if err != nil {
		return err
	}
	if weightsHash != rec.WeightsHash {
		return errors.New("sealer: weights digest mismatch")
	}
	normalizedPrompt := strings.TrimSpace(prompt)
	if normalizedPrompt == "" {
		return errors.New("sealer: prompt must not be empty")
	}
	promptHash, err := hashBytes([]byte(normalizedPrompt))
	if err != nil {
		return err
	}
	if promptHash != rec.PromptHash {
		return errors.New("sealer: prompt digest mismatch")
	}
	tensorHash, err := hashBytes(tensors)
	if err != nil {
		return err
	}
	if tensorHash != rec.TensorHash {
		return errors.New("sealer: tensor digest mismatch")
	}
	return VerifySeal(rec)
}
