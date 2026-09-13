package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"integin/pkg/id"
)

func tagTime() time.Time     { return time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC) }
func machineTime() time.Time { return time.Date(2026, 8, 1, 8, 30, 0, 0, time.UTC) }

func tagShellA() *AssetAdministrationShell {
	return NewFunctionalLocationAAS("aas-tag-rig01-crane-a", FunctionalLocation{
		TagID:         "TAG-RIG01-CRANE-A",
		FacilityID:    "FAC-RIG01",
		HierarchyPath: "RIG01.CRANE-A",
		Description:   "Crane pad on rig 01",
	})
}

func equipmentShellA() *AssetAdministrationShell {
	p := fencedPassport()
	p.AssetDID = "did:integin:asset:9940120ac"
	p.Manufacturer = "Liebherr"
	p.ChassisSerial = "SER-LIEBHERR-99401"
	p.ModelNumber = "LTM 1500"
	p.RatedSWL = "500 Ton"
	p.CurrentStatus = AssetStatusOperational
	p.JurisdictionCode = "SA"
	p.RegisteredAt = machineTime()
	return NewInstalledEquipmentAAS("aas-eq-liebherr-99401", InstalledEquipment{
		SerialNumber: "SER-LIEBHERR-99401",
		Description:  "Liebherr LTM 1500 mobile crane",
		Passport:     *p,
	})
}

func TestAASSerializationRoundTrip(t *testing.T) {
	eq := equipmentShellA()
	eq.Submodels.TechnicalSpecification = TechnicalSpecification{
		OEM:                 "Liebherr",
		Model:               "LTM 1500",
		SerialNumber:        "SER-LIEBHERR-99401",
		RatedSWL:            "500 Ton",
		MaterialHeatNumbers: []string{"HEAT-4451-A", "HEAT-4451-B"},
		MillTestCertificates: []DocumentRef{
			{DocumentID: "MTC-001", Title: "Mill test cert", SealedAt: machineTime(), SealedBy: "steelwork", Hash: "abc123"},
		},
		CapacityCharts: []DocumentRef{
			{DocumentID: "CH-991", Title: "Rated capacity chart LTM 1500", SealedAt: machineTime(), SealedBy: "oem", Hash: "def456"},
		},
	}
	eq.Submodels.OperationalState = OperationalState{
		CustodyTenantID:        "tenant-saudi",
		JurisdictionISOCode:    "SA",
		AccumulativeLoadCycles: 42,
		CurrentStatus:          AssetStatusOperational,
		Epoch:                  0,
		LastUpdatedAt:          tagTime(),
	}
	eq.Submodels.AssuranceEvidence = AssuranceEvidence{
		SealedCertificates: []DocumentRef{
			{DocumentID: "CERT-TOR-100", Title: "TÜV annual certificate", SealedAt: tagTime(), SealedBy: "authority", Hash: "fff000"},
		},
		NDTReports: []DocumentRef{
			{DocumentID: "NDT-UT-77", Title: "Ultrasonic report boom", SealedAt: tagTime(), SealedBy: "ndt-lab", Hash: "111222"},
		},
		ProofLoadTests: []ProofLoadRecord{
			{TestID: "PLT-3001", PerformedAt: machineTime(), LoadTonnes: 600, Result: "PASS"},
		},
	}
	if err := eq.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	first, err := json.Marshal(eq)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back AssetAdministrationShell
	if err := json.Unmarshal(first, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	second, err := json.Marshal(&back)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("round-trip unstable:\nfirst  %s\nsecond %s", first, second)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(first, &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}
	for _, key := range []string{"technical_specification", "operational_state", "assurance_evidence"} {
		if doc["submodels"].(map[string]interface{})[key] == nil {
			t.Fatalf("submodel %q missing from serialized AAS", key)
		}
	}
	if doc["class"] != string(ClassInstalledEquipment) {
		t.Fatalf("class = %v, want %s", doc["class"], ClassInstalledEquipment)
	}
}

func TestAASIdentityValidation(t *testing.T) {
	t.Run("empty id rejected", func(t *testing.T) {
		s := NewInstalledEquipmentAAS("", InstalledEquipment{SerialNumber: "SER-1"})
		if err := s.Validate(); !errors.Is(err, ErrEmptyAASID) {
			t.Fatalf("err = %v, want %v", err, ErrEmptyAASID)
		}
	})

	t.Run("tag shell requires location", func(t *testing.T) {
		s := &AssetAdministrationShell{ID: "aas-tag", Class: ClassFunctionalLocation}
		if err := s.Validate(); !errors.Is(err, ErrAASNoFunctionalLocation) {
			t.Fatalf("err = %v, want %v", err, ErrAASNoFunctionalLocation)
		}
	})

	t.Run("equipment shell requires passport identity", func(t *testing.T) {
		s := NewInstalledEquipmentAAS("aas-eq", InstalledEquipment{SerialNumber: "SER-1"})
		if err := s.Validate(); !errors.Is(err, ErrEmptyAssetDID) {
			t.Fatalf("err = %v, want %v", err, ErrEmptyAssetDID)
		}
	})

	t.Run("invalid class rejected", func(t *testing.T) {
		s := &AssetAdministrationShell{ID: "aas-x", Class: AASItemClass("MAGIC")}
		if err := s.Validate(); !errors.Is(err, ErrInvalidAASClass) {
			t.Fatalf("err = %v, want %v", err, ErrInvalidAASClass)
		}
	})
}

func TestMountEquipmentBindsBothSides(t *testing.T) {
	tag := tagShellA()
	eq := equipmentShellA()

	if err := tag.MountEquipment(eq, "supervisor-1"); err != nil {
		t.Fatalf("mount: %v", err)
	}
	if tag.Location.MountedSerial != "SER-LIEBHERR-99401" {
		t.Fatalf("tag mounted serial = %q", tag.Location.MountedSerial)
	}
	if eq.Equipment.MountedTagID != tag.ID {
		t.Fatalf("equipment mounted tag = %q, want %q", eq.Equipment.MountedTagID, tag.ID)
	}
	if len(tag.MountHistory) != 1 || len(eq.MountHistory) != 1 {
		t.Fatalf("mount history lens tag=%d eq=%d, want 1/1", len(tag.MountHistory), len(eq.MountHistory))
	}
	ev := tag.MountHistory[0]
	if ev.Action != MountActionMount || ev.EquipmentSerial != "SER-LIEBHERR-99401" {
		t.Fatalf("mount event = %+v", ev)
	}
	if !id.IsValidV7(ev.EventID) {
		t.Fatalf("mount event id %q is not a valid UUIDv7", ev.EventID)
	}
	if ev != eq.MountHistory[0] {
		t.Fatal("the same mount event must be recorded on both independent histories")
	}
	if err := tag.Validate(); err != nil {
		t.Fatalf("tag validate after mount: %v", err)
	}
	if err := eq.Validate(); err != nil {
		t.Fatalf("eq validate after mount: %v", err)
	}
}

func TestMountEquipmentRejectsOccupiedTag(t *testing.T) {
	tag := tagShellA()
	eq := equipmentShellA()
	eq2 := equipmentShellA()
	eq2.Equipment.SerialNumber = "SER-LIEBHERR-99402"
	if err := tag.MountEquipment(eq, "supervisor-1"); err != nil {
		t.Fatalf("first mount: %v", err)
	}
	if err := tag.MountEquipment(eq2, "supervisor-1"); !errors.Is(err, ErrTagOccupied) {
		t.Fatalf("err = %v, want %v", err, ErrTagOccupied)
	}
	if tag.Location.MountedSerial != "SER-LIEBHERR-99401" {
		t.Fatalf("binding mutated on rejected mount: %q", tag.Location.MountedSerial)
	}
}

func TestMountEquipmentRejectsDoubleMountedMachine(t *testing.T) {
	tag := tagShellA()
	tag2 := NewFunctionalLocationAAS("aas-tag-rig02-crane-b", FunctionalLocation{TagID: "TAG-RIG02-CRANE-B"})
	eq := equipmentShellA()
	if err := tag.MountEquipment(eq, "supervisor-1"); err != nil {
		t.Fatalf("first mount: %v", err)
	}
	if err := tag2.MountEquipment(eq, "supervisor-2"); !errors.Is(err, ErrEquipmentAlreadyMounted) {
		t.Fatalf("err = %v, want %v", err, ErrEquipmentAlreadyMounted)
	}
	if eq.Equipment.MountedTagID != tag.ID {
		t.Fatalf("machine binding mutated on rejected mount: %q", eq.Equipment.MountedTagID)
	}
}

func TestMountEquipmentRejectsNonTagShell(t *testing.T) {
	tag := tagShellA()
	eq := equipmentShellA()
	if err := eq.MountEquipment(tag, "supervisor-1"); !errors.Is(err, ErrAASNoFunctionalLocation) {
		t.Fatalf("err = %v, want %v", err, ErrAASNoFunctionalLocation)
	}
	if err := tag.MountEquipment(nil, "supervisor-1"); !errors.Is(err, ErrAASNoInstalledEquipment) {
		t.Fatalf("err = %v, want %v", err, ErrAASNoInstalledEquipment)
	}
}

func TestUnmountEquipmentClearsBindingAndPreservesHistory(t *testing.T) {
	tag := tagShellA()
	eq := equipmentShellA()
	if err := tag.MountEquipment(eq, "supervisor-1"); err != nil {
		t.Fatalf("mount: %v", err)
	}
	if err := tag.UnmountEquipment(eq, "supervisor-1"); err != nil {
		t.Fatalf("unmount: %v", err)
	}
	if tag.Location.MountedSerial != "" {
		t.Fatalf("tag still mounted: %q", tag.Location.MountedSerial)
	}
	if eq.Equipment.MountedTagID != "" {
		t.Fatalf("equipment still bound: %q", eq.Equipment.MountedTagID)
	}
	if len(tag.MountHistory) != 2 || len(eq.MountHistory) != 2 {
		t.Fatalf("history lens tag=%d eq=%d after unmount, want 2/2", len(tag.MountHistory), len(eq.MountHistory))
	}
	// Mount event must persist unchanged in both independent histories.
	if tag.MountHistory[0].Action != MountActionMount || eq.MountHistory[0].Action != MountActionMount {
		t.Fatal("mount event was lost from history")
	}
	if tag.MountHistory[1].Action != MountActionUnmount || eq.MountHistory[1].Action != MountActionUnmount {
		t.Fatalf("latest event = %+v / %+v, want UNMOUNT", tag.MountHistory[1], eq.MountHistory[1])
	}
}

func TestUnmountMismatchedSerialRejected(t *testing.T) {
	tag := tagShellA()
	eq := equipmentShellA()
	if err := tag.MountEquipment(eq, "supervisor-1"); err != nil {
		t.Fatalf("mount: %v", err)
	}
	ring := equipmentShellA()
	ring.Equipment.SerialNumber = "SER-LIEBHERR-99999"
	if err := tag.UnmountEquipment(ring, "supervisor-1"); !errors.Is(err, ErrMountedSerialMismatch) {
		t.Fatalf("err = %v, want %v", err, ErrMountedSerialMismatch)
	}
	if tag.Location.MountedSerial != "SER-LIEBHERR-99401" {
		t.Fatal("binding mutated on rejected unmount")
	}
}

func TestUnmountEmptyTagRejected(t *testing.T) {
	tag := tagShellA()
	eq := equipmentShellA()
	if err := tag.UnmountEquipment(eq, "supervisor-1"); !errors.Is(err, ErrNoMountedEquipment) {
		t.Fatalf("err = %v, want %v", err, ErrNoMountedEquipment)
	}
}

func TestSyncOperationalSubmodelMirrorsPassport(t *testing.T) {
	eq := equipmentShellA()
	if err := eq.SyncOperationalSubmodel(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	s := eq.Submodels.OperationalState
	if s.JurisdictionISOCode != "SA" || s.CurrentStatus != AssetStatusOperational || s.Epoch != 0 {
		t.Fatalf("synced state = %+v", s)
	}
	if err := tagShellA().SyncOperationalSubmodel(); !errors.Is(err, ErrAASNoInstalledEquipment) {
		t.Fatalf("err = %v, want %v", err, ErrAASNoInstalledEquipment)
	}
}

func TestTransferEquipmentCustodyFencedEnforcesEpochMonotonicity(t *testing.T) {
	t.Run("happy path advances epoch and syncs submodel", func(t *testing.T) {
		eq := equipmentShellA()
		if err := eq.TransferEquipmentCustody(fencedTransfer()); err != nil {
			t.Fatalf("transfer: %v", err)
		}
		if eq.Equipment.Passport.Epoch != 1 {
			t.Fatalf("passport epoch = %d, want 1", eq.Equipment.Passport.Epoch)
		}
		s := eq.Submodels.OperationalState
		if s.Epoch != 1 || s.JurisdictionISOCode != "AE" || s.CustodyTenantID != "tenant-uae-offshore" {
			t.Fatalf("operational submodel not synced: %+v", s)
		}
	})

	t.Run("stale replay rejected", func(t *testing.T) {
		eq := equipmentShellA()
		if err := eq.TransferEquipmentCustody(fencedTransfer()); err != nil {
			t.Fatalf("first transfer: %v", err)
		}
		replayed := fencedTransfer() // still expects epoch 0
		if err := eq.TransferEquipmentCustody(replayed); !errors.Is(err, ErrEpochMismatch) {
			t.Fatalf("err = %v, want %v", err, ErrEpochMismatch)
		}
		if eq.Submodels.OperationalState.Epoch != 1 {
			t.Fatalf("submodel epoch moved to %d on rejected replay, want 1", eq.Submodels.OperationalState.Epoch)
		}
	})

	t.Run("split-brain epoch rejected", func(t *testing.T) {
		eq := equipmentShellA()
		sb := fencedTransfer()
		sb.ExpectedEpoch = 7
		if err := eq.TransferEquipmentCustody(sb); !errors.Is(err, ErrEpochMismatch) {
			t.Fatalf("err = %v, want %v", err, ErrEpochMismatch)
		}
	})

	t.Run("tag shell has no custody epoch", func(t *testing.T) {
		if err := tagShellA().TransferEquipmentCustody(fencedTransfer()); !errors.Is(err, ErrAASNoInstalledEquipment) {
			t.Fatalf("err = %v, want %v", err, ErrAASNoInstalledEquipment)
		}
	})
}
