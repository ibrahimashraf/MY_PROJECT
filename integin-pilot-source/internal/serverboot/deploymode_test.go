package serverboot

import (
	"testing"
)

func TestParseDeployMode(t *testing.T) {
	cases := []struct {
		raw     string
		want    DeployMode
		wantErr bool
	}{
		{"", DeployModeCloud, false},
		{"cloud", DeployModeCloud, false},
		{"CLOUD", DeployModeCloud, false},
		{"sovereign", DeployModeSovereign, false},
		{"airgap", DeployModeAirgap, false},
		{" AIRGAP ", DeployModeAirgap, false},
		{"edge", "", true},
	}
	for _, c := range cases {
		got, err := parseDeployMode(c.raw)
		if c.wantErr && err == nil {
			t.Fatalf("parseDeployMode(%q): expected error", c.raw)
		}
		if !c.wantErr && (err != nil || got != c.want) {
			t.Fatalf("parseDeployMode(%q) = %q, %v; want %q", c.raw, got, err, c.want)
		}
	}
}

func TestCheckDeployModeCloudAlwaysPasses(t *testing.T) {
	if err := checkDeployMode(DeployModeCloud, "", true, "https://accounts.google.com", "https://s3.amazonaws.com", "", "0.0.0.0:8080", false, "https://tsa.example.com"); err != nil {
		t.Fatalf("cloud must not gate anything: %v", err)
	}
}

func TestCheckDeployModeRequiresDB(t *testing.T) {
	for _, mode := range []DeployMode{DeployModeSovereign, DeployModeAirgap} {
		if err := checkDeployMode(mode, "", false, "", "", "rustfs", "127.0.0.1:8080", false, ""); err == nil {
			t.Fatalf("%s without INTEGIN_DB_URL must fail", mode)
		}
	}
}

func TestCheckDeployModeAirgapEndpoints(t *testing.T) {
	db := "postgres://u:p@postgres:5432/db?sslmode=disable"
	internal := []struct{ issuer, s3, store string }{
		{"http://127.0.0.1:18180/realms/x", "http://127.0.0.1:19000", "rustfs"},
		{"http://localhost:18180/realms/x", "http://rustfs:9000", "rustfs"},
		{"http://casdoor:8000", "http://minio.storage.svc.cluster.local:9000", "s3"},
		{"http://idp.local:8443", "", "memory"},
		{"", "http://s3.internal:9000", "minio"},
		{"", "", "memory"},
	}
	for _, c := range internal {
		if err := checkDeployMode(DeployModeAirgap, db, c.issuer != "", c.issuer, c.s3, c.store, "127.0.0.1:18080", false, ""); err != nil {
			t.Fatalf("internal endpoints must pass (issuer=%q s3=%q store=%q): %v", c.issuer, c.s3, c.store, err)
		}
	}
	public := []struct{ issuer, s3, store string }{
		{"https://accounts.google.com", "", "memory"},
		{"", "https://s3.amazonaws.com", "rustfs"},
		{"https://login.microsoftonline.com/tenant", "https://minio.example.com", "s3"},
		{"http://127.0.0.1:18180/realms/x", "http://127.0.0.1:19000", ""},
		{"http://127.0.0.1:18180/realms/x", "http://127.0.0.1:19000", "disabled"},
	}
	for _, c := range public {
		if err := checkDeployMode(DeployModeAirgap, db, c.issuer != "", c.issuer, c.s3, c.store, "127.0.0.1:18080", false, ""); err == nil {
			t.Fatalf("public endpoints or missing store must fail (issuer=%q s3=%q store=%q)", c.issuer, c.s3, c.store)
		}
	}
}

func TestCheckDeployModeAirgapTSA(t *testing.T) {
	db := "postgres://u:p@postgres:5432/db?sslmode=disable"
	base := func(tsa string) error {
		return checkDeployMode(DeployModeAirgap, db, false, "", "", "memory", "127.0.0.1:18080", false, tsa)
	}
	for _, tsa := range []string{"", "http://127.0.0.1:18280", "http://tsa:8080", "http://tsa.internal:8080"} {
		if err := base(tsa); err != nil {
			t.Fatalf("unset or internal TSA %q must pass: %v", tsa, err)
		}
	}
	for _, tsa := range []string{"https://tsa.example.com", "https://freetsa.org/tsr", "http://203.0.113.7:8080"} {
		if err := base(tsa); err == nil {
			t.Fatalf("public TSA %q must fail", tsa)
		}
	}
}

func TestCheckDeployModeAirgapBind(t *testing.T) {
	db := "postgres://u:p@postgres:5432/db?sslmode=disable"
	for _, bind := range []string{"127.0.0.1:8080", "localhost:8080", "[::1]:8080"} {
		if err := checkDeployMode(DeployModeAirgap, db, false, "", "", "memory", bind, false, ""); err != nil {
			t.Fatalf("loopback bind %q must pass: %v", bind, err)
		}
	}
	if err := checkDeployMode(DeployModeAirgap, db, false, "", "", "memory", ":8080", false, ""); err == nil {
		t.Fatal("default all-interfaces bind must fail without opt-out")
	}
	if err := checkDeployMode(DeployModeAirgap, db, false, "", "", "memory", "0.0.0.0:8080", false, ""); err == nil {
		t.Fatal("LAN bind must fail without opt-out")
	}
	if err := checkDeployMode(DeployModeAirgap, db, false, "", "", "memory", "0.0.0.0:8080", true, ""); err != nil {
		t.Fatalf("LAN bind with explicit opt-out must pass: %v", err)
	}
	if err := checkDeployMode(DeployModeSovereign, db, false, "", "", "", "0.0.0.0:8080", false, "https://tsa.example.com"); err != nil {
		t.Fatalf("sovereign must not gate bind, store, or TSA: %v", err)
	}
}

func TestIsInternalHost(t *testing.T) {
	internal := []string{"127.0.0.1", "::1", "localhost", "postgres", "casdoor", "idp.local", "s3.internal", "minio.storage.svc.cluster.local", "svc.svc"}
	for _, h := range internal {
		if !isInternalHost(h) {
			t.Fatalf("isInternalHost(%q) = false, want true", h)
		}
	}
	external := []string{"accounts.google.com", "s3.amazonaws.com", "8.8.8.8", "2001:4860:4860::8888", "",
		// RFC 1918 private IPs are deliberately external: a private address
		// appears on every LAN and cannot prove the box is on the enclave LAN.
		"10.0.0.5", "172.16.0.5", "172.31.255.1", "192.168.1.50"}
	for _, h := range external {
		if isInternalHost(h) {
			t.Fatalf("isInternalHost(%q) = true, want false", h)
		}
	}
}
