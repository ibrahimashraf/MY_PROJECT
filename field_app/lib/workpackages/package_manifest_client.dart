// INTEGIN Field manifest binding: signed, version-bound packages only.
import 'dart:convert';

import 'package:crypto/crypto.dart';
import 'package:cryptography/cryptography.dart';

import '../domain/inspection_draft.dart';
import '../security/transaction_signer.dart';
import 'approved_work_package_cache.dart';

class SignedPackageManifest {
  SignedPackageManifest(this.json);
  final Map<String, dynamic> json;

  String get packageHash => json['package_hash'] as String;
  String get keyId => json['key_id'] as String;
  int get authorityEpoch => json['authority_epoch'] as int;
  DateTime get expiresAt =>
      DateTime.parse(json['expires_at'] as String).toUtc();
  String get inspectionId => json['inspection_id'] as String;
}

class PackageManifestVerifier {
  PackageManifestVerifier({required this.authorityPublicKey});
  final SimplePublicKey authorityPublicKey;

  Future<CachedApprovedWorkPackage> verifyAndBind(
    SignedPackageManifest manifest, {
    required DateTime now,
  }) async {
    final json = manifest.json;
    if (manifest.expiresAt.isBefore(now.toUtc()) ||
        manifest.authorityEpoch <= 0) {
      throw StateError(
          'package manifest is expired or has invalid authority epoch');
    }
    final signature = base64Decode(json['signature'] as String);
    final verified = await Ed25519().verify(
      utf8.encode(_canonicalManifest(json)),
      signature: Signature(signature, publicKey: authorityPublicKey),
    );
    if (!verified) throw StateError('package manifest signature is invalid');
    final workPack = _workPackFromManifest(json);
    if (workPack.packageHash != manifest.packageHash) {
      throw StateError('package manifest hash binding is invalid');
    }
    return CachedApprovedWorkPackage(
      workPack: workPack,
      cachedAt: now.toUtc(),
      expiresAt: manifest.expiresAt,
      authorityEpoch: manifest.authorityEpoch,
      signingKeyId: manifest.keyId,
    );
  }

  Future<CachedApprovedWorkPackage> verifyBindAndCache(
    SignedPackageManifest manifest, {
    required ApprovedWorkPackageCache cache,
    required DateTime now,
  }) async {
    final bound = await verifyAndBind(manifest, now: now);
    await cache.save(bound);
    return bound;
  }

  String _canonicalManifest(Map<String, dynamic> json) {
    final context =
        Map<String, dynamic>.from(json['assignment_context'] as Map);
    final fields = Map<String, dynamic>.from(
        context['field_asset_ids'] as Map? ?? const {});
    final pairs = fields.entries
        .map((entry) => '${entry.key}=${entry.value}')
        .toList()
      ..sort();
    final contextCanonical = [
      context['root_asset_id'],
      context['inspection_type'],
      context['procedure_version'],
      canonicalTimestamp(DateTime.parse(context['scheduled_at'] as String)),
      pairs.join(','),
    ].join('|');
    return [
      json['manifest_version'],
      json['tenant_id'],
      json['organization_id'],
      json['inspection_id'],
      json['device_id'],
      json['package_id'],
      json['package_version'],
      json['package_hash'],
      contextCanonical,
      json['schema_version'],
      json['authority_epoch'],
      canonicalTimestamp(DateTime.parse(json['issued_at'] as String)),
      canonicalTimestamp(DateTime.parse(json['expires_at'] as String)),
      json['signature_algorithm'],
      json['key_id'],
    ].join('|');
  }

  InspectionWorkPack _workPackFromManifest(Map<String, dynamic> json) {
    final package = Map<String, dynamic>.from(json['package'] as Map);
    final context =
        Map<String, dynamic>.from(json['assignment_context'] as Map);
    final assets = Map<String, dynamic>.from(
        context['field_asset_ids'] as Map? ?? const {});
    final hashPayload = <String, dynamic>{
      'id': package['id'],
      'tenant_id': package['tenant_id'],
      'organization_id': package['organization_id'],
      'template_code': package['template_code'],
      'template_version': package['template_version'],
      'package_version': package['package_version'],
      'schema_version': package['schema_version'],
      'state': package['state'],
      'sections': package['sections'],
    };
    final computed =
        'sha256:${sha256.convert(utf8.encode(jsonEncode(hashPayload)))}';
    if (computed != json['package_hash']) {
      throw StateError('approved package hash does not match manifest');
    }
    final items = <ChecklistItem>[];
    for (final rawSection in List<dynamic>.from(package['sections'] as List)) {
      final section = Map<String, dynamic>.from(rawSection as Map);
      for (final rawField in List<dynamic>.from(section['fields'] as List)) {
        final field = Map<String, dynamic>.from(rawField as Map);
        final id = field['id'] as String;
        final response = switch (field['type'] as String) {
          'number' => ChecklistResponseType.number,
          'boolean' => ChecklistResponseType.boolean,
          'choice' => ChecklistResponseType.choice,
          'pass_fail_na' => ChecklistResponseType.passFailNA,
          _ => ChecklistResponseType.text,
        };
        items.add(ChecklistItem(
            id: id,
            sectionId: section['id'] as String,
            prompt: field['prompt'] as String,
            assetId:
                assets[id] as String? ?? context['root_asset_id'] as String,
            required: field['required'] as bool? ?? true,
            responseType: response,
            options: List<String>.from(field['options'] as List? ?? const [])));
      }
    }
    return InspectionWorkPack(
      inspectionId: json['inspection_id'] as String,
      rootAssetId: context['root_asset_id'] as String,
      inspectionType: context['inspection_type'] as String,
      procedureVersion: context['procedure_version'] as String,
      packageId: json['package_id'] as String,
      packageVersion: json['package_version'] as int,
      schemaVersion: json['schema_version'] as int,
      packageHash: json['package_hash'] as String,
      scheduledDate: DateTime.parse(context['scheduled_at'] as String).toUtc(),
      items: items,
    );
  }
}
