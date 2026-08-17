import 'dart:convert';

import '../domain/inspection_draft.dart';
import '../storage/persistent_outbox.dart';

/// The locally known trust state of a previously verified approved package.
enum ApprovedWorkPackageState {
  verifiedCurrent,
  verifiedExpiring,
  expired,
}

class CachedApprovedWorkPackage {
  CachedApprovedWorkPackage({
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

  /// An immutable cache identity prevents a newer package from erasing the
  /// historic package/hash needed by another in-progress inspection draft.
  String get bindingKey =>
      '${workPack.inspectionId}\u0000${workPack.packageHash}';

  ApprovedWorkPackageState stateAt(
    DateTime now, {
    Duration refreshThreshold = const Duration(hours: 1),
  }) {
    final current = now.toUtc();
    if (!expiresAt.toUtc().isAfter(current)) {
      return ApprovedWorkPackageState.expired;
    }
    if (expiresAt.toUtc().difference(current) <= refreshThreshold) {
      return ApprovedWorkPackageState.verifiedExpiring;
    }
    return ApprovedWorkPackageState.verifiedCurrent;
  }

  bool isUsableAt(DateTime now) =>
      stateAt(now) != ApprovedWorkPackageState.expired;

  void validateForStorage() {
    if (workPack.inspectionId.trim().isEmpty ||
        workPack.packageId.trim().isEmpty ||
        workPack.packageHash.trim().isEmpty ||
        signingKeyId.trim().isEmpty ||
        authorityEpoch <= 0) {
      throw ArgumentError('cached approved work package binding is incomplete');
    }
    if (!workPack.packageHash.startsWith('sha256:') ||
        workPack.packageHash.length != 'sha256:'.length + 64) {
      throw ArgumentError('cached approved work package has invalid digest');
    }
    if (!expiresAt.toUtc().isAfter(cachedAt.toUtc())) {
      throw ArgumentError('cached approved work package expiry is invalid');
    }
  }

  Map<String, Object?> toJson() => <String, Object?>{
        'cached_at': cachedAt.toUtc().toIso8601String(),
        'expires_at': expiresAt.toUtc().toIso8601String(),
        'authority_epoch': authorityEpoch,
        'signing_key_id': signingKeyId,
        'work_package': <String, Object?>{
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
              .map(
                (item) => <String, Object?>{
                  'id': item.id,
                  'section_id': item.sectionId,
                  'prompt': item.prompt,
                  'asset_id': item.assetId,
                  'required': item.required,
                  'response_type': item.responseType.name,
                  'options': item.options,
                },
              )
              .toList(growable: false),
        },
      };

  static CachedApprovedWorkPackage fromJson(Map<String, dynamic> json) {
    final package = Map<String, dynamic>.from(json['work_package'] as Map);
    final rawItems = List<dynamic>.from(package['items'] as List);
    final cached = CachedApprovedWorkPackage(
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
            responseType: ChecklistResponseType.values.firstWhere(
              (type) => type.name == item['response_type'],
              orElse: () => ChecklistResponseType.text,
            ),
            options: List<String>.from(item['options'] as List? ?? const []),
          );
        }).toList(growable: false),
      ),
      cachedAt: DateTime.parse(json['cached_at'] as String).toUtc(),
      expiresAt: DateTime.parse(json['expires_at'] as String).toUtc(),
      authorityEpoch: json['authority_epoch'] as int,
      signingKeyId: json['signing_key_id'] as String,
    );
    cached.validateForStorage();
    return cached;
  }
}

/// Stores only previously verified packages. A write happens as a single value
/// replacement after all retained entries and the candidate have been decoded
/// and validated, so a pre-write failure leaves the prior cache intact.
class ApprovedWorkPackageCache {
  ApprovedWorkPackageCache({required KeyValueStore store}) : _store = store;

  static const _storageKey = 'integin.approved_work_packages.v1';
  final KeyValueStore _store;

  Future<void> save(CachedApprovedWorkPackage cachedPackage) async {
    cachedPackage.validateForStorage();
    final packages = await loadAll(includeExpired: true);
    final retained = packages
        .where((existing) => existing.bindingKey != cachedPackage.bindingKey)
        .toList(growable: false);
    final next = <CachedApprovedWorkPackage>[...retained, cachedPackage];
    final encoded = jsonEncode(next.map((entry) => entry.toJson()).toList());
    await _store.write(_storageKey, encoded);
  }

  Future<CachedApprovedWorkPackage?> load(
    String packageId, {
    DateTime? now,
  }) async {
    final current = (now ?? DateTime.now()).toUtc();
    final matches = (await loadAll(includeExpired: true))
        .where(
          (entry) =>
              entry.workPack.packageId == packageId &&
              entry.isUsableAt(current),
        )
        .toList()
      ..sort((left, right) => right.cachedAt.compareTo(left.cachedAt));
    return matches.isEmpty ? null : matches.first;
  }

  Future<CachedApprovedWorkPackage?> loadForInspection(
    String inspectionId, {
    String? packageHash,
    DateTime? now,
  }) async {
    final current = (now ?? DateTime.now()).toUtc();
    final matches = (await loadAll(includeExpired: true))
        .where(
          (entry) =>
              entry.workPack.inspectionId == inspectionId &&
              (packageHash == null ||
                  entry.workPack.packageHash == packageHash) &&
              entry.isUsableAt(current),
        )
        .toList()
      ..sort((left, right) => right.cachedAt.compareTo(left.cachedAt));
    return matches.isEmpty ? null : matches.first;
  }

  Future<List<CachedApprovedWorkPackage>> loadAll({
    bool includeExpired = false,
    DateTime? now,
  }) async {
    final encoded = await _store.read(_storageKey);
    if (encoded == null || encoded.trim().isEmpty) return const [];
    try {
      final raw = jsonDecode(encoded);
      if (raw is! List) throw const FormatException('cache is not a list');
      final parsed = raw
          .map(
            (entry) => CachedApprovedWorkPackage.fromJson(
              Map<String, dynamic>.from(entry as Map),
            ),
          )
          .toList(growable: false);
      if (includeExpired) return parsed;
      final current = (now ?? DateTime.now()).toUtc();
      return parsed
          .where((entry) => entry.isUsableAt(current))
          .toList(growable: false);
    } on FormatException catch (error) {
      throw StateError(
          'approved work package cache is invalid: ${error.message}');
    } on TypeError {
      throw StateError('approved work package cache is invalid');
    }
  }
}
