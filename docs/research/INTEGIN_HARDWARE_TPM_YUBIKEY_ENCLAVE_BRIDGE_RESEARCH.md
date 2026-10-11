# 🛡️ Primary Research: Native TPM 2.0 & YubiKey PIV Hardware Enclave Bridge Architecture

**Document ID:** RESEARCH-2026-10-11-TPM-YUBIKEY-BRIDGE  
**Author:** Lead Systems Architect & Core Assurance Authority  
**Status:** RATIFIED & IMPLEMENTED ✅  
**Governing Initiatives:** INTEGIN Wayfinder (`docs/decisions/WAYFINDER_RUST_SYSTEMS_ADOPTION_MAP.md`), Master Tier L8 Hardware Attestation  
**Target Environments:** Rugged Field Tablets (Panasonic Toughbook 40/G2, Dell Latitude 7230 Rugged Extreme, Zebra L10, Getac UX10), Windows 11 IoT Enterprise, Rugged Industrial Linux  

---

## 1. Executive Summary & Problem Formulation

In high-risk heavy lifting and industrial crane operations (ISO 17020 §6.2, ASME B30.5, OSHA 1926.1400), inspection sign-offs require non-repudiable cryptographic device attestation. 

Prior to this work, INTEGIN's mobile attestation subsystem (`pkg/onboarding`) supported consumer mobile device enclaves:
1. **Apple Secure Enclave (`SECURE_ENCLAVE`)** via App Attest / DeviceCheck (FIPS 140-3 Level 2).
2. **Android StrongBox (`STRONGBOX`)** via Keymaster hardware key attestation (FIPS 140-3 Level 3).
3. **Android TEE (`TEE`)** via TrustZone keystores.

However, commercial heavy-lifting field inspectors overwhelmingly operate **ruggedized industrial PC/tablet hardware** running Windows 11 IoT Enterprise or embedded Linux. These devices **do not possess Apple SEP or Android StrongBox chips**. Instead, enterprise-grade hardware assurance is delivered via:
- **Discrete TPM 2.0 Microcontrollers:** (e.g. Infineon OPTIGA™ SLB 9670, STMicroelectronics ST33, Nuvoton NPCT750) meeting TCG TPM 2.0 Library Spec and Common Criteria EAL4+ / FIPS 140-3 Level 2.
- **Hardware Security Tokens / SmartCards:** (e.g. YubiKey 5 Series FIPS, Identiv uTrust) implementing NIST SP 800-73-4 (PIV card interface) meeting FIPS 140-3 Level 3 (Overall and Physical Security).

This research resolves **NY-01**: establishing the architectural bridge and contract surface for TPM 2.0 and YubiKey PIV attestation without violating INTEGIN's bedrock invariants (`CGO_ENABLED=0`, zero in-process FFI, and Go + PostgreSQL sole authority).

---

## 2. Invariant Boundary & Architecture Enforcement

### 2.1 The Anti-Pattern: In-Process CGO / C DLLs
Attempting to link PC/SC (`winscard.dll`, `libpcsclite.so`) or TCG TSS2 libraries (`tss2-esys`, `tss2-tcti`) directly into the Go backend runtime via `cgo`:
- Violates **AGENTS.md Invariant 1 & 5** (`CGO_ENABLED=0` strictly enforced).
- Compromises cross-compilation and air-gapped sovereign appliance distribution.
- Introduces unmanaged C memory vulnerabilities and stack-switching performance penalties (50–160 ns per invocation).

### 2.2 The Approved Pattern: Out-of-Process Hardware Bridge & Pure Go Verification
The bridge is bifurcated across a clear execution seam:

```
┌────────────────────────────────────────────────────────────────────────┐
│                   FIELD TERMINAL / CLIENT APPLIANCE                    │
│                                                                        │
│  [Hardware Token]               [Discrete Chip]                        │
│   YubiKey 5 FIPS                  TPM 2.0 (Infineon/ST33)              │
│         │                               │                              │
│         ▼                               ▼                              │
│   PC/SC / PIV Driver              Windows CNG / /dev/tpmrm0            │
│         │                               │                              │
│         └───────────────┬───────────────┘                              │
│                         ▼                                              │
│         [Out-of-Process Hardware Agent / Rust CLI]                     │
│         • Reads X.509 Attestation Cert (Slot 9a/9c)                    │
│         • Performs TPM2_ActivateCredential / AIK Quote                 │
│         • Signs Nonce + Generates AttestationClaim                     │
└─────────────────────────┼──────────────────────────────────────────────┘
                          │ JSON Enrollment / Receipt Submission
                          │ (HTTPS / Offline Barcode)
                          ▼
┌────────────────────────────────────────────────────────────────────────┐
│             INTEGIN GO MODULAR MONOLITH (CGO_ENABLED=0)                │
│                                                                        │
│  pkg/onboarding/attestation.go                                         │
│  • KeyOriginTPM ("TPM_2_0")       ──► FIPS 140-3 Level 2              │
│  • KeyOriginYubiKey ("YUBIKEY_PIV") ──► FIPS 140-3 Level 3              │
│  • Policy enforcement (RequireHardware, RequireBiometric, RequireFIPS) │
│  • Pure Go crypto verification (Ed25519 / RSA / ECDSA / X.509)         │
│  • Binds trusted DeviceTrustRecord to PostgreSQL RLS Context           │
└────────────────────────────────────────────────────────────────────────┘
```

1. **Hardware Interaction (Client-Side):** The field app communicates with the TPM or YubiKey using native OS platform channels (Windows CNG Platform Crypto Provider, Linux `/dev/tpmrm0`, or standalone Rust CLI utilities). The client extracts the attestation certificate/quote, signs the server challenge nonce, and packs standard JSON.
2. **Server Verification (Pure Go):** The Go monolith receives the JSON claim, validates structural integrity via `AttestationClaim.Validate()`, validates origin trust in `CheckPosture()`, and binds the device identity into PostgreSQL RLS.

---

## 3. Cryptographic Specification of Origins

### 3.1 `KeyOriginTPM` (`"TPM_2_0"`)
- **Underlying Hardware:** TCG TPM 2.0 discrete security chip or firmware TPM (fTPM/PTT).
- **Attestation Evidence:** 
  - Endorsement Key (EK) certificate chain rooted in the manufacturer CA (Infineon, STMicro, Intel, AMD).
  - Attestation Identity Key (AIK) quote verifying PCR states (secure boot integrity).
- **Security Rating:** **FIPS 140-3 Level 2** (tamper evidence, role-based authentication).
- **Binding Requirement:** Windows Hello PIN/Biometric or TPM authorization policy session.

### 3.2 `KeyOriginYubiKey` (`"YUBIKEY_PIV"`)
- **Underlying Hardware:** YubiKey 5 Series FIPS / YubiKey 5 NFC FIPS (NIST SP 800-73-4 interface).
- **Attestation Evidence:**
  - PIV Slot 9a (Authentication) or 9c (Digital Signature) certificate.
  - Slot `0xf9` Attestation Statement signed by the Yubico PIV Attestation Intermediate CA.
- **Security Rating:** **FIPS 140-3 Level 3** (physical tamper resistance, identity-based authentication).
- **Binding Requirement:** PIN verification + physical user presence touch sensor (biometric touch on YubiKey Bio).

---

## 4. Contract Implementation Map (`pkg/onboarding`)

| Go Type / Identifier | Value | Classification | FIPS 140-3 Compliance |
|:---|:---|:---|:---:|
| `KeyOriginSecureEnclave` | `"SECURE_ENCLAVE"` | Hardware (Apple SEP) | Level 2 (with Biometric) |
| `KeyOriginStrongBox` | `"STRONGBOX"` | Hardware (Android Chip) | Level 3 (with Biometric) |
| `KeyOriginTEE` | `"TEE"` | Hardware (ARM TrustZone) | Non-compliant (No discrete chip) |
| **`KeyOriginTPM`** | **`"TPM_2_0"`** | **Hardware (Discrete TPM)** | **Level 2 (with PIN/Presence)** |
| **`KeyOriginYubiKey`** | **`"YUBIKEY_PIV"`** | **Hardware (SmartCard/Token)** | **Level 3 (with PIN/Presence)** |
| `KeyOriginSoftware` | `"SOFTWARE"` | Software (Ed25519) | Non-compliant |
| `KeyOriginNone` | `"NONE"` | Unattested | Non-compliant |

---

## 5. Conclusion & Ratification

The addition of `KeyOriginTPM` and `KeyOriginYubiKey` completes the industrial hardware assurance matrix for rugged enterprise field deployments. All operations remain strictly in pure Go (`CGO_ENABLED=0`), preserving zero-dependency single-binary builds and absolute sovereign isolation.
