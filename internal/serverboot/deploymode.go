package serverboot

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// DeployMode is the deployment topology selected by INTEGIN_DEPLOY_MODE.
// Empty/unset means cloud (current behavior, unchanged).
type DeployMode string

const (
	DeployModeCloud     DeployMode = "cloud"
	DeployModeSovereign DeployMode = "sovereign"
	DeployModeAirgap    DeployMode = "airgap"
)

// parseDeployMode normalizes the raw env value. Empty means cloud.
func parseDeployMode(raw string) (DeployMode, error) {
	switch mode := DeployMode(strings.TrimSpace(strings.ToLower(raw))); mode {
	case "", DeployModeCloud:
		return DeployModeCloud, nil
	case DeployModeSovereign:
		return DeployModeSovereign, nil
	case DeployModeAirgap:
		return DeployModeAirgap, nil
	default:
		return "", fmt.Errorf("invalid INTEGIN_DEPLOY_MODE %q: want cloud|sovereign|airgap", raw)
	}
}

// checkDeployMode enforces the topology contract, fail-closed at boot:
//
// This is an operator-misconfiguration guard (wrong endpoint pasted into an
// airgap env file), not a security boundary: it classifies URL strings once
// at boot and cannot see DNS rebinding or runtime egress. Enclave containment
// still requires network-layer egress rules.
//
//   - sovereign/airgap require INTEGIN_DB_URL (no legacy JSON bootstrap, so
//     tenant RLS is always enforced).
//   - airgap additionally refuses public outbound endpoints: when OIDC is
//     enabled its issuer must be internal, a configured S3 evidence endpoint
//     must be internal, and a configured TSA timestamp endpoint must be
//     internal (timestamps carry certificate hashes outward). Single-label
//     hostnames (compose service names), loopback, and
//     .local/.internal/cluster suffixes count as internal; anything else is
//     treated as data leaving the enclave. Deliberately, RFC 1918 private IPs
//     count as external: a private address appears on every LAN and cannot
//     prove the box is on the enclave LAN.
//   - airgap requires a local evidence store (INTEGIN_EVIDENCE_STORE set to
//     anything but empty/disabled): an airgap box that accepts inspections
//     without anywhere to store evidence fails loudly at startup instead of
//     losing evidence silently.
//   - airgap requires a loopback bind (INTEGIN_HTTP_ADDR on localhost/127.0.0.1)
//     unless the operator explicitly opts out with
//     INTEGIN_AIRGAP_ALLOW_LAN_BIND=true (for proxy-fronted or direct-sync rigs;
//     the opt-out is logged loudly at boot). Rationale: direct LAN exposure
//     of the API must be a deliberate choice, never the default.
func checkDeployMode(mode DeployMode, dbURL string, oidcEnabled bool, oidcIssuer, s3Endpoint, evidenceStore, bindAddr string, allowLANBind bool, tsaURL string) error {
	if mode == DeployModeCloud {
		return nil
	}
	if strings.TrimSpace(dbURL) == "" {
		return fmt.Errorf("INTEGIN_DEPLOY_MODE=%s requires INTEGIN_DB_URL (legacy JSON bootstrap is not permitted)", mode)
	}
	if mode == DeployModeSovereign {
		return nil
	}
	if oidcEnabled && !isInternalEndpoint(oidcIssuer) {
		return fmt.Errorf("INTEGIN_DEPLOY_MODE=airgap refuses public OIDC issuer %q (use a loopback or in-enclave IdP, or disable OIDC)", oidcIssuer)
	}
	if strings.TrimSpace(s3Endpoint) != "" && !isInternalEndpoint(s3Endpoint) {
		return fmt.Errorf("INTEGIN_DEPLOY_MODE=airgap refuses public S3 endpoint %q (use in-enclave RustFS/MinIO)", s3Endpoint)
	}
	if strings.TrimSpace(tsaURL) != "" && !isInternalEndpoint(tsaURL) {
		return fmt.Errorf("INTEGIN_DEPLOY_MODE=airgap refuses public TSA endpoint %q (timestamps carry certificate hashes outward; use the in-enclave TSA)", tsaURL)
	}
	switch store := strings.TrimSpace(strings.ToLower(evidenceStore)); store {
	case "", "disabled":
		return fmt.Errorf("INTEGIN_DEPLOY_MODE=airgap requires INTEGIN_EVIDENCE_STORE (memory or in-enclave RustFS/MinIO); refusing to accept inspections with nowhere to store evidence")
	}
	if !isLoopbackAddress(bindAddr) && !allowLANBind {
		return fmt.Errorf("INTEGIN_DEPLOY_MODE=airgap refuses non-loopback bind %q (bind localhost, or set INTEGIN_AIRGAP_ALLOW_LAN_BIND=true for proxy-fronted/direct-sync rigs)", bindAddr)
	}
	return nil
}

// isInternalEndpoint reports whether a URL points inside the enclave.
func isInternalEndpoint(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" {
		return false
	}
	return isInternalHost(u.Hostname())
}

// isInternalHost classifies one hostname: loopback IPs, localhost,
// single-label names (compose/K8s service names), and private suffixes.
func isInternalHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	if host == "localhost" {
		return true
	}
	if !strings.Contains(host, ".") {
		return true
	}
	for _, suffix := range []string{".local", ".internal", ".svc.cluster.local", ".svc"} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}
