# 🏛️ INTEGIN Independent Export Verifier — Regulatory Audit Guide

**Document Classification:** Judicial & Regulatory Operating Standard  
**Tool Target:** `integin-export-verifier` (Single-Binary Distribution)  
**Standard Compliance:** US Federal Rules of Evidence Rule 902(13)/(14), ISO/IEC 17020:2012, Daubert Evidentiary Standard  

---

## 1. Assurance & Legal Defensibility Model

The `integin-export-verifier` is an **air-gapped, read-only forensic utility** designed specifically for:
- Statutory bodies (OSHA, Saudi SASO, UK HSE)
- Third-party inspection accreditation authorities (Lloyd's Register, DNV, Bureau Veritas)
- Judicial fact-finders and courtroom forensic examiners

### Inviolable Trust Principles
1. **Zero Network Connectivity:** The verifier executes zero HTTP/RPC requests, binds no network ports, and queries no external DNS.
2. **Zero Database Credentials:** Operates 100% offline from local disk files or read-only storage media without database connections.
3. **Strict Cryptographic Falsifiability:** Employs strict Ed25519 verification (`verify_strict`) and byte-for-byte canonical JSON reconstruction to verify data authenticity.

---

## 2. Binary Integrity Verification

Before executing an audit, verify that your local binary matches the published SHA-256 release digest in `SHA256SUMS.txt`:

### Windows (PowerShell)
```powershell
Get-FileHash -Algorithm SHA256 ./integin-export-verifier.exe
```

### Linux / macOS
```bash
sha256sum ./integin-export-verifier
```

---

## 3. Verification Commands

### Mode A: Full Evidence Archive Verification (Recommended)
Validates the complete sealed package, including all embedded photos, sensor streams, calibration certificates, and the cryptographic manifest:

```bash
integin-export-verifier \
  --archive <export_bundle.zip> \
  --public-key <authority_public_key_hex_or_file>
```

### Mode B: Metadata-Only Manifest Verification
Validates the structural manifest schema, canonical payload order, and Ed25519 signature:

```bash
integin-export-verifier \
  --manifest <manifest.json> \
  --public-key <authority_public_key_hex_or_file>
```

### Mode C: Manifest + Local Blobs Directory
Validates manifest metadata alongside unpacked physical evidence files stored in a directory:

```bash
integin-export-verifier \
  --manifest <manifest.json> \
  --public-key <authority_public_key_hex_or_file> \
  --evidence-dir <path_to_unpacked_blobs>
```

---

## 4. Exit Codes & Evidentiary Disposition

| Exit Code | Disposition | Forensic Meaning | Courtroom Finding |
|:---:|:---|:---|:---|
| **`0`** | **`VERIFIED_AUTHENTIC`** | Canonical JSON matched, SHA-256 digests identical across all objects, strict Ed25519 signature valid. | **Tamper-Free Record Admissible** |
| **`1`** | **`VERIFICATION_FAILED`** | Digest mismatch, signature rejection, altered byte lengths, or missing objects. | **REJECTED: Evidence Tampered / Corrupt** |
| **`2`** | **`USAGE_ERROR`** | Missing arguments, unreadable file, or invalid public key format. | Operational Fault |
