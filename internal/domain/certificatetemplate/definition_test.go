package certificatetemplate

import "testing"

func validDefinition() Definition {
	return Definition{ID: "template-1", TemplateCode: "lifting_certificate", Version: 1, Title: "Lifting certificate", AssetType: "lifting", Status: Draft, CatalogVersion: 1, PageCount: 1, PageWidth: 595, PageHeight: 842, Cells: []Cell{
		{ID: "inspection_id", PageNumber: 1, Rectangle: Rectangle{X: 20, Y: 20, Width: 180, Height: 30}, Kind: TextCell, BindingKey: InspectionID, FitPolicy: SingleLineRequired, MaxLines: 1, Required: true},
		{ID: "asset_id", PageNumber: 1, Rectangle: Rectangle{X: 20, Y: 60, Width: 180, Height: 30}, Kind: TextCell, BindingKey: InspectionAssetID, FitPolicy: SingleLineRequired, MaxLines: 1, Required: true},
	}}
}

func TestDefinitionValidateAcceptsBoundedCanonicalCells(t *testing.T) {
	if err := validDefinition().Validate(DefaultCatalog()); err != nil {
		t.Fatalf("validate bounded canonical definition: %v", err)
	}
}

func TestDefinitionValidateRejectsUnsafeOrUnavailableBindings(t *testing.T) {
	definition := validDefinition()
	definition.Cells[0].BindingKey = "asset.serial_number"
	if err := definition.Validate(DefaultCatalog()); err == nil {
		t.Fatal("expected unavailable serial-number binding rejection")
	}
	definition = validDefinition()
	definition.Cells[0].Kind = RepeatingRegion
	definition.Cells[0].RepeatSource = "inspection.findings"
	definition.Cells[0].FitPolicy = RepeatRequired
	definition.Cells[0].MaxItems = 4
	if err := definition.Validate(DefaultCatalog()); err == nil {
		t.Fatal("expected unavailable repeating source rejection")
	}
}

func TestDefinitionValidateRejectsOverlapAndUnboundedOverflow(t *testing.T) {
	definition := validDefinition()
	definition.Cells[1].Rectangle = definition.Cells[0].Rectangle
	if err := definition.Validate(DefaultCatalog()); err == nil {
		t.Fatal("expected overlap rejection")
	}
	definition = validDefinition()
	definition.Cells[0].FitPolicy = WrapRequired
	definition.Cells[0].MaxLines = 0
	if err := definition.Validate(DefaultCatalog()); err == nil {
		t.Fatal("expected unbounded text rejection")
	}
}
