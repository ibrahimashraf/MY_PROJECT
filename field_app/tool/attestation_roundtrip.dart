import 'dart:convert';
import 'dart:io';

import 'package:integin_field_app/security/attestation.dart';

/// CLI driver for the Go Dart round-trip test (pkg/onboarding/roundtrip_test.go).
///
/// Prints exactly one line: the JSON enrollment submission. No other output.
Future<void> main(List<String> args) async {
  String take(String flag) {
    final i = args.indexOf(flag);
    if (i < 0 || i + 1 >= args.length) {
      stderr.writeln('missing value for $flag');
      exit(2);
    }
    return args[i + 1];
  }

  final challengeId = take('--challenge-id');
  final inspectorId = take('--inspector-id');
  final nonce = take('--nonce');
  final deviceModel = take('--device-model');

  final provider = SimulatedAttestationProvider();
  final bundle = await provider.attestEnrollment(
    challengeId: challengeId,
    inspectorId: inspectorId,
    nonceHex: nonce,
    deviceModel: deviceModel,
  );
  stdout.writeln(jsonEncode(buildEnrollmentSubmission(
    challengeId: challengeId,
    inspectorId: inspectorId,
    bundle: bundle,
  )));
}