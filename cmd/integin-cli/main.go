package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"integin/pkg/domain"
	"integin/pkg/licensing"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "gen-key":
		handleGenKey()
	case "gen-tsa":
		handleGenTSA(os.Args[2:])
	case "issue-license":
		handleIssueLicense(os.Args[2:])
	case "verify-license":
		handleVerifyLicense(os.Args[2:])
	case "asset-did":
		handleAssetDID(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("INTEGIN Master Enterprise CLI (v5.0.0)")
	fmt.Println("Usage: integin-cli <command> [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  gen-key            Generate a fresh Ed25519 root master authority keypair")
	fmt.Println("  gen-tsa              Generate a pilot RFC 3161 TSA signer keypair + self-signed roots PEM")
	fmt.Println("  issue-license      Mint a cryptographically signed license token")
	fmt.Println("  verify-license     Verify a signed license token locally against a public key")
	fmt.Println("  asset-did          Generate a deterministic W3C Asset DID")
}

func handleGenKey() {
	pub, priv, err := licensing.GenerateMasterKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== INTEGIN Root Authority Ed25519 Keypair ===")
	fmt.Printf("Public Key  (Hex): %s\n", hex.EncodeToString(pub))
	fmt.Printf("Private Key (Hex): %s\n", hex.EncodeToString(priv))
	fmt.Println("\nCAUTION: Store the Private Key in an HSM or sealed vault. Distribute the Public Key to nodes.")
}

func handleGenTSA(args []string) {
	fs := flag.NewFlagSet("gen-tsa", flag.ExitOnError)
	keyOut := fs.String("key-out", `C:\MY_PROJECT\private\integin-secrets\tsa-pilot-signer.key`, "Signer private key output path (PKCS#8 PEM, mode 0600)")
	rootsOut := fs.String("roots-out", `C:\MY_PROJECT\private\integin-secrets\tsa-pilot-roots.pem`, "Self-signed TSA roots output path (PEM, mode 0600)")
	commonName := fs.String("cn", "INTEGIN Pilot TSA", "Signer certificate common name")
	years := fs.Int("years", 10, "Certificate validity in years")
	force := fs.Bool("force", false, "Overwrite existing output files")

	_ = fs.Parse(args)

	if *keyOut == "" || *rootsOut == "" {
		fmt.Fprintln(os.Stderr, "Error: -key-out and -roots-out are required")
		os.Exit(1)
	}
	if !*force {
		for _, path := range []string{*keyOut, *rootsOut} {
			if _, err := os.Stat(path); err == nil {
				fmt.Fprintf(os.Stderr, "Error: %s exists (pass -force to overwrite)\n", path)
				os.Exit(1)
			}
		}
	}
	if *years <= 0 || *years > 30 {
		fmt.Fprintln(os.Stderr, "Error: -years must be between 1 and 30")
		os.Exit(1)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
		os.Exit(1)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating serial: %v\n", err)
		os.Exit(1)
	}
	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: *commonName},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.AddDate(*years, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating certificate: %v\n", err)
		os.Exit(1)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding key: %v\n", err)
		os.Exit(1)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	rootsPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(*keyOut, keyPEM, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write key: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*rootsOut, rootsPEM, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write roots: %v\n", err)
		os.Exit(1)
	}
	fingerprint := sha256.Sum256(der)
	fmt.Println("Pilot TSA signer generated (pilot trust anchor only — never use for production timestamps)")
	fmt.Printf("   Key:         %s\n", *keyOut)
	fmt.Printf("   Roots:       %s\n", *rootsOut)
	fmt.Printf("   Fingerprint: %s\n", hex.EncodeToString(fingerprint[:]))
	fmt.Printf("   Valid:       %s to %s\n", tmpl.NotBefore.Format(time.RFC3339), tmpl.NotAfter.Format(time.RFC3339))
	fmt.Println("Provision the server with INTEGIN_TSA_URL + INTEGIN_TSA_ROOTS_FILE=<roots> and the responder with INTEGIN_TSA_SIGNER_KEY_FILE=<key> + INTEGIN_TSA_ROOTS_FILE=<roots>.")
}

func handleIssueLicense(args []string) {
	fs := flag.NewFlagSet("issue-license", flag.ExitOnError)
	privKeyHex := fs.String("priv-key", "", "Root Authority Private Key (Hex)")
	org := fs.String("org", "Enterprise Corp", "Licensed Organization Name")
	tier := fs.String("tier", "ENTERPRISE", "Tier: COMMUNITY, ENTERPRISE, SOVEREIGN_AIRGAP")
	mode := fs.String("mode", "CLOUD_SAAS", "Mode: CLOUD_SAAS, ON_PREMISE, AIR_GAPPED_EDGE")
	jurisdictions := fs.String("jurisdictions", "GLOBAL", "Comma-separated ISO codes (e.g. 'SA,AE,US' or 'GLOBAL')")
	days := fs.Int("days", 365, "Validity duration in days")
	hwLock := fs.String("hw-lock", "", "Optional hardware lock ID for air-gapped rigs")
	outPath := fs.String("out", "integin_license.json", "Output JSON path")

	_ = fs.Parse(args)

	if *privKeyHex == "" {
		fmt.Fprintln(os.Stderr, "Error: -priv-key is required")
		os.Exit(1)
	}

	privBytes, err := hex.DecodeString(strings.TrimSpace(*privKeyHex))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid private key hex: %v\n", err)
		os.Exit(1)
	}

	issuer, err := licensing.NewMasterLicenseIssuer(privBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create issuer: %v\n", err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	jurList := strings.Split(*jurisdictions, ",")
	for i := range jurList {
		jurList[i] = strings.TrimSpace(jurList[i])
	}

	payload := licensing.LicensePayload{
		LicenseID:      fmt.Sprintf("LIC-%d", now.UnixNano()),
		Issuer:         "did:integin:authority:master-pki",
		IssuedToOrg:    *org,
		Tier:           licensing.LicenseTier(*tier),
		Mode:           licensing.DeploymentMode(*mode),
		IssuedAt:       now,
		NotBefore:      now.Add(-5 * time.Minute),
		ExpiresAt:      now.Add(time.Duration(*days) * 24 * time.Hour),
		HardwareLockID: strings.TrimSpace(*hwLock),
		Covenants: licensing.PlatformCovenants{
			MaxTenants:               100,
			MaxInspectorsPerTenant:   1000,
			AllowedJurisdictions:     jurList,
			EnabledModules:           []string{"LIFTING", "NDT", "PRESSURE", "OFFSHORE"},
			AirGapOfflineGraceDays:   180,
			AllowSubcontractorPortal: true,
			AllowDynamicRulesAuthor:  true,
		},
	}

	token, err := issuer.IssueLicense(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error issuing license: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode token: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outPath, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ License issued successfully: %s\n", *outPath)
	fmt.Printf("   Org:       %s\n", *org)
	fmt.Printf("   Tier:      %s\n", *tier)
	fmt.Printf("   Expires:   %s\n", payload.ExpiresAt.Format(time.RFC3339))
}

func handleVerifyLicense(args []string) {
	fs := flag.NewFlagSet("verify-license", flag.ExitOnError)
	pubKeyHex := fs.String("pub-key", "", "Root Authority Public Key (Hex)")
	licPath := fs.String("lic", "integin_license.json", "License token JSON path")
	hwID := fs.String("hw-id", "", "Current hardware ID (if hardware lock enabled)")

	_ = fs.Parse(args)

	if *pubKeyHex == "" {
		fmt.Fprintln(os.Stderr, "Error: -pub-key is required")
		os.Exit(1)
	}

	data, err := os.ReadFile(*licPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read license file: %v\n", err)
		os.Exit(1)
	}

	var token licensing.SignedLicenseToken
	if err := json.Unmarshal(data, &token); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid license JSON: %v\n", err)
		os.Exit(1)
	}

	validator, err := licensing.NewUniversalLicenseValidator(*pubKeyHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize validator: %v\n", err)
		os.Exit(1)
	}

	if err := validator.ValidateToken(&token, *hwID); err != nil {
		fmt.Fprintf(os.Stderr, "❌ License Verification FAILED: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ License Verification PASSED (Valid Cryptographic Signature)")
	fmt.Printf("   Issued To: %s\n", token.Payload.IssuedToOrg)
	fmt.Printf("   Tier:      %s\n", token.Payload.Tier)
	fmt.Printf("   Valid To:  %s\n", token.Payload.ExpiresAt.Format(time.RFC3339))
}

func handleAssetDID(args []string) {
	fs := flag.NewFlagSet("asset-did", flag.ExitOnError)
	mfg := fs.String("mfg", "", "Manufacturer (e.g. Kato, Liebherr)")
	model := fs.String("model", "", "Model (e.g. NK-1000)")
	serial := fs.String("serial", "", "Serial Number")
	epoch := fs.Int64("epoch", 1700000000, "Base registration unix epoch")

	_ = fs.Parse(args)

	if *mfg == "" || *model == "" || *serial == "" {
		fmt.Fprintln(os.Stderr, "Error: -mfg, -model, and -serial are required")
		os.Exit(1)
	}

	did := domain.GenerateAssetDID(*mfg, *model, *serial, *epoch)
	fmt.Printf("Generated W3C Asset DID:\n%s\n", did.String())
}
