import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/security/hardware_attestation_provider.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final tenantContext = TenantContext(
    tenantId: 'tenant-apex-1',
    organizationId: 'org-apex-1',
    environment: 'LIVE',
  );

  test('OfflineMutation with FIPS 140-3 StrongBox HardwareAttestationClaim round-trips correctly', () {
    final claim = HardwareAttestationClaim(
      keyOrigin: keyOriginStrongBox,
      biometricBound: true,
      fips140_3Compliant: true,
      fipsLevel: 'FIPS_140_3_LEVEL_3',
      deviceFingerprint: 'android-hw-fips-sn9982',
      hardwareSignature: 'a1b2c3d4e5f6',
      attestationCertificateChain: [
        '-----BEGIN CERTIFICATE-----\nLEAF\n-----END CERTIFICATE-----',
        '-----BEGIN CERTIFICATE-----\nROOT\n-----END CERTIFICATE-----',
      ],
    );

    final mutation = OfflineMutation(
      transactionId: 'tx-fips-001',
      context: tenantContext,
      deviceId: 'dev-samsung-tabactive4',
      userId: 'insp-tariq-01',
      sequenceNumber: 1,
      operation: 'InspectionSubmitted',
      entityId: 'insp-rec-988',
      payload: {'crane_swl': '100T', 'result': 'PASSED'},
      capturedAt: DateTime.parse('2026-10-04T12:00:00Z'),
      authorityId: 'auth-01',
      authorityEpoch: 7,
      signature: 'device-sig-ed25519-valid',
      hardwareAttestation: claim,
      biometricVerified: true,
    );

    final json = mutation.toJson();
    expect(json['biometric_verified'], isTrue);
    expect(json['hardware_attestation'], isNotNull);

    final hardwareMap = json['hardware_attestation'] as Map<String, Object?>;
    expect(hardwareMap['key_origin'], keyOriginStrongBox);
    expect(hardwareMap['biometric_bound'], isTrue);
    expect(hardwareMap['fips_140_3_compliant'], isTrue);
    expect(hardwareMap['fips_level'], 'FIPS_140_3_LEVEL_3');
    expect(hardwareMap['attestation_certificate_chain'], hasLength(2));

    final deserialized = OfflineMutation.fromJson(json);
    expect(deserialized.transactionId, 'tx-fips-001');
    expect(deserialized.biometricVerified, isTrue);
    expect(deserialized.hardwareAttestation, isNotNull);
    expect(deserialized.hardwareAttestation!.keyOrigin, keyOriginStrongBox);
    expect(deserialized.hardwareAttestation!.fips140_3Compliant, isTrue);
    expect(deserialized.hardwareAttestation!.fipsLevel, 'FIPS_140_3_LEVEL_3');
  });

  test('OfflineMutation with Apple Secure Enclave FIPS 140-3 Level 2 round-trips correctly', () {
    final claim = HardwareAttestationClaim(
      keyOrigin: keyOriginSecureEnclave,
      biometricBound: true,
      fips140_3Compliant: true,
      fipsLevel: 'FIPS_140_3_LEVEL_2',
      deviceFingerprint: 'ios-hw-fips-ipadpro',
      hardwareSignature: 'b1c2d3e4f5',
    );

    final mutation = OfflineMutation(
      transactionId: 'tx-fips-002',
      context: tenantContext,
      deviceId: 'dev-ipad-pro',
      userId: 'insp-fahad-02',
      sequenceNumber: 2,
      operation: 'ProofLoadRecorded',
      entityId: 'insp-rec-989',
      payload: {'measured_load': '55.2T'},
      capturedAt: DateTime.parse('2026-10-04T12:05:00Z'),
      authorityId: 'auth-01',
      authorityEpoch: 7,
      signature: 'device-sig-enclave-valid',
      hardwareAttestation: claim,
      biometricVerified: true,
    );

    final json = mutation.toJson();
    final deserialized = OfflineMutation.fromJson(json);
    expect(deserialized.hardwareAttestation!.keyOrigin, keyOriginSecureEnclave);
    expect(deserialized.hardwareAttestation!.fipsLevel, 'FIPS_140_3_LEVEL_2');
    expect(deserialized.biometricVerified, isTrue);
  });

  test('OfflineMutation without hardware attestation preserves backwards compatibility', () {
    final mutation = OfflineMutation(
      transactionId: 'tx-legacy-003',
      context: tenantContext,
      deviceId: 'dev-legacy-tablet',
      userId: 'insp-legacy-03',
      sequenceNumber: 3,
      operation: 'FindingRecorded',
      entityId: 'finding-101',
      payload: {'item': 'hook_latch', 'status': 'pass'},
      capturedAt: DateTime.parse('2026-10-04T12:10:00Z'),
      authorityId: 'auth-01',
      authorityEpoch: 7,
      signature: 'legacy-sig',
    );

    final json = mutation.toJson();
    expect(json.containsKey('hardware_attestation'), isFalse);
    expect(json.containsKey('biometric_verified'), isFalse);

    final deserialized = OfflineMutation.fromJson(json);
    expect(deserialized.hardwareAttestation, isNull);
    expect(deserialized.biometricVerified, isFalse);
  });
}
