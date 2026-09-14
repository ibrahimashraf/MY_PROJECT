package conformity

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"integin/pkg/domain"
)

func sapExportFixture() domain.SAPPMExport {
	return domain.SAPPMExport{
		EquipmentRecords: []domain.EquipmentMasterRecord{{
			EquipmentNumber:   "EQ-1001",
			Manufacturer:      "Liebherr",
			ModelNumber:       "LR-1300",
			SerialNumber:      "SN-1300-01",
			EquipmentCategory: "CRAWLER_CRANE",
			RatedSWL:          "300t",
			MountedTagID:      "TAG-CRANE-01",
			AssetDID:          "did:key:z6Mkackle",
			JurisdictionISO:   "SA",
			Status:            domain.AssetStatusOperational,
		}},
		CharacteristicValues: []domain.CharacteristicRecord{{
			CharacteristicName:  "RATED_SWL",
			CharacteristicValue: "300t",
			ObjectID:            "EQ-1001",
			CFIHOSClassifier:    domain.CFIHOSRatedSWL,
		}},
	}
}

func maximoExportFixture() domain.MaximoAssetExport {
	return domain.MaximoAssetExport{
		Assets: []domain.MaximoAsset{{
			AssetNum:         "EQ-1001",
			Description:      "LR-1300 crane",
			SerialNum:        "SN-1300-01",
			Status:           "OPERATING",
			SiteID:           "INTEGIN",
			AssetTag:         "TAG-CRANE-01",
			ClassStructureID: "INTEGER-PM",
			Classification:   domain.CFIHOSInstalledEquipment,
		}},
		Locations: []domain.MaximoLocation{{
			Location:       "FL-CRANE-01",
			Description:    "Crane pad",
			SiteID:         "INTEGIN",
			Status:         "OPERATING",
			Classification: domain.CFIHOSFunctionalLocation,
		}},
	}
}

func TestSAPPostEquipmentRecordsWithTransactionID(t *testing.T) {
	conn := NewStagingSAPConnector()
	export := sapExportFixture()

	resp, err := conn.PostEquipment(context.Background(), export)
	if err != nil {
		t.Fatalf("PostEquipment: %v", err)
	}
	if resp.Status != "OK" || resp.TransactionID == "" {
		t.Fatalf("resp = %+v", resp)
	}
	submitted := conn.Submitted()
	if len(submitted) != 1 {
		t.Fatalf("recorded %d exports, want 1", len(submitted))
	}
	if !reflect.DeepEqual(submitted[0], export) {
		t.Fatalf("payload fidelity lost:\n got %+v\nwant %+v", submitted[0], export)
	}
}

func TestSAPInvalidCFIHOSClassifierRejected(t *testing.T) {
	conn := NewStagingSAPConnector()
	export := sapExportFixture()
	export.CharacteristicValues[0].CFIHOSClassifier = "EQUIPMENT.RATED_SWL"

	resp, err := conn.PostEquipment(context.Background(), export)
	if !errors.Is(err, ErrInvalidSAPPayload) {
		t.Fatalf("err = %v, want ErrInvalidSAPPayload", err)
	}
	if resp.Status != "REJECTED" {
		t.Fatalf("status = %q, want REJECTED", resp.Status)
	}
	if len(conn.Submitted()) != 0 {
		t.Fatal("rejected export was recorded")
	}
}

func TestSAPEmptyExportRejected(t *testing.T) {
	if err := ValidateSAPPayload(domain.SAPPMExport{}); !errors.Is(err, ErrInvalidSAPPayload) {
		t.Fatalf("err = %v, want ErrInvalidSAPPayload", err)
	}
}

func TestSAPTransientFailureRetriesDispatch(t *testing.T) {
	conn := NewStagingSAPConnector()
	conn.FailNext(2)

	resp, err := PostEquipmentRetry(context.Background(), conn, sapExportFixture(), 3)
	if err != nil {
		t.Fatalf("PostEquipmentRetry: %v", err)
	}
	if resp.Status != "OK" {
		t.Fatalf("status = %q, want OK", resp.Status)
	}
	if got := len(conn.Submitted()); got != 1 {
		t.Fatalf("recorded %d exports after retry, want 1", got)
	}
}

func TestSAPTransientFailureExhaustsAttempts(t *testing.T) {
	conn := NewStagingSAPConnector()
	conn.FailNext(5)

	_, err := PostEquipmentRetry(context.Background(), conn, sapExportFixture(), 2)
	if !errors.Is(err, ErrTransientNetwork) {
		t.Fatalf("err = %v, want ErrTransientNetwork after attempts exhausted", err)
	}
}

func TestMaximoSyncAssetsStatusAndCount(t *testing.T) {
	conn := NewStagingMaximoConnector()
	resp, err := conn.SyncAssets(context.Background(), maximoExportFixture())
	if err != nil {
		t.Fatalf("SyncAssets: %v", err)
	}
	if resp.Status != "OK" || resp.Count != 2 {
		t.Fatalf("resp = %+v, want OK count 2", resp)
	}
	if got := len(conn.Submitted()); got != 1 {
		t.Fatalf("recorded %d exports, want 1", got)
	}
}

func TestMaximoInvalidAssetRejected(t *testing.T) {
	conn := NewStagingMaximoConnector()
	export := maximoExportFixture()
	export.Assets[0].Classification = "NOT_CFIHOS"

	_, err := conn.SyncAssets(context.Background(), export)
	if !errors.Is(err, ErrInvalidMaximoPayload) {
		t.Fatalf("err = %v, want ErrInvalidMaximoPayload", err)
	}
	if len(conn.Submitted()) != 0 {
		t.Fatal("rejected export was recorded")
	}
}

func TestMaximoTransientFailureRetriesSync(t *testing.T) {
	conn := NewStagingMaximoConnector()
	conn.FailNext(1)

	resp, err := SyncAssetsRetry(context.Background(), conn, maximoExportFixture(), 3)
	if err != nil {
		t.Fatalf("SyncAssetsRetry: %v", err)
	}
	if resp.Status != "OK" || resp.Count != 2 {
		t.Fatalf("resp = %+v", resp)
	}
}
