package domain

import (
	"errors"
	"testing"
	"time"

	"integin/pkg/id"
)

func fencedPassport() *UniversalAssetPassport {
	return &UniversalAssetPassport{
		AssetDID:         "did:integin:asset:7f8a9e",
		Manufacturer:     "Liebherr",
		ChassisSerial:    "SER-0001",
		CurrentStatus:    AssetStatusOperational,
		JurisdictionCode: "SA",
	}
}

func fencedTransfer() CustodyTransferEvent {
	return CustodyTransferEvent{
		PreviousTenantID: "tenant-saudi",
		NewTenantID:      "tenant-uae-offshore",
		CountryCodeISO2:  "AE",
		TransferDate:     time.Now().UTC(),
		AuthorizedBy:     "QA-Director-Ahmed",
		ReceiptHash:      "hash-123456",
		ExpectedEpoch:    0,
	}
}

func TestTransferCustodyFencedHappyPath(t *testing.T) {
	p := fencedPassport()
	if err := p.TransferCustodyFenced(fencedTransfer()); err != nil {
		t.Fatalf("transfer failed: %v", err)
	}
	if p.Epoch != 1 {
		t.Fatalf("epoch = %d, want 1", p.Epoch)
	}
	if p.JurisdictionCode != "AE" {
		t.Fatalf("jurisdiction = %q, want AE", p.JurisdictionCode)
	}
	if len(p.ChainOfCustody) != 1 {
		t.Fatalf("chain length = %d, want 1", len(p.ChainOfCustody))
	}
	eventID := p.ChainOfCustody[0].EventID
	if !id.IsValidV7(eventID) {
		t.Fatalf("auto-generated EventID %q is not a valid UUIDv7", eventID)
	}
}

func TestTransferCustodyFencedStaleReplayRejected(t *testing.T) {
	p := fencedPassport()
	if err := p.TransferCustodyFenced(fencedTransfer()); err != nil {
		t.Fatalf("first transfer failed: %v", err)
	}

	replayed := fencedTransfer() // still expects epoch 0
	err := p.TransferCustodyFenced(replayed)
	if !errors.Is(err, ErrEpochMismatch) {
		t.Fatalf("stale replay err = %v, want ErrEpochMismatch", err)
	}
	if p.Epoch != 1 {
		t.Fatalf("epoch advanced to %d after rejected replay, want 1", p.Epoch)
	}
	if len(p.ChainOfCustody) != 1 {
		t.Fatalf("chain length = %d after rejected replay, want 1", len(p.ChainOfCustody))
	}
}

func TestTransferCustodyFencedSplitBrainWrongEpochRejected(t *testing.T) {
	p := fencedPassport()
	sb := fencedTransfer()
	sb.ExpectedEpoch = 3 // split-brain custodian believes it is at epoch 3
	if err := p.TransferCustodyFenced(sb); !errors.Is(err, ErrEpochMismatch) {
		t.Fatalf("err = %v, want ErrEpochMismatch", err)
	}
	if p.Epoch != 0 {
		t.Fatalf("epoch = %d, want 0", p.Epoch)
	}
}

func TestTransferCustodyFencedSequentialTransfersRequireNextEpoch(t *testing.T) {
	p := fencedPassport()
	first := fencedTransfer()
	if err := p.TransferCustodyFenced(first); err != nil {
		t.Fatalf("first transfer failed: %v", err)
	}

	second := fencedTransfer()
	second.ExpectedEpoch = 1
	second.NewTenantID = "tenant-eu-hub"
	second.CountryCodeISO2 = "DE"
	if err := p.TransferCustodyFenced(second); err != nil {
		t.Fatalf("second transfer failed: %v", err)
	}
	if p.Epoch != 2 {
		t.Fatalf("epoch = %d, want 2", p.Epoch)
	}
	if p.JurisdictionCode != "DE" {
		t.Fatalf("jurisdiction = %q, want DE", p.JurisdictionCode)
	}
	if !id.IsValidV7(p.ChainOfCustody[1].EventID) {
		t.Fatalf("second EventID %q is not a valid UUIDv7", p.ChainOfCustody[1].EventID)
	}
}

func TestTransferCustodyFencedProvidedEventIDAccepted(t *testing.T) {
	p := fencedPassport()
	provided, err := id.NewV7()
	if err != nil {
		t.Fatalf("NewV7: %v", err)
	}
	x := fencedTransfer()
	x.EventID = provided
	if err := p.TransferCustodyFenced(x); err != nil {
		t.Fatalf("transfer failed: %v", err)
	}
	if got := p.ChainOfCustody[0].EventID; got != provided {
		t.Fatalf("EventID = %q, want %q", got, provided)
	}
}

func TestTransferCustodyFencedInvalidEventIDRejected(t *testing.T) {
	p := fencedPassport()
	x := fencedTransfer()
	x.EventID = "EV-9901" // not a UUIDv7
	if err := p.TransferCustodyFenced(x); !errors.Is(err, ErrInvalidEventID) {
		t.Fatalf("err = %v, want ErrInvalidEventID", err)
	}
	if p.Epoch != 0 || len(p.ChainOfCustody) != 0 {
		t.Fatalf("state mutated on rejected event id: epoch=%d chain=%d", p.Epoch, len(p.ChainOfCustody))
	}
}

func TestTransferCustodyFencedMissingRecipientRejected(t *testing.T) {
	p := fencedPassport()
	x := fencedTransfer()
	x.NewTenantID = ""
	if err := p.TransferCustodyFenced(x); !errors.Is(err, ErrInvalidCustodyTransfer) {
		t.Fatalf("err = %v, want ErrInvalidCustodyTransfer", err)
	}
	if p.Epoch != 0 {
		t.Fatalf("epoch = %d, want 0", p.Epoch)
	}
}
