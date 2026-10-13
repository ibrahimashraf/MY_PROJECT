package domain

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Canonical eCl@ss IRDI semantic identifiers bound to the AAS submodels and
// elements. ADN785#003/TechnicalSpecification and ABI502#001/ProofLoadTests are
// the standard IDs; the remaining IRDIs are the project-canonical bindings for
// the sovereign twin submodels.
const (
	SemanticIDTechnicalSpecification = "0173-1#01-ADN785#003" // TechnicalSpecification (equipment)
	SemanticIDProofLoadTest          = "0173-1#02-ABI502#001" // Proof load / load test

	eclassIDOperationalState   = "0173-1#01-ADN497#002"
	eclassIDAssuranceEvidence  = "0173-1#01-ADN433#006"
	eclassIDSealedCertificate  = "0173-1#02-AAH378#003"
	eclassIDNDTReport          = "0173-1#02-ABD516#008"
	eclassIDInstalledEquipment = "0173-1#04-ABE744#008"
	eclassIDFunctionalLocation = "0173-1#01-BAA598#004"
)

// OPC package part paths mandated by IEC 63278 (.aasx).
const (
	aasxContentTypesPart = "[Content_Types].xml"
	aasxRelsPart         = "_rels/.rels"
	aasxAASEnvPart       = "aasx/aasenv.json"

	aasxContentTypeJSON = "application/asset-administration-shell-package+json"
)

var (
	ErrAASXNoShell        = errors.New("aasx package contains no asset administration shell")
	ErrAASXInvalidPackage = errors.New("aasx package is malformed")
	ErrAASXMissingPart    = errors.New("aasx package is missing a required OPC part")
)

// aasxEnvironment is the IEC 63278 AssetAdministrationShellEnvironment envelope
// serialized to aasx/aasenv.json.
type aasxEnvironment struct {
	AssetAdministrationShells []aasxShellRecord `json:"assetAdministrationShells"`
}

// aasxShellRecord binds a fully serialized shell to its eCl@ss semantic ID,
// its submodel/element metadata, and its referenced asset.
type aasxShellRecord struct {
	ID         string                    `json:"id"`
	IDShort    string                    `json:"idShort"`
	SemanticID string                    `json:"semanticId"`
	Submodels  []aasxSubmodelRecord      `json:"submodels"`
	Assets     []aasxAssetRecord         `json:"assets"`
	Shell      *AssetAdministrationShell `json:"shell"`
}

// aasxSubmodelRecord lists one shell submodel with its canonical eCl@ss the
// semantic ID and any element-level IDs (e.g. the ProofLoadTests element).
type aasxSubmodelRecord struct {
	ID         string              `json:"id"`
	IDShort    string              `json:"idShort"`
	SemanticID string              `json:"semanticId"`
	Path       string              `json:"path"`
	Elements   []aasxElementRecord `json:"elements,omitempty"`
}

// aasxElementRecord binds one submodel element (e.g. a proof load test series)
// to its eCl@ss semantic ID.
type aasxElementRecord struct {
	ID         string `json:"id"`
	IDShort    string `json:"idShort"`
	SemanticID string `json:"semanticId"`
}

// aasxAssetRecord references the physical asset addressed by the shell.
type aasxAssetRecord struct {
	ID            string `json:"id"`
	IDShort       string `json:"idShort"`
	GlobalAssetID string `json:"globalAssetId"`
}

// PackageAASX serializes the shell into a standards-compliant IEC 63278 .aasx
// Open Packaging Conventions package (pure archive/zip):
//
//	[Content_Types].xml          OPC content-type declarations
//	_rels/.rels                  package relationship to the AAS environment
//	aasx/aasenv.json             AssetAdministrationShellEnvironment JSON
//
// The environment binds every submodel and the ProofLoadTests element to its
// canonical eCl@ss semantic identifier.
func PackageAASX(shell *AssetAdministrationShell, w io.Writer) error {
	if shell == nil {
		return ErrAASXNoShell
	}
	if err := shell.Validate(); err != nil {
		return err
	}
	if w == nil {
		return fmt.Errorf("aasx output writer cannot be nil")
	}

	env, err := buildEnvironment(shell)
	if err != nil {
		return err
	}
	envJSON, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize aasx environment: %w", err)
	}
	envJSON = append(envJSON, '\n')

	zw := zip.NewWriter(w)
	if err := writePart(zw, aasxContentTypesPart, aasxContentTypes()); err != nil {
		return err
	}
	if err := writePart(zw, aasxRelsPart, aasxRelationships()); err != nil {
		return err
	}
	if err := writePart(zw, aasxAASEnvPart, envJSON); err != nil {
		return err
	}
	return zw.Close()
}

// UnpackAASX parses a standards-compliant IEC 63278 .aasx package and restores
// the AssetAdministrationShell it carries.
func UnpackAASX(r io.ReaderAt, size int64) (*AssetAdministrationShell, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAASXInvalidPackage, err)
	}

	var envBytes, contentTypesBytes []byte
	var sawRels bool
	for _, f := range zr.File {
		switch f.Name {
		case aasxContentTypesPart:
			contentTypesBytes, err = readPart(f)
		case aasxRelsPart:
			sawRels = true
		case aasxAASEnvPart:
			envBytes, err = readPart(f)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: read %s: %v", ErrAASXInvalidPackage, f.Name, err)
		}
	}

	if len(envBytes) == 0 {
		return nil, fmt.Errorf("%w: missing %s", ErrAASXMissingPart, aasxAASEnvPart)
	}
	if !sawRels {
		return nil, fmt.Errorf("%w: missing %s", ErrAASXMissingPart, aasxRelsPart)
	}
	if !bytes.Contains(contentTypesBytes, []byte(aasxContentTypeJSON)) {
		return nil, fmt.Errorf("%w: %s does not declare %s", ErrAASXInvalidPackage, aasxContentTypesPart, aasxContentTypeJSON)
	}

	var env aasxEnvironment
	if err := json.Unmarshal(envBytes, &env); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAASXInvalidPackage, err)
	}
	if len(env.AssetAdministrationShells) != 1 || env.AssetAdministrationShells[0].Shell == nil {
		return nil, ErrAASXNoShell
	}
	shell := env.AssetAdministrationShells[0].Shell
	if err := shell.Validate(); err != nil {
		return nil, err
	}
	return shell, nil
}

func readPart(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return nil, err
	}
	return b, nil
}

func writePart(zw *zip.Writer, name string, body []byte) error {
	hdr := &zip.FileHeader{Name: name, Method: zip.Store}
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

// buildEnvironment constructs the IEC 63278 environment envelope for one shell,
// deriving canonical eCl@ss semantic IDs per class, submodel, and element.
func buildEnvironment(shell *AssetAdministrationShell) (aasxEnvironment, error) {
	rec := aasxShellRecord{
		ID:         shell.ID,
		IDShort:    shellIDShort(shell),
		SemanticID: shellClassSemanticID(shell.Class),
		Shell:      shell,
		Assets: []aasxAssetRecord{{
			ID:            shell.ID + ":asset",
			IDShort:       shellIDShort(shell),
			GlobalAssetID: shellGlobalAssetID(shell),
		}},
	}

	rec.Submodels = append(rec.Submodels, aasxSubmodelRecord{
		ID:         shell.ID + ":technical_specification",
		IDShort:    "TechnicalSpecification",
		SemanticID: SemanticIDTechnicalSpecification,
		Path:       "technical_specification",
		Elements: []aasxElementRecord{
			{ID: shell.ID + ":technical_specification:rated_swl", IDShort: "RatedSWL", SemanticID: SemanticIDTechnicalSpecification},
		},
	})
	rec.Submodels = append(rec.Submodels, aasxSubmodelRecord{
		ID:         shell.ID + ":operational_state",
		IDShort:    "OperationalState",
		SemanticID: eclassIDOperationalState,
		Path:       "operational_state",
	})
	rec.Submodels = append(rec.Submodels, aasxSubmodelRecord{
		ID:         shell.ID + ":assurance_evidence",
		IDShort:    "AssuranceEvidence",
		SemanticID: eclassIDAssuranceEvidence,
		Path:       "assurance_evidence",
		Elements: []aasxElementRecord{
			{ID: shell.ID + ":assurance_evidence:proof_load_tests", IDShort: "ProofLoadTests", SemanticID: SemanticIDProofLoadTest},
			{ID: shell.ID + ":assurance_evidence:ndt_reports", IDShort: "NDTReports", SemanticID: eclassIDNDTReport},
			{ID: shell.ID + ":assurance_evidence:sealed_certificates", IDShort: "SealedCertificates", SemanticID: eclassIDSealedCertificate},
		},
	})

	return aasxEnvironment{AssetAdministrationShells: []aasxShellRecord{rec}}, nil
}

func shellIDShort(shell *AssetAdministrationShell) string {
	switch shell.Class {
	case ClassFunctionalLocation:
		if shell.Location != nil && shell.Location.TagID != "" {
			return shell.Location.TagID
		}
	case ClassInstalledEquipment:
		if shell.Equipment != nil && shell.Equipment.SerialNumber != "" {
			return shell.Equipment.SerialNumber
		}
	}
	return shell.ID
}

func shellClassSemanticID(class AASItemClass) string {
	if class == ClassInstalledEquipment {
		return eclassIDInstalledEquipment
	}
	return eclassIDFunctionalLocation
}

func shellGlobalAssetID(shell *AssetAdministrationShell) string {
	if shell.Class == ClassInstalledEquipment && shell.Equipment != nil {
		return shell.Equipment.Passport.AssetDID
	}
	return shell.ID
}

func aasxContentTypes() []byte {
	return []byte(strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`,
		`  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`,
		`  <Default Extension="xml" ContentType="application/xml"/>`,
		`  <Override PartName="/aasx/aasenv.json" ContentType="` + aasxContentTypeJSON + `"/>`,
		`</Types>`,
	}, "\n"))
}

func aasxRelationships() []byte {
	return []byte(strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`,
		`  <Relationship Id="rel-aasx" Type="http://admin-shell.io/aasx/relationships/aasx-package" Target="aasx/aasenv.json"/>`,
		`</Relationships>`,
	}, "\n"))
}
