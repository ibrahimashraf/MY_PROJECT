// INTEGIN Field cache: stores only previously approved, version-bound packages for offline use.
// The cache never approves a package or bypasses server-side completion validation.
import 'dart:convert';

import '../domain/inspection_draft.dart';
import '../storage/persistent_outbox.dart';

class CachedApprovedWorkPackage {
  const CachedApprovedWorkPackage({
    required this.workPack,
    required this.cachedAt,
    required this.expiresAt,
    required this.authorityEpoch,
    required this.signingKeyId,
  });

  final InspectionWorkPack workPack;
  final DateTime cachedAt;
  final DateTime expiresAt;
  final int authorityEpoch;
  final String signingKeyId;

  bool isUsableAt(DateTime now) =>
      now.toUtc().isBefore(expiresAt.toUtc()) && authorityEpoch > 0;

  void validateForStorage() {
    if (workPack.packageId.trim().isEmpty ||
        workPack.packageVersion <= 0 ||
        workPack.schemaVersion <= 0) {
      throw ArgumentError('work package identity and versions are required');
    }
    if (!RegExp(r'^sha256:[a-f0-9]{64}$').hasMatch(workPack.packageHash)) {
      throw ArgumentError(
          'work package hash must be a lowercase SHA-256 digest');
    }
    if (!expiresAt.isAfter(cachedAt) || authorityEpoch <= 0) {
      throw ArgumentError('work package authority validity is invalid');
    }
    if (signingKeyId.trim().isEmpty) {
      throw ArgumentError('work package signing key id is required');
    }
  }

  Map<String, Object?> toJson() => {
        'cached_at': cachedAt.toUtc().toIso8601String(),
        'expires_at': expiresAt.toUtc().toIso8601String(),
        'authority_epoch': authorityEpoch,
        'signing_key_id': signingKeyId,
        'work_package': {
          'inspection_id': workPack.inspectionId,
          'root_asset_id': workPack.rootAssetId,
          'inspection_type': workPack.inspectionType,
          'procedure_version': workPack.procedureVersion,
          'package_id': workPack.packageId,
          'package_version': workPack.packageVersion,
          'schema_version': workPack.schemaVersion,
          'package_hash': workPack.packageHash,
          'scheduled_date': workPack.scheduledDate.toUtc().toIso8601String(),
          'items': workPack.items
              .map((item) => {
                    'id': item.id,
                    'section_id': item.sectionId,
                    'prompt': item.prompt,
                    'asset_id': item.assetId,
                    'required': item.required,
                  })
              .toList(),
        },
      };

  static CachedApprovedWorkPackage fromJson(Map<String, dynamic> json) {
    final package = Map<String, dynamic>.from(json['work_package'] as Map);
    final rawItems = List<dynamic>.from(package['items'] as List);
    return CachedApprovedWorkPackage(
      workPack: InspectionWorkPack(
        inspectionId: package['inspection_id'] as String,
        rootAssetId: package['root_asset_id'] as String,
        inspectionType: package['inspection_type'] as String,
        procedureVersion: package['procedure_version'] as String,
        packageId: package['package_id'] as String,
        packageVersion: package['package_version'] as int,
        schemaVersion: package['schema_version'] as int,
        packageHash: package['package_hash'] as String,
        scheduledDate:
            DateTime.parse(package['scheduled_date'] as String).toUtc(),
        items: rawItems.map((rawItem) {
          final item = Map<String, dynamic>.from(rawItem as Map);
          return ChecklistItem(
            id: item['id'] as String,
            sectionId: item['section_id'] as String,
            prompt: item['prompt'] as String,
            assetId: item['asset_id'] as String,
            required: item['required'] as bool? ?? true,
          );
        }).toList(growable: false),
      ),
      cachedAt: DateTime.parse(json['cached_at'] as String).toUtc(),
      expiresAt: DateTime.parse(json['expires_at'] as String).toUtc(),
      authorityEpoch: json['authority_epoch'] as int,
      signingKeyId: json['signing_key_id'] as String,
    );
  }
}

class ApprovedWorkPackageCache {
  ApprovedWorkPackageCache({required KeyValueStore store}) : _store = store;

  static const _storageKey = 'integin.approved_work_packages.v1';

  final KeyValueStore _store;

  Future<void> save(CachedApprovedWorkPackage cachedPackage) async {
    cachedPackage.validateForStorage();
    final packages = await loadAll(includeExpired: true);
    final retained = packages
        .where((existing) =>
            existing.workPack.packageId != cachedPackage.workPack.packageId)
        .toList();
    retained.add(cachedPackage);
    await _store.write(
      _storageKey,
      jsonEncode(retained.map((entry) => entry.toJson()).toList()),
    );
  }

  Future<CachedApprovedWorkPackage?> load(
    String packageId, {
    DateTime? now,
  }) async {
    final current = (now ?? DateTime.now()).toUtc();
    final packages = await loadAll(includeExpired: true);
    for (final package in packages) {
      if (package.workPack.packageId == packageId &&
          package.isUsableAt(current)) {
        return package;
      }
    }
    return null;
  }

  Future<List<CachedApprovedWorkPackage>> loadAll({
    bool includeExpired = false,
    DateTime? now,
  }) async {
    final encoded = await _store.read(_storageKey);
    if (encoded == null || encoded.trim().isEmpty) {
      return const [];
    }
    final raw = jsonDecode(encoded) as List<dynamic>;
    final parsed = raw
        .map((entry) => CachedApprovedWorkPackage.fromJson(
              Map<String, dynamic>.from(entry as Map),
            ))
        .toList(growable: false);
    if (includeExpired) {
      return parsed;
    }
    final current = (now ?? DateTime.now()).toUtc();
    return parsed
        .where((entry) => entry.isUsableAt(current))
        .toList(growable: false);
  }
}
