package domain

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

// evidencedEquipmentShell builds a fully-populated installed-equipment shell
// with all three sovereign submodels exercised.
func evidencedEquipmentShell() *AssetAdministrationShell {
	eq := equipmentShellA()
	eq.Submodels.TechnicalSpecification = TechnicalSpecification{
		OEM:                 "Liebherr",
		Model:               "LTM 1500",
		SerialNumber:        "SER-LIEBHERR-99401",
		RatedSWL:            "500 Ton",
		MaterialHeatNumbers: []string{"HEAT-4451-A"},
		MillTestCertificates: []DocumentRef{
			{DocumentID: "MTC-001", Title: "Mill test cert", SealedAt: machineTime(), SealedBy: "steelwork", Hash: "abc123"},
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
			{DocumentID: "CERT-TOR-100", Title: "annual certificate", SealedAt: tagTime(), SealedBy: "authority", Hash: "fff000"},
		},
		NDTReports: []DocumentRef{
			{DocumentID: "NDT-UT-77", Title: "Ultrasonic report boom", SealedAt: tagTime(), SealedBy: "ndt-lab", Hash: "111222"},
		},
		ProofLoadTests: []ProofLoadRecord{
			{TestID: "PLT-3001", PerformedAt: machineTime(), LoadTonnes: 600, Result: "PASS"},
		},
	}
	return eq
}

func TestAASXPackageRoundTrip(t *testing.T) {
	shell := evidencedEquipmentShell()
	if err := shell.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	var buf bytes.Buffer
	if err := PackageAASX(shell, &buf); err != nil {
		t.Fatalf("package: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("package emitted zero bytes")
	}

	back, err := UnpackAASX(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	if err := back.Validate(); err != nil {
		t.Fatalf("unpacked shell invalid: %v", err)
	}

	want, err := json.Marshal(shell)
	if err != nil {
		t.Fatalf("marshal original: %v", err)
	}
	got, err := json.Marshal(back)
	if err != nil {
		t.Fatalf("marshal unpacked: %v", err)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("round-trip shell mismatch:\nwant %s\ngot  %s", want, got)
	}
}

func TestAASXPackageStructure(t *testing.T) {
	var buf bytes.Buffer
	if err := PackageAASX(evidencedEquipmentShell(), &buf); err != nil {
		t.Fatalf("package: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}

	names := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		names[f.Name] = f
	}
	for _, want := range []string{aasxContentTypesPart, aasxRelsPart, aasxAASEnvPart} {
		if names[want] == nil {
			t.Fatalf("part %q missing from aasx package", want)
		}
	}

	ct := names[aasxContentTypesPart]
	rc, err := ct.Open()
	if err != nil {
		t.Fatalf("open content types: %v", err)
	}
	body, readErr := io.ReadAll(rc)
	rc.Close()
	if readErr != nil {
		t.Fatalf("read content types: %v", readErr)
	}
	if !strings.Contains(string(body), aasxContentTypeJSON) {
		t.Fatalf("content types missing %s: %s", aasxContentTypeJSON, body)
	}
}

func TestAASXSemanticIDs(t *testing.T) {
	shell := equipmentShellA()
	env, err := buildEnvironment(shell)
	if err != nil {
		t.Fatalf("build environment: %v", err)
	}
	if len(env.AssetAdministrationShells) != 1 {
		t.Fatalf("expected 1 shell record, got %d", len(env.AssetAdministrationShells))
	}
	rec := env.AssetAdministrationShells[0]
	if rec.SemanticID != eclassIDInstalledEquipment {
		t.Fatalf("shell semantic id = %q, want %q", rec.SemanticID, eclassIDInstalledEquipment)
	}
	if len(rec.Submodels) != 3 {
		t.Fatalf("expected 3 submodel records, got %d", len(rec.Submodels))
	}
	wantSubmodels := map[string]string{
		"technical_specification": SemanticIDTechnicalSpecification,
		"operational_state":       eclassIDOperationalState,
		"assurance_evidence":      eclassIDAssuranceEvidence,
	}
	for _, sm := range rec.Submodels {
		if want := wantSubmodels[sm.Path]; want == "" {
			t.Fatalf("unexpected submodel path %q", sm.Path)
		} else if sm.SemanticID != want {
			t.Fatalf("%s semantic id = %q, want %q", sm.Path, sm.SemanticID, want)
		}
	}
	proofLoad := SemIDsSimHasElement(rec.Submodels, "ProofLoadTests")
	if proofLoad != SemanticIDProofLoadTest {
		t.Fatalf("proof load element semantic id = %q, want %q", proofLoad, SemanticIDProofLoadTest)
	}
}

func SemIDsSimHasElement(submodels []aasxSubmodelRecord, idShort string) string {
	for _, sm := range submodels {
		for _, el := range sm.Elements {
			if el.IDShort == idShort {
				return el.SemanticID
			}
		}
	}
	return ""
}

func TestUnpackAASXCorruptZip(t *testing.T) {
	data := []byte("this is not a zip archive")
	_, err := UnpackAASX(bytes.NewReader(data), int64(len(data)))
	if err == nil {
		t.Fatal("expected error for corrupt archive")
	}
	if !errors.Is(err, ErrAASXInvalidPackage) {
		t.Fatalf("err = %v, want %v", err, ErrAASXInvalidPackage)
	}
}

func TestPackageAASXNilShell(t *testing.T) {
	var buf bytes.Buffer
	if err := PackageAASX(nil, &buf); !errors.Is(err, ErrAASXNoShell) {
		t.Fatalf("err = %v, want %v", err, ErrAASXNoShell)
	}
	if err := PackageAASX(equipmentShellA(), nil); err == nil {
		t.Fatal("expected error for nil writer")
	}
}
