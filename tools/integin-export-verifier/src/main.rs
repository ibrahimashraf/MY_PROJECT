//! integin-export-verifier — read-only Export Manifest v1 verifier.
//!
//! Validates an evidence export manifest produced by
//! `pkg/evidenceexport` (Go) against the exact canonical payload contract:
//!
//! 1. Structural checks (object count, total byte length, non-empty fields,
//!    valid SHA-256 digests).
//! 2. Rebuilds the canonical JSON payload byte-for-byte as the Go
//!    `canonicalPayload()` does (sorted `evidence_items`, RFC3339Nano
//!    timestamps, exact key order).
//! 3. Asserts `manifest_sha256 == SHA-256(canonicalPayload)`.
//! 4. Verifies `export_signature` over the canonical payload with Ed25519.
//!
//! Completely offline: no database credentials, no network calls.

use std::env;
use std::fmt::Write as _;
use std::fs;
use std::io::Read as _;
use std::path::{Path, PathBuf};
use std::process::ExitCode;

use base64::{
    engine::general_purpose::{STANDARD as B64_STD, URL_SAFE_NO_PAD as B64_URL},
    Engine as _,
};
use ed25519_dalek::{Signature, VerifyingKey};
use serde::Deserialize;
use sha2::{Digest, Sha256};

const MANIFEST_VERSION: &str = "1.0";
const SHA256_HEX_LEN: usize = 64;
const EXIT_OK: u8 = 0;
const EXIT_VERIFY_FAILED: u8 = 1;
const EXIT_USAGE: u8 = 2;

/// Error carrying a stage-tagged failure message.
type Error = String;

// ---------------------------------------------------------------------------
// Manifest schema (mirrors pkg/evidenceexport/manifest.go)
// ---------------------------------------------------------------------------

#[derive(Debug, Deserialize)]
struct Manifest {
    manifest_version: String,
    export_id: String,
    tenant_id: String,
    organization_id: String,
    created_at: String,
    exporter_identity: ExporterIdentity,
    procedure_version: String,
    evidence_items: Vec<EvidenceItem>,
    object_count: usize,
    total_byte_length: u64,
    manifest_sha256: String,
    export_signature: String,
}

#[derive(Debug, Deserialize)]
struct ExporterIdentity {
    user_id: String,
    role: String,
    client_version: String,
}

#[derive(Debug, Deserialize, Clone)]
struct EvidenceItem {
    evidence_id: String,
    object_key: String,
    content_type: String,
    captured_at: String,
    plaintext_sha256: String,
    ciphertext_sha256: String,
    byte_length: i64,
    encryption_algorithm: String,
    authority_device_id: String,
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

fn main() -> ExitCode {
    let args: Vec<String> = env::args().collect();
    let parsed = match parse_args(&args) {
        Ok(parsed) => parsed,
        Err(()) => return usage(),
    };
    match run(&parsed.0, &parsed.1) {
        Ok(checks) => {
            for line in &checks {
                eprintln!("[ok] {line}");
            }
            eprintln!("[ok] manifest verified: canonical payload and Ed25519 signature are valid");
            ExitCode::from(EXIT_OK)
        }
        Err(msg) => {
            eprintln!("[fail] {msg}");
            ExitCode::from(EXIT_VERIFY_FAILED)
        }
    }
}

fn usage() -> ExitCode {
    eprintln!(
        "usage: integin-export-verifier --manifest <manifest.json> --public-key <hex|file|-|->"
    );
    ExitCode::from(EXIT_USAGE)
}

fn run(manifest_path: &str, key_arg: &str) -> Result<Vec<String>, Error> {
    let raw = fs::read_to_string(manifest_path)
        .map_err(|e| format!("cannot read {}: {e}", manifest_path))?;
    let manifest: Manifest = serde_json::from_str(&raw)
        .map_err(|e| format!("manifest is not valid Export Manifest v1 JSON: {e}"))?;

    let public_key = load_public_key(&key_arg)?;
    let verifying_key = VerifyingKey::from_bytes(&public_key)
        .map_err(|e| format!("public key is not a valid Ed25519 key: {e}"))?;

    let mut checks = Vec::new();
    validate_structure(&manifest).map_err(|e| format!("structural check failed: {e}"))?;
    checks.push("structure: object_count, total_byte_length, fields, sha256 digests".into());

    let canonical = canonical_payload(&manifest)?;
    checks.push(format!(
        "canonical payload rebuilt ({} bytes, evidence_items sorted, RFC3339Nano)",
        canonical.len()
    ));

    let digest = Sha256::digest(&canonical);
    let digest_hex = hex::encode(digest);
    let declared = manifest.manifest_sha256.trim().to_ascii_lowercase();
    if declared.is_empty() {
        return Err("manifest_sha256 is required".into());
    }
    if declared != digest_hex {
        return Err(format!(
            "manifest_sha256 mismatch: manifest declares {declared}, canonical payload hashes to {digest_hex}"
        ));
    }
    checks.push(format!("manifest_sha256 matches SHA-256(canonicalPayload): {digest_hex}"));

    if manifest.export_signature.trim().is_empty() {
        return Err("export_signature is required".into());
    }
    let sig_bytes = decode_signature(&manifest.export_signature)?;
    if sig_bytes.len() != 64 {
        return Err(format!(
            "signature length = {}, want 64",
            sig_bytes.len()
        ));
    }
    let signature = Signature::try_from(sig_bytes.as_slice())
        .map_err(|_| "signature bytes do not form a valid ed25519 signature".to_string())?;
    verifying_key
        .verify_strict(&canonical, &signature)
        .map_err(|_| "export_signature verification failed".to_string())?;
    checks.push("export_signature verified with Ed25519".into());

    Ok(checks)
}

// ---------------------------------------------------------------------------
// Argument handling
// ---------------------------------------------------------------------------

fn parse_args(args: &[String]) -> Result<(String, String), ()> {
    let mut manifest: Option<String> = None;
    let mut key: Option<String> = None;
    let mut i = 1;
    while i < args.len() {
        match args[i].as_str() {
            "--manifest" => {
                i += 1;
                if i >= args.len() {
                    return Err(());
                }
                manifest = Some(args[i].clone());
            }
            "--public-key" => {
                i += 1;
                if i >= args.len() {
                    return Err(());
                }
                key = Some(args[i].clone());
            }
            _ => return Err(()),
        }
        i += 1;
    }
    match (manifest, key) {
        (Some(m), Some(k)) => Ok((m, k)),
        _ => Err(()),
    }
}

/// Resolves the Ed25519 public key argument: a hex string, a path to a file
/// containing hex, or `-`/stdin.
fn load_public_key(arg: &str) -> Result<[u8; 32], Error> {
    let contents = if arg == "-" {
        let mut buf = String::new();
        std::io::stdin()
            .read_to_string(&mut buf)
            .map_err(|e| format!("cannot read public key from stdin: {e}"))?;
        buf
    } else if Path::new(arg).is_file() {
        fs::read_to_string(arg)
            .map_err(|e| format!("cannot read public key file {arg}: {e}"))?
    } else {
        arg.to_string()
    };

    let cleaned = contents.trim().strip_prefix("0x").unwrap_or(contents.trim());
    let compact: String = cleaned.chars().filter(|c| !c.is_whitespace()).collect();
    let bytes = hex::decode(&compact)
        .map_err(|e| format!("public key {arg} is not valid hex: {e}"))?;
    let fixed: [u8; 32] = bytes
        .try_into()
        .map_err(|_| format!("public key length = {}, want 64 hex chars (32 bytes)", bytes.len() * 2))?;
    Ok(fixed)
}

// ---------------------------------------------------------------------------
// Structural validation
// ---------------------------------------------------------------------------

fn validate_structure(m: &Manifest) -> Result<(), Error> {
    for (field, value) in [
        ("manifest_version", m.manifest_version.as_str()),
        ("export_id", m.export_id.as_str()),
        ("tenant_id", m.tenant_id.as_str()),
        ("organization_id", m.organization_id.as_str()),
        ("procedure_version", m.procedure_version.as_str()),
    ] {
        if value.trim().is_empty() {
            return Err(format!("{field} is required"));
        }
    }
    if m.manifest_version != MANIFEST_VERSION {
        return Err(format!(
            "manifest_version = {}, want {MANIFEST_VERSION}",
            m.manifest_version
        ));
    }
    for (field, value) in [
        ("user_id", m.exporter_identity.user_id.as_str()),
        ("role", m.exporter_identity.role.as_str()),
        ("client_version", m.exporter_identity.client_version.as_str()),
    ] {
        if value.trim().is_empty() {
            return Err(format!("exporter {field} is required"));
        }
    }
    if m.created_at.trim().is_empty() {
        return Err("created_at is required".into());
    }
    rfc3339nano(&m.created_at)?;

    if m.object_count != m.evidence_items.len() {
        return Err(format!(
            "object_count = {}, want {} (len evidence_items)",
            m.object_count,
            m.evidence_items.len()
        ));
    }

    let mut total: u64 = 0;
    let mut ids = std::collections::HashSet::new();
    let mut keys = std::collections::HashSet::new();
    for item in &m.evidence_items {
        validate_evidence_item(item, &mut ids, &mut keys)?;
        total += item.byte_length as u64;
    }
    if m.total_byte_length != total {
        return Err(format!(
            "total_byte_length = {}, want {} (sum of evidence byte_lengths)",
            m.total_byte_length, total
        ));
    }
    Ok(())
}

fn validate_evidence_item(
    item: &EvidenceItem,
    ids: &mut std::collections::HashSet<String>,
    keys: &mut std::collections::HashSet<String>,
) -> Result<(), Error> {
    for (field, value) in [
        ("evidence_id", item.evidence_id.as_str()),
        ("object_key", item.object_key.as_str()),
        ("content_type", item.content_type.as_str()),
        ("plaintext_sha256", item.plaintext_sha256.as_str()),
        ("ciphertext_sha256", item.ciphertext_sha256.as_str()),
        ("encryption_algorithm", item.encryption_algorithm.as_str()),
        ("authority_device_id", item.authority_device_id.as_str()),
    ] {
        if value.trim().is_empty() {
            return Err(format!("evidence {} is required", field));
        }
    }
    if item.captured_at.trim().is_empty() {
        return Err(format!("evidence {} captured_at is required", item.evidence_id));
    }
    rfc3339nano(&item.captured_at)?;
    if item.byte_length < 0 {
        return Err(format!(
            "evidence {} byte_length = {}, want >= 0",
            item.evidence_id, item.byte_length
        ));
    }
    is_sha256_hex(&item.plaintext_sha256)
        .map_err(|e| format!("evidence {} plaintext_sha256: {e}", item.evidence_id))?;
    is_sha256_hex(&item.ciphertext_sha256)
        .map_err(|e| format!("evidence {} ciphertext_sha256: {e}", item.evidence_id))?;
    if item.object_key.contains("..") {
        return Err(format!(
            "evidence {} object_key must not contain traversal segments",
            item.evidence_id
        ));
    }
    if !ids.insert(item.evidence_id.clone()) {
        return Err(format!("duplicate evidence_id {}", item.evidence_id));
    }
    if !keys.insert(item.object_key.clone()) {
        return Err(format!("duplicate object_key {}", item.object_key));
    }
    Ok(())
}

fn is_sha256_hex(s: &str) -> Result<(), Error> {
    if s.len() != SHA256_HEX_LEN {
        return Err(format!(
            "must be {} hex characters, got {}",
            SHA256_HEX_LEN,
            s.len()
        ));
    }
    hex::decode(s).map_err(|e| format!("must be valid hex: {e}"))?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Canonical payload (byte-exact Go json.Marshal semantics)
// ---------------------------------------------------------------------------

/// Rebuilds the canonical payload exactly as Go's `canonicalPayload()` does:
/// sorted `evidence_items` and the fixed struct field order above.
fn canonical_payload(m: &Manifest) -> Result<Vec<u8>, Error> {
    let mut items = m.evidence_items.clone();
    items.sort_by(|a, b| {
        a.evidence_id
            .cmp(&b.evidence_id)
            .then(a.object_key.cmp(&b.object_key))
    });

    let mut out = Vec::new();
    out.push(b'{');
    string_key(&mut out, "manifest_version");
    string_value(&mut out, &m.manifest_version);
    out.extend(b",");
    string_key(&mut out, "export_id");
    string_value(&mut out, &m.export_id);
    out.extend(b",");
    string_key(&mut out, "tenant_id");
    string_value(&mut out, &m.tenant_id);
    out.extend(b",");
    string_key(&mut out, "organization_id");
    string_value(&mut out, &m.organization_id);
    out.extend(b",");
    string_key(&mut out, "created_at");
    string_value(&mut out, &rfc3339nano(&m.created_at)?);
    out.extend(b",");
    string_key(&mut out, "exporter_identity");
    out.push(b'{');
    string_key(&mut out, "user_id");
    string_value(&mut out, &m.exporter_identity.user_id);
    out.extend(b",");
    string_key(&mut out, "role");
    string_value(&mut out, &m.exporter_identity.role);
    out.extend(b",");
    string_key(&mut out, "client_version");
    string_value(&mut out, &m.exporter_identity.client_version);
    out.push(b'}');
    out.extend(b",");
    string_key(&mut out, "procedure_version");
    string_value(&mut out, &m.procedure_version);
    out.extend(b",");
    string_key(&mut out, "evidence_items");
    out.push(b'[');
    for (i, item) in items.iter().enumerate() {
        if i > 0 {
            out.push(b',');
        }
        out.push(b'{');
        string_key(&mut out, "evidence_id");
        string_value(&mut out, &item.evidence_id);
        out.extend(b",");
        string_key(&mut out, "object_key");
        string_value(&mut out, &item.object_key);
        out.extend(b",");
        string_key(&mut out, "content_type");
        string_value(&mut out, &item.content_type);
        out.extend(b",");
        string_key(&mut out, "captured_at");
        string_value(&mut out, &rfc3339nano(&item.captured_at)?);
        out.extend(b",");
        string_key(&mut out, "plaintext_sha256");
        string_value(&mut out, &item.plaintext_sha256);
        out.extend(b",");
        string_key(&mut out, "ciphertext_sha256");
        string_value(&mut out, &item.ciphertext_sha256);
        out.extend(b",");
        string_key(&mut out, "byte_length");
        write!(out, "{}", item.byte_length).map_err(|e: std::fmt::Error| e.to_string())?;
        out.extend(b",");
        string_key(&mut out, "encryption_algorithm");
        string_value(&mut out, &item.encryption_algorithm);
        out.extend(b",");
        string_key(&mut out, "authority_device_id");
        string_value(&mut out, &item.authority_device_id);
        out.push(b'}');
    }
    out.push(b']');
    out.extend(b",");
    string_key(&mut out, "object_count");
    write!(out, "{}", m.object_count).map_err(|e: std::fmt::Error| e.to_string())?;
    out.extend(b",");
    string_key(&mut out, "total_byte_length");
    write!(out, "{}", m.total_byte_length).map_err(|e: std::fmt::Error| e.to_string())?;
    out.push(b'}');
    Ok(out)
}

fn string_key(out: &mut Vec<u8>, key: &str) {
    out.push(b'"');
    out.extend_from_slice(key.as_bytes());
    out.extend_from_slice(b"\":");
}

/// Writes a JSON string using Go `encoding/json` escaping: quotes, backslash,
/// control characters, and HTML escapes for `<`, `>`, `&`, U+2028, U+2029.
fn string_value(out: &mut Vec<u8>, value: &str) {
    out.push(b'"');
    for ch in value.chars() {
        match ch {
            '"' => out.extend_from_slice(b"\\\""),
            '\\' => out.extend_from_slice(b"\\\\"),
            '\n' => out.extend_from_slice(b"\\n"),
            '\r' => out.extend_from_slice(b"\\r"),
            '\t' => out.extend_from_slice(b"\\t"),
            '\u{0008}' => out.extend_from_slice(b"\\b"),
            '\u{000c}' => out.extend_from_slice(b"\\f"),
            '<' => out.extend_from_slice(b"\\u003c"),
            '>' => out.extend_from_slice(b"\\u003e"),
            '&' => out.extend_from_slice(b"\\u0026"),
            '\u{2028}' => out.extend_from_slice(b"\\u2028"),
            '\u{2029}' => out.extend_from_slice(b"\\u2029"),
            ch if (ch as u32) < 0x20 => {
                write!(out, "\\u{:04x}", ch as u32).unwrap();
            }
            ch => {
                let mut buf = [0u8; 4];
                out.extend_from_slice(ch.encode_utf8(&mut buf).as_bytes());
            }
        }
    }
    out.push(b'"');
}

/// Normalizes an RFC3339 timestamp to Go `time.Format(time.RFC3339Nano)`
/// output: uppercase `T`/`Z`, trailing-zero-trimmed fraction (omitted when
/// zero), and `Z` emitted for any zero offset.
fn rfc3339nano(raw: &str) -> Result<String, Error> {
    let bytes = raw.as_bytes();
    if bytes.len() < 20 {
        return Err(format!("timestamp {raw:?} is too short for RFC3339"));
    }
    if bytes[4] != b'-' || bytes[7] != b'-' || bytes[13] != b':' || bytes[16] != b':' {
        return Err(format!("timestamp {raw:?} has malformed separators"));
    }
    if bytes[10] != b'T' && bytes[10] != b't' {
        return Err(format!("timestamp {raw:?} must separate date and time with 'T'"));
    }

    let year: i64 = raw[0..4].parse().map_err(|_| format!("bad year in {raw:?}"))?;
    let month: i64 = raw[5..7].parse().map_err(|_| format!("bad month in {raw:?}"))?;
    let day: i64 = raw[8..10].parse().map_err(|_| format!("bad day in {raw:?}"))?;
    let hour: i64 = raw[11..13].parse().map_err(|_| format!("bad hour in {raw:?}"))?;
    let min: i64 = raw[14..16].parse().map_err(|_| format!("bad minute in {raw:?}"))?;
    let sec: i64 = raw[17..19].parse().map_err(|_| format!("bad second in {raw:?}"))?;
    if year < 1 || !(1..=12).contains(&month) || !(1..=31).contains(&day) {
        return Err(format!("date out of range in {raw:?}"));
    }
    if hour > 23 || min > 59 || sec > 60 {
        return Err(format!("time out of range in {raw:?}"));
    }

    let mut rest = &raw[19..];
    let mut fraction = "";
    if let Some(stripped) = rest.strip_prefix('.') {
        let digit_end = stripped
            .find(|c: char| !c.is_ascii_digit())
            .unwrap_or(stripped.len());
        if digit_end == 0 {
            return Err(format!("empty fraction in {raw:?}"));
        }
        fraction = &stripped[..digit_end];
        rest = &rest[1 + digit_end..];
    }

    let zone = if rest == "Z" || rest == "z" {
        "Z".to_string()
    } else if rest.len() == 6 && (rest.starts_with('+') || rest.starts_with('-')) && rest.as_bytes()[3] == b':' {
        let hh: i64 = rest[1..3].parse().map_err(|_| format!("bad hour offset in {raw:?}"))?;
        let mm: i64 = rest[4..6].parse().map_err(|_| format!("bad minute offset in {raw:?}"))?;
        if hh > 23 || mm > 59 {
            return Err(format!("zone offset out of range in {raw:?}"));
        }
        if hh == 0 && mm == 0 {
            "Z".to_string()
        } else {
            format!("{}{:02}:{:02}", &rest[0..1], hh, mm)
        }
    } else {
        return Err(format!("timestamp {raw:?} has no valid timezone"));
    };

    let trimmed = fraction.trim_end_matches('0');
    let mut out = format!("{}T{}", &raw[0..10], &raw[11..19]);
    if !trimmed.is_empty() {
        out.push('.');
        out.push_str(trimmed);
    }
    out.push_str(&zone);
    Ok(out)
}

// ---------------------------------------------------------------------------
// Signature decoding
// ---------------------------------------------------------------------------

/// Decodes the signature byte string: URL-safe raw base64 first (matching Go's
/// `RawURLEncoding`), then standard padded base64.
fn decode_signature(encoded: &str) -> Result<Vec<u8>, Error> {
    let trimmed = encoded.trim();
    if let Ok(bytes) = B64_URL.decode(trimmed) {
        return Ok(bytes);
    }
    B64_STD
        .decode(trimmed)
        .map_err(|_| "export_signature is not valid base64 (URL-safe or standard)".to_string())
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    fn fixture_dir() -> PathBuf {
        PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("testdata")
    }

    fn golden_manifest() -> Manifest {
        let raw =
            fs::read_to_string(fixture_dir().join("manifest.json")).expect("golden manifest.json");
        serde_json::from_str(&raw).expect("golden manifest parses")
    }

    #[test]
    fn golden_manifest_verifies() {
        let manifest = golden_manifest();
        let key = fs::read_to_string(fixture_dir().join("public_key.hex"))
            .expect("golden public_key.hex")
            .trim()
            .to_string();
        let public_key =
            load_public_key(&key).expect("public key loads from inline hex");
        let verifying_key =
            VerifyingKey::from_bytes(&public_key).expect("valid ed25519 key");

        validate_structure(&manifest).expect("structure valid");
        let canonical = canonical_payload(&manifest).expect("canonical payload builds");
        let digest = hex::encode(Sha256::digest(&canonical));
        assert_eq!(
            manifest.manifest_sha256.to_ascii_lowercase(),
            digest,
            "canonical payload must hash to the Go-computed manifest_sha256"
        );
        let signature = Signature::try_from(decode_signature(&manifest.export_signature).unwrap().as_slice())
            .unwrap();
        verifying_key
            .verify_strict(&canonical, &signature)
            .expect("signature verifies over Rust-rebuilt canonical payload");
    }

    #[test]
    fn canonical_sorting_matches_go() {
        let manifest = golden_manifest();
        let canonical = canonical_payload(&manifest).unwrap();
        let text = String::from_utf8(canonical).unwrap();
        let first = text.find("EV-1001").expect("EV-1001 present");
        let second = text.find("EV-1002").expect("EV-1002 present");
        let third = text.find("EV-1003").expect("EV-1003 present");
        assert!(first < second && second < third, "evidence must sort by evidence_id");
        assert!(text.starts_with(
            "{\"manifest_version\":\"1.0\",\"export_id\":\""
        ), "exact key order starts with manifest_version/export_id");
    }

    #[test]
    fn tampered_total_byte_length_fails() {
        let mut manifest = golden_manifest();
        manifest.total_byte_length += 1;
        assert!(validate_structure(&manifest).is_err());
    }

    #[test]
    fn tampered_object_count_fails() {
        let mut manifest = golden_manifest();
        manifest.object_count = manifest.evidence_items.len() + 1;
        assert!(validate_structure(&manifest).is_err());
    }

    #[test]
    fn tampered_digest_fails() {
        let manifest = golden_manifest();
        let mut canonical = canonical_payload(&manifest).unwrap();
        *canonical.last_mut().unwrap() = b' ';
        let digest = hex::encode(Sha256::digest(&canonical));
        assert_ne!(digest, manifest.manifest_sha256.to_ascii_lowercase());
    }

    #[test]
    fn rfc3339nano_normalization() {
        assert_eq!(rfc3339nano("2026-08-21T09:30:15.123456789Z").unwrap(), "2026-08-21T09:30:15.123456789Z");
        assert_eq!(rfc3339nano("2026-08-21T09:30:15.250000000Z").unwrap(), "2026-08-21T09:30:15.25Z");
        assert_eq!(rfc3339nano("2026-08-21T09:30:15.000000000Z").unwrap(), "2026-08-21T09:30:15Z");
        assert_eq!(rfc3339nano("2026-08-21T09:30:15z").unwrap(), "2026-08-21T09:30:15Z");
        assert_eq!(rfc3339nano("2026-08-21T09:30:15+00:00").unwrap(), "2026-08-21T09:30:15Z");
        assert_eq!(rfc3339nano("2026-08-21T09:30:15+05:30").unwrap(), "2026-08-21T09:30:15+05:30");
        assert_eq!(rfc3339nano("2026-08-21t09:30:15.5z").unwrap(), "2026-08-21T09:30:15.5Z");
        assert!(rfc3339nano("not-a-timestamp").is_err());
    }

    #[test]
    fn signature_decode_accepts_both_encodings() {
        let manifest = golden_manifest();
        let sig = decode_signature(&manifest.export_signature)
            .expect("golden signature decodes");
        assert_eq!(sig.len(), 64);
        let url: String = B64_URL.encode(&sig);
        let std: String = B64_STD.encode(&sig);
        assert_eq!(decode_signature(&url).unwrap(), sig);
        assert_eq!(decode_signature(&std).unwrap(), sig);
        assert!(decode_signature("not-an-encoding!!!").is_err());
    }
}