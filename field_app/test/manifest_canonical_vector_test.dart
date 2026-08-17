import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/workpackages/package_manifest_client.dart';

void main() {
  test('signed manifest canonicalization matches shared context-order vector',
      () async {
    final file = File('../contracts/vectors/v1/manifest_canonical_v1.json');
    final vector = jsonDecode(await file.readAsString()) as Map<String, dynamic>;
    final manifest = <String, dynamic>{
      ...vector,
      'package': <String, dynamic>{},
      'signature': '',
    };
    expect(
      PackageManifestVerifier.canonicalManifest(manifest),
      vector['canonical_utf8'],
    );
  });
}
