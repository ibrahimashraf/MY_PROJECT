package domain

import (
	"errors"
	"testing"
)

func TestExportSAP_PMEquipment(t *testing.T) {
	shell := evidencedEquipmentShell()
	out, err := ExportSAP_PM(shell)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(out.EquipmentRecords) != 1 {
		t.Fatalf("expected 1 EQUI record, got %d", len(out.EquipmentRecords))
	}
	rec := out.EquipmentRecords[0]
	if rec.EquipmentNumber != "SER-LIEBHERR-99401" || rec.Manufacturer != "Liebherr" {
		t.Fatalf("EQUI record = %+v", rec)
	}
	if rec.Status != AssetStatusOperational || rec.JurisdictionISO != "SA" {
		t.Fatalf("EQUI operational fields = %+v", rec)
	}
	if len(out.FunctionalLocations) != 0 {
		t.Fatalf("equipment export must not emit IFLOT rows, got %d", len(out.FunctionalLocations))
	}

	wantRows := 9
	if len(out.CharacteristicValues) != wantRows {
		t.Fatalf("expected %d AUSP rows, got %d", wantRows, len(out.CharacteristicValues))
	}
	sawSWL := false
	for _, row := range out.CharacteristicValues {
		if row.CharacteristicName == "RATED_SWL" {
			sawSWL = true
			if row.CharacteristicValue != "500 Ton" || row.CFIHOSClassifier != CFIHOSRatedSWL {
				t.Fatalf("SWL row = %+v", row)
			}
		}
	}
	if !sawSWL {
		t.Fatal("RATED_SWL AUSP row missing")
	}
}

func TestExportSAP_PMFunctionalLocation(t *testing.T) {
	shell := tagShellA()
	out, err := ExportSAP_PM(shell)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(out.FunctionalLocations) != 1 {
		t.Fatalf("expected 1 IFLOT record, got %d", len(out.FunctionalLocations))
	}
	rec := out.FunctionalLocations[0]
	if rec.FunctionalLocation != "TAG-RIG01-CRANE-A" || rec.FacilityID != "FAC-RIG01" {
		t.Fatalf("IFLOT record = %+v", rec)
	}
	if len(out.EquipmentRecords) != 0 || len(out.CharacteristicValues) != 0 {
		t.Fatalf("functional location export must not emit EQUI/AUSP rows")
	}
}

func TestExportSAP_PMInvalidClass(t *testing.T) {
	shell := &AssetAdministrationShell{ID: "aas-x", Class: AASItemClass("MAGIC")}
	_, err := ExportSAP_PM(shell)
	if !errors.Is(err, ErrInvalidAASClass) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidAASClass)
	}
}

func TestExportMaximoEquipment(t *testing.T) {
	shell := evidencedEquipmentShell()
	testSite := "SITE-YANBU-01"
	out, err := ExportMaximo(shell, testSite)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(out.Assets) != 1 {
		t.Fatalf("expected 1 ASSET MBO, got %d", len(out.Assets))
	}
	a := out.Assets[0]
	if a.AssetNum != "SER-LIEBHERR-99401" || a.SerialNum != "SER-LIEBHERR-99401" {
		t.Fatalf("ASSET MBO = %+v", a)
	}
	if a.SiteID != testSite || a.Classification != CFIHOSInstalledEquipment {
		t.Fatalf("ASSET MBO site/classification = %+v", a)
	}
	if a.Status != string(AssetStatusOperational) {
		t.Fatalf("ASSET MBO status = %q", a.Status)
	}
	if len(out.Locations) != 0 {
		t.Fatalf("equipment export must not emit LOCATIONS MBO")
	}

	// Verify empty siteID error
	_, err = ExportMaximo(shell, "")
	if !errors.Is(err, ErrEmptySiteID) {
		t.Fatalf("expected ErrEmptySiteID for empty site, got %v", err)
	}
}

func TestExportMaximoFunctionalLocation(t *testing.T) {
	shell := tagShellA()
	// Functional location with fallback to shell.Location.FacilityID ("FAC-RIG01")
	out, err := ExportMaximo(shell, "")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(out.Locations) != 1 {
		t.Fatalf("expected 1 LOCATIONS MBO, got %d", len(out.Locations))
	}
	loc := out.Locations[0]
	if loc.Location != "TAG-RIG01-CRANE-A" || loc.Classification != CFIHOSFunctionalLocation {
		t.Fatalf("LOCATIONS MBO = %+v", loc)
	}
	if loc.SiteID != "FAC-RIG01" {
		t.Fatalf("expected SiteID FAC-RIG01, got %q", loc.SiteID)
	}
	if len(out.Assets) != 0 {
		t.Fatalf("functional location export must not emit ASSET MBO")
	}

	// Functional location with explicit siteID override
	outExplicit, err := ExportMaximo(shell, "SITE-OFFSHORE-02")
	if err != nil {
		t.Fatalf("export explicit: %v", err)
	}
	if outExplicit.Locations[0].SiteID != "SITE-OFFSHORE-02" {
		t.Fatalf("expected SiteID SITE-OFFSHORE-02, got %q", outExplicit.Locations[0].SiteID)
	}
}

func TestGenerateRegulatoryDossierAllStandards(t *testing.T) {
	standards := []string{"OSHA_1910_180", "LOLER_1998", "DNV_ST_N001", "ARAMCO_GI_7_027", "ADNOC_COP_HSE_038"}
	for _, std := range standards {
		t.Run(std, func(t *testing.T) {
			dossier, err := GenerateRegulatoryDossier(evidencedEquipmentShell(), std)
			if err != nil {
				t.Fatalf("dossier %s: %v", std, err)
			}
			if dossier.Status != "COMPLIANT" {
				t.Fatalf("dossier %s status = %q, want COMPLIANT: %+v", std, dossier.Status, dossier.Checks)
			}
			if len(dossier.Statements) == 0 || len(dossier.Checks) == 0 {
				t.Fatalf("dossier %s emitted no statements/checks", std)
			}
			for _, c := range dossier.Checks {
				if c.Status != "PASS" {
					t.Fatalf("dossier %s check %q = %s, want PASS", std, c.Requirement, c.Status)
				}
			}
		})
	}
}

func TestGenerateRegulatoryDossierMissingEvidence(t *testing.T) {
	shell := equipmentShellA()
	shell.Submodels.OperationalState.CurrentStatus = AssetStatusOperational
	dossier, err := GenerateRegulatoryDossier(shell, "LOLER_1998")
	if err != nil {
		t.Fatalf("dossier: %v", err)
	}
	if dossier.Status != "NON_COMPLIANT" {
		t.Fatalf("status = %q, want NON_COMPLIANT: %+v", dossier.Status, dossier.Checks)
	}
	statusByReq := map[string]string{}
	for _, c := range dossier.Checks {
		statusByReq[c.Requirement] = c.Status
	}
	for _, req := range []string{"Statutory certificate on file", "NDT report on file", "Proof load test passed"} {
		if statusByReq[req] != "FAIL" {
			t.Fatalf("check %q = %q, want FAIL", req, statusByReq[req])
		}
	}
}

func TestGenerateRegulatoryDossierQuarantinedFails(t *testing.T) {
	shell := evidencedEquipmentShell()
	shell.Submodels.OperationalState.CurrentStatus = AssetStatusQuarantined
	dossier, err := GenerateRegulatoryDossier(shell, "ADNOC_COP_HSE_038")
	if err != nil {
		t.Fatalf("dossier: %v", err)
	}
	if dossier.Status != "NON_COMPLIANT" {
		t.Fatalf("status = %q, want NON_COMPLIANT", dossier.Status)
	}
	for _, c := range dossier.Checks {
		if c.Requirement == "Asset released for lifting" && c.Status != "FAIL" {
			t.Fatalf("check %q = %s, want FAIL", c.Requirement, c.Status)
		}
	}
}

func TestGenerateRegulatoryDossierUnknownStandard(t *testing.T) {
	_, err := GenerateRegulatoryDossier(evidencedEquipmentShell(), "NOMINAL_STD_X")
	if !errors.Is(err, ErrUnsupportedRegulatoryStandard) {
		t.Fatalf("err = %v, want %v", err, ErrUnsupportedRegulatoryStandard)
	}
}

func TestGenerateRegulatoryDossierNilShell(t *testing.T) {
	_, err := GenerateRegulatoryDossier(nil, "OSHA_1910_180")
	if !errors.Is(err, ErrNilShell) {
		t.Fatalf("err = %v, want %v", err, ErrNilShell)
	}
}
