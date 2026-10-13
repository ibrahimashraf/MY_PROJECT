import 'dart:convert';

import '../storage/persistent_outbox.dart';
import 'rule_bundle.dart';

/// Local storage and cache for dynamic rule bundles with Tenant Shadowing.
/// Reuses the existing [KeyValueStore] abstraction without modifying Drift schemas.
class RuleBundleStore {
  RuleBundleStore({required KeyValueStore storage}) : _storage = storage;

  static const _storagePrefix = 'integin.assurance.rule_bundles.v1.';
  final KeyValueStore _storage;

  /// Builds a storage key. Tenant-specific bundles are partitioned under tenant ID.
  String _buildKey(String bundleId, {String? tenantId}) {
    final tenantSegment = tenantId != null && tenantId.trim().isNotEmpty
        ? 'tenant_$tenantId'
        : 'global';
    return '$_storagePrefix${bundleId}_$tenantSegment';
  }

  /// Saves a rule bundle to local offline storage.
  Future<void> saveBundle(RuleBundle bundle) async {
    final key = _buildKey(bundle.bundleId, tenantId: bundle.tenantId);
    await _storage.write(key, bundle.toJsonString());
  }

  /// Retrieves a rule bundle implementing strict Tenant Priority Shadowing:
  /// 1. Tries to find a tenant-specific override first.
  /// 2. If absent, falls back to the global baseline standard.
  Future<RuleBundle?> getEffectiveBundle(
    String bundleId, {
    String? tenantId,
  }) async {
    // 1. Try tenant-specific override
    if (tenantId != null && tenantId.trim().isNotEmpty) {
      final tenantKey = _buildKey(bundleId, tenantId: tenantId);
      final rawTenantJson = await _storage.read(tenantKey);
      if (rawTenantJson != null && rawTenantJson.isNotEmpty) {
        return RuleBundle.fromJson(
            jsonDecode(rawTenantJson) as Map<String, dynamic>);
      }
    }

    // 2. Fallback to global bundle
    final globalKey = _buildKey(bundleId);
    final rawGlobalJson = await _storage.read(globalKey);
    if (rawGlobalJson != null && rawGlobalJson.isNotEmpty) {
      return RuleBundle.fromJson(
          jsonDecode(rawGlobalJson) as Map<String, dynamic>);
    }

    return null;
  }
}
