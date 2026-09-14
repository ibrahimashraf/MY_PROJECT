package domain

import (
	"errors"
	"fmt"
	"time"
)

// CFIHOS (ISO 18101) classifiers used to map AAS submodel data onto enterprise
// maintenance taxonomies (SAP PM, IBM Maximo).
const (
	CFIHOSManufacturer       = "CFIHOS.EQUIPMENT.MANUFACTURER"
	CFIHOSModel              = "CFIHOS.EQUIPMENT.MODEL"
	CFIHOSSerial             = "CFIHOS.EQUIPMENT.SERIAL_NUMBER"
	CFIHOSRatedSWL           = "CFIHOS.EQUIPMENT.RATED_SWL"
	CFIHOSCategory           = "CFIHOS.EQUIPMENT.CATEGORY"
	CFIHOSJurisdiction       = "CFIHOS.OPERATION.JURISDICTION_ISO"
	CFIHOSOperationalStatus  = "CFIHOS.OPERATION.STATUS"
	CFIHOSCustodyTenant      = "CFIHOS.OPERATION.CUSTODY_TENANT"
	CFIHOSEpoch              = "CFIHOS.OPERATION.EPOCH"
	CFIHOSMountedTag         = "CFIHOS.INSTALLATION.MOUNTED_TAG"
	CFIHOSFunctionalLocation = "CFIHOS.FUNCTIONAL_LOCATION"
	CFIHOSInstalledEquipment = "CFIHOS.EQUIPMENT.INSTALLED"
)

var (
	ErrUnsupportedRegulatoryStandard = errors.New("unsupported statutory regulatory standard")
	ErrNilShell                      = errors.New("asset administration shell cannot be nil")
	ErrEmptySiteID                   = errors.New("site id cannot be empty")
)

// SAPPMExport is the SAP Plant Maintenance representation: Equipment Master
// (EQUI), Functional Location (IFLOT), and Maintenance Characteristic values
// (AUSP).
type SAPPMExport struct {
	EquipmentRecords     []EquipmentMasterRecord    `json:"equipment_records"`
	FunctionalLocations  []FunctionalLocationRecord `json:"functional_locations"`
	CharacteristicValues []CharacteristicRecord     `json:"characteristic_values"`
}

// EquipmentMasterRecord maps one installed-equipment shell onto the SAP PM
// EQUI equipment master table.
type EquipmentMasterRecord struct {
	EquipmentNumber   string      `json:"equipment_number"`
	Manufacturer      string      `json:"manufacturer"`
	ModelNumber       string      `json:"model_number"`
	SerialNumber      string      `json:"serial_number"`
	EquipmentCategory string      `json:"equipment_category"`
	RatedSWL          string      `json:"rated_swl"`
	MountedTagID      string      `json:"mounted_tag_id"`
	AssetDID          string      `json:"asset_did"`
	JurisdictionISO   string      `json:"jurisdiction_iso"`
	Status            AssetStatus `json:"status"`
}

// FunctionalLocationRecord maps one functional-location shell onto the SAP PM
// IFLOT functional location table.
type FunctionalLocationRecord struct {
	FunctionalLocation string `json:"functional_location"`
	Description        string `json:"description"`
	HierarchyPath      string `json:"hierarchy_path"`
	FacilityID         string `json:"facility_id"`
	MountedSerial      string `json:"mounted_serial"`
}

// CharacteristicRecord maps one maintenance characteristic value onto the SAP
// PM AUSP characteristic table, tagged with its CFIHOS classifier.
type CharacteristicRecord struct {
	CharacteristicName  string `json:"characteristic_name"`
	CharacteristicValue string `json:"characteristic_value"`
	ObjectID            string `json:"object_id"`
	CFIHOSClassifier    string `json:"cfihos_classifier"`
}

// ExportSAP_PM formats the shell as an SAP Plant Maintenance export (EQUI,
// IFLOT, AUSP). Equipment shells produce an EQUI record plus AUSP
// characteristic rows; functional-location shells produce an IFLOT record.
func ExportSAP_PM(shell *AssetAdministrationShell) (SAPPMExport, error) {
	if err := shell.Validate(); err != nil {
		return SAPPMExport{}, err
	}

	var out SAPPMExport
	switch shell.Class {
	case ClassFunctionalLocation:
		loc := shell.Location
		out.FunctionalLocations = []FunctionalLocationRecord{{
			FunctionalLocation: loc.TagID,
			Description:        loc.Description,
			HierarchyPath:      loc.HierarchyPath,
			FacilityID:         loc.FacilityID,
			MountedSerial:      loc.MountedSerial,
		}}
	case ClassInstalledEquipment:
		eq := shell.Equipment
		tech := shell.Submodels.TechnicalSpecification
		ops := shell.Submodels.OperationalState
		out.EquipmentRecords = []EquipmentMasterRecord{{
			EquipmentNumber:   eq.SerialNumber,
			Manufacturer:      tech.OEM,
			ModelNumber:       tech.Model,
			SerialNumber:      eq.SerialNumber,
			EquipmentCategory: eq.Passport.EquipmentCategory,
			RatedSWL:          tech.RatedSWL,
			MountedTagID:      eq.MountedTagID,
			AssetDID:          eq.Passport.AssetDID,
			JurisdictionISO:   ops.JurisdictionISOCode,
			Status:            ops.CurrentStatus,
		}}
		out.CharacteristicValues = characteristicRows(eq.SerialNumber, tech, ops, eq)
	default:
		return SAPPMExport{}, ErrInvalidAASClass
	}
	return out, nil
}

func characteristicRows(obj string, tech TechnicalSpecification, ops OperationalState, eq *InstalledEquipment) []CharacteristicRecord {
	rows := []CharacteristicRecord{
		{CharacteristicName: "MANUFACTURER", CharacteristicValue: tech.OEM, ObjectID: obj, CFIHOSClassifier: CFIHOSManufacturer},
		{CharacteristicName: "MODEL", CharacteristicValue: tech.Model, ObjectID: obj, CFIHOSClassifier: CFIHOSModel},
		{CharacteristicName: "SERIAL_NUMBER", CharacteristicValue: eq.SerialNumber, ObjectID: obj, CFIHOSClassifier: CFIHOSSerial},
		{CharacteristicName: "RATED_SWL", CharacteristicValue: tech.RatedSWL, ObjectID: obj, CFIHOSClassifier: CFIHOSRatedSWL},
		{CharacteristicName: "EQUIPMENT_CATEGORY", CharacteristicValue: eq.Passport.EquipmentCategory, ObjectID: obj, CFIHOSClassifier: CFIHOSCategory},
		{CharacteristicName: "JURISDICTION_ISO", CharacteristicValue: ops.JurisdictionISOCode, ObjectID: obj, CFIHOSClassifier: CFIHOSJurisdiction},
		{CharacteristicName: "STATUS", CharacteristicValue: string(ops.CurrentStatus), ObjectID: obj, CFIHOSClassifier: CFIHOSOperationalStatus},
		{CharacteristicName: "CUSTODY_TENANT", CharacteristicValue: ops.CustodyTenantID, ObjectID: obj, CFIHOSClassifier: CFIHOSCustodyTenant},
		{CharacteristicName: "EPOCH", CharacteristicValue: fmt.Sprintf("%d", ops.Epoch), ObjectID: obj, CFIHOSClassifier: CFIHOSEpoch},
	}
	if eq.MountedTagID != "" {
		rows = append(rows, CharacteristicRecord{
			CharacteristicName: "MOUNTED_TAG", CharacteristicValue: eq.MountedTagID, ObjectID: obj, CFIHOSClassifier: CFIHOSMountedTag,
		})
	}
	return rows
}

// MaximoAssetExport is the IBM Maximo representation using classic MBO schemas:
// the ASSET MBO and the LOCATIONS MBO, classified per CFIHOS.
type MaximoAssetExport struct {
	Assets    []MaximoAsset    `json:"assets"`
	Locations []MaximoLocation `json:"locations"`
}

// MaximoAsset maps an installed-equipment shell onto the Maximo ASSET MBO.
type MaximoAsset struct {
	AssetNum         string `json:"ASSETNUM"`
	Description      string `json:"DESCRIPTION"`
	SerialNum        string `json:"SERIALNUM"`
	Status           string `json:"STATUS"`
	SiteID           string `json:"SITEID"`
	AssetTag         string `json:"ASSETTAG"`
	ClassStructureID string `json:"CLASSSTRUCTUREID"`
	Classification   string `json:"CFIHOS_CLASSIFICATION"`
}

// MaximoLocation maps a functional-location shell onto the Maximo LOCATIONS MBO.
type MaximoLocation struct {
	Location       string `json:"LOCATION"`
	Description    string `json:"DESCRIPTION"`
	SiteID         string `json:"SITEID"`
	Status         string `json:"STATUS"`
	Classification string `json:"CFIHOS_CLASSIFICATION"`
}

// ExportMaximo formats the shell for IBM Maximo as ASSET (equipment) or
// LOCATIONS (functional location) MBO records with CFIHOS classification.
// The siteID identifies the target Maximo organizational site/facility and must
// be explicitly supplied from tenant/site context.
func ExportMaximo(shell *AssetAdministrationShell, siteID string) (MaximoAssetExport, error) {
	if err := shell.Validate(); err != nil {
		return MaximoAssetExport{}, err
	}
	if siteID == "" {
		// If functional location provides FacilityID, use it as fallback site identifier
		if shell.Class == ClassFunctionalLocation && shell.Location != nil && shell.Location.FacilityID != "" {
			siteID = shell.Location.FacilityID
		} else {
			return MaximoAssetExport{}, ErrEmptySiteID
		}
	}

	var out MaximoAssetExport
	switch shell.Class {
	case ClassFunctionalLocation:
		loc := shell.Location
		out.Locations = []MaximoLocation{{
			Location:       loc.TagID,
			Description:    loc.Description,
			SiteID:         siteID,
			Status:         "OPERATING",
			Classification: CFIHOSFunctionalLocation,
		}}
	case ClassInstalledEquipment:
		eq := shell.Equipment
		tech := shell.Submodels.TechnicalSpecification
		status := string(AssetStatusOperational)
		if ops := shell.Submodels.OperationalState.CurrentStatus; ops != "" {
			status = string(ops)
		}
		out.Assets = []MaximoAsset{{
			AssetNum:         eq.SerialNumber,
			Description:      eq.Description,
			SerialNum:        tech.SerialNumber,
			Status:           status,
			SiteID:           siteID,
			AssetTag:         eq.MountedTagID,
			ClassStructureID: "INTEGER-PM",
			Classification:   CFIHOSInstalledEquipment,
		}}
	default:
		return MaximoAssetExport{}, ErrInvalidAASClass
	}
	return out, nil
}

// RegulatoryDossier is the automated statutory compliance dossier for a single
// regulatory standard.
type RegulatoryDossier struct {
	Standard    string            `json:"standard"`
	AssetID     string            `json:"asset_id"`
	Status      string            `json:"status"` // COMPLIANT | NON_COMPLIANT
	Statements  []string          `json:"statements"`
	Checks      []RegulatoryCheck `json:"checks"`
	GeneratedAt time.Time         `json:"generated_at"`
}

// RegulatoryCheck itemizes one statutory requirement and its outcome.
type RegulatoryCheck struct {
	Requirement string `json:"requirement"`
	Status      string `json:"status"` // PASS | FAIL
	Detail      string `json:"detail"`
}

// regulatoryRequirement describes the evidence an asset must hold under a
// statutory standard. Thresholds are deterministic set-membership assertions —
// zero calculated values (Hazard 7).
type regulatoryRequirement struct {
	certificatesRequired bool
	ndtRequired          bool
	proofLoadRequired    bool
}

var regulatoryStandards = map[string]struct {
	name        string
	citation    string
	requirement regulatoryRequirement
}{
	"OSHA_1910_180": {
		name: "OSHA 29 CFR 1910.180", citation: "Crawler, locomotive and truck cranes",
		requirement: regulatoryRequirement{certificatesRequired: true, ndtRequired: true, proofLoadRequired: true},
	},
	"LOLER_1998": {
		name: "LOLER 1998", citation: "Lifting Operations and Lifting Equipment Regulations",
		requirement: regulatoryRequirement{certificatesRequired: true, ndtRequired: true, proofLoadRequired: true},
	},
	"DNV_ST_N001": {
		name: "DNV-ST-N001", citation: "Marine operations and heavy lift",
		requirement: regulatoryRequirement{certificatesRequired: true, ndtRequired: true, proofLoadRequired: true},
	},
	"ARAMCO_GI_7_027": {
		name: "Saudi Aramco GI 7.027", citation: "Crane and rigging safety",
		requirement: regulatoryRequirement{certificatesRequired: true, ndtRequired: true, proofLoadRequired: true},
	},
	"ADNOC_COP_HSE_038": {
		name: "ADNOC CoP-HSE-038", citation: "Lifting operations code of practice",
		requirement: regulatoryRequirement{certificatesRequired: true, ndtRequired: true, proofLoadRequired: true},
	},
}

// GenerateRegulatoryDossier builds an automated statutory compliance dossier
// for the asset under one of the supported standards. A certificate, an NDT
// report, a passed proof-load test, and a released operational status are
// required; any missing evidence fails the dossier closed.
func GenerateRegulatoryDossier(shell *AssetAdministrationShell, standard string) (RegulatoryDossier, error) {
	if shell == nil {
		return RegulatoryDossier{}, ErrNilShell
	}
	if err := shell.Validate(); err != nil {
		return RegulatoryDossier{}, err
	}
	std, ok := regulatoryStandards[standard]
	if !ok {
		return RegulatoryDossier{}, fmt.Errorf("%w: %s", ErrUnsupportedRegulatoryStandard, standard)
	}

	evidence := shell.Submodels.AssuranceEvidence
	tech := shell.Submodels.TechnicalSpecification

	checks := []RegulatoryCheck{
		{
			Requirement: "Manufacturer identification",
			Status:      passFail(tech.OEM != "" && tech.SerialNumber != ""),
			Detail:      fmt.Sprintf("OEM %q serial %q", tech.OEM, tech.SerialNumber),
		},
	}
	checks = append(checks, RegulatoryCheck{
		Requirement: "Statutory certificate on file",
		Status:      passFail(std.requirement.certificatesRequired && len(evidence.SealedCertificates) > 0),
		Detail:      fmt.Sprintf("%d sealed certificate(s)", len(evidence.SealedCertificates)),
	})
	checks = append(checks, RegulatoryCheck{
		Requirement: "NDT report on file",
		Status:      passFail(std.requirement.ndtRequired && len(evidence.NDTReports) > 0),
		Detail:      fmt.Sprintf("%d NDT report(s)", len(evidence.NDTReports)),
	})
	checks = append(checks, RegulatoryCheck{
		Requirement: "Proof load test passed",
		Status:      passFail(std.requirement.proofLoadRequired && hasPassedProofLoad(evidence.ProofLoadTests)),
		Detail:      fmt.Sprintf("%d proof load record(s)", len(evidence.ProofLoadTests)),
	})
	checks = append(checks, RegulatoryCheck{
		Requirement: "Asset released for lifting",
		Status:      passFail(shell.Submodels.OperationalState.CurrentStatus == AssetStatusOperational),
		Detail:      string(shell.Submodels.OperationalState.CurrentStatus),
	})

	status := "COMPLIANT"
	statements := make([]string, 0, len(checks))
	for _, c := range checks {
		statements = append(statements, fmt.Sprintf("%s (%s): %s → %s", std.name, std.citation, c.Detail, c.Status))
		if c.Status == "FAIL" {
			status = "NON_COMPLIANT"
		}
	}

	return RegulatoryDossier{
		Standard:    standard,
		AssetID:     shell.ID,
		Status:      status,
		Statements:  statements,
		Checks:      checks,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func passFail(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

func hasPassedProofLoad(records []ProofLoadRecord) bool {
	for _, r := range records {
		if r.Result == "PASS" {
			return true
		}
	}
	return false
}
