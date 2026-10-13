import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/assurance/rule_bundle.dart';
import 'package:integin_field_app/assurance/rule_bundle_store.dart';
import 'package:integin_field_app/storage/persistent_outbox.dart';

class _MemoryKeyValueStore implements KeyValueStore {
  final Map<String, String> _data = {};

  @override
  Future<String?> read(String key) async => _data[key];

  @override
  Future<void> write(String key, String value) async {
    _data[key] = value;
  }
}

void main() {
  group('RuleBundleStore Tenant Priority Shadowing', () {
    late _MemoryKeyValueStore storage;
    late RuleBundleStore store;

    setUp(() {
      storage = _MemoryKeyValueStore();
      store = RuleBundleStore(storage: storage);
    });

    test('retrieves global bundle when no tenant override exists', () async {
      const globalBundle = RuleBundle(
        bundleId: 'ISO4309_LIFTING_ROPES',
        version: '1.0.0',
        ruleHash: 'sha256:global',
        rules: [],
      );

      await store.saveBundle(globalBundle);

      final retrieved = await store.getEffectiveBundle(
        'ISO4309_LIFTING_ROPES',
        tenantId: 'tenant-acme',
      );

      expect(retrieved, isNotNull);
      expect(retrieved!.ruleHash, 'sha256:global');
      expect(retrieved.tenantId, isNull);
    });

    test('prioritizes tenant-specific override over global bundle', () async {
      const globalBundle = RuleBundle(
        bundleId: 'ISO4309_LIFTING_ROPES',
        version: '1.0.0',
        ruleHash: 'sha256:global',
        rules: [],
      );

      const tenantOverrideBundle = RuleBundle(
        bundleId: 'ISO4309_LIFTING_ROPES',
        version: '1.0.1-custom',
        ruleHash: 'sha256:tenant-acme-custom',
        tenantId: 'tenant-acme',
        rules: [],
      );

      // Save both global and tenant-specific bundles
      await store.saveBundle(globalBundle);
      await store.saveBundle(tenantOverrideBundle);

      // 1. ACME tenant must receive its own custom override
      final acmeBundle = await store.getEffectiveBundle(
        'ISO4309_LIFTING_ROPES',
        tenantId: 'tenant-acme',
      );
      expect(acmeBundle, isNotNull);
      expect(acmeBundle!.ruleHash, 'sha256:tenant-acme-custom');
      expect(acmeBundle.version, '1.0.1-custom');

      // 2. Another tenant without override must fallback to global
      final otherBundle = await store.getEffectiveBundle(
        'ISO4309_LIFTING_ROPES',
        tenantId: 'tenant-other',
      );
      expect(otherBundle, isNotNull);
      expect(otherBundle!.ruleHash, 'sha256:global');
    });

    test('returns null when bundle is not found', () async {
      final notFound = await store.getEffectiveBundle('UNKNOWN_BUNDLE');
      expect(notFound, isNull);
    });
  });
}
