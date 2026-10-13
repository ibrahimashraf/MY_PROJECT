enum GraceStatus {
  normal,
  graceActive,
  hardLocked,
}

class CheckpointLedgerEntry {
  final DateTime wallTimestamp;
  final int monotonicNanos;
  final String bootSessionId;

  const CheckpointLedgerEntry({
    required this.wallTimestamp,
    required this.monotonicNanos,
    required this.bootSessionId,
  });

  bool get isUninitialized =>
      monotonicNanos == 0 &&
      bootSessionId.isEmpty &&
      wallTimestamp.millisecondsSinceEpoch == 0;
}

class TimeHorizonToken {
  final DateTime anchorWallTime;
  final int anchorMonoNanos;
  final Duration horizonDuration;
  final Duration maxSkewAllowed;

  const TimeHorizonToken({
    required this.anchorWallTime,
    required this.anchorMonoNanos,
    required this.horizonDuration,
    required this.maxSkewAllowed,
  });
}

class TTLPolicy {
  final Duration maxRebootGrace;
  const TTLPolicy({required this.maxRebootGrace});
}

class GraceEvaluation {
  final GraceStatus status;
  final Duration remainingGrace;
  final String? blockReason;

  const GraceEvaluation({
    required this.status,
    required this.remainingGrace,
    this.blockReason,
  });
}

class GraceStateMachine {
  static GraceEvaluation evaluate({
    required DateTime currentWall,
    required int currentMonoNanos,
    required String currentBootSessionId,
    required CheckpointLedgerEntry checkpoint,
    required TimeHorizonToken token,
    required TTLPolicy policy,
  }) {
    if (checkpoint.isUninitialized) {
      return const GraceEvaluation(
        status: GraceStatus.hardLocked,
        remainingGrace: Duration.zero,
        blockReason: "Tamper detected: checkpoint ledger entry missing or uninitialized",
      );
    }

    final elapsedWall = currentWall.difference(token.anchorWallTime);
    if (elapsedWall < Duration.zero) {
      return const GraceEvaluation(
        status: GraceStatus.hardLocked,
        remainingGrace: Duration.zero,
        blockReason: "Wall clock drift exceeds statutory threshold: negative drift",
      );
    }

    if (token.maxSkewAllowed > Duration.zero &&
        elapsedWall > (token.horizonDuration + token.maxSkewAllowed)) {
      return const GraceEvaluation(
        status: GraceStatus.hardLocked,
        remainingGrace: Duration.zero,
        blockReason: "Wall clock drift exceeds statutory threshold",
      );
    }

    // Monotonic regression with same boot session = tamper, not reboot
    if (currentMonoNanos < checkpoint.monotonicNanos &&
        currentBootSessionId == checkpoint.bootSessionId) {
      return const GraceEvaluation(
        status: GraceStatus.hardLocked,
        remainingGrace: Duration.zero,
        blockReason: "Monotonic regression without reboot detected",
      );
    }

    final rebootDetected =
        currentBootSessionId != checkpoint.bootSessionId;

    if (!rebootDetected) {
      final elapsedMono = Duration(
        microseconds: (currentMonoNanos - token.anchorMonoNanos) ~/ 1000,
      );

      if (elapsedMono > token.horizonDuration) {
        return const GraceEvaluation(
          status: GraceStatus.hardLocked,
          remainingGrace: Duration.zero,
          blockReason: "Execution exceeds token horizon duration",
        );
      }

      return const GraceEvaluation(
        status: GraceStatus.normal,
        remainingGrace: Duration.zero,
      );
    }

    // Reboot detected: validate cold boot state
    if (currentBootSessionId.isEmpty) {
      return const GraceEvaluation(
        status: GraceStatus.hardLocked,
        remainingGrace: Duration.zero,
        blockReason: "Tamper detected: unverified boot session ID",
      );
    }

    // Check horizon expiry BEFORE reboot grace evaluation
    final remainingHorizon = token.horizonDuration -
        checkpoint.wallTimestamp.difference(token.anchorWallTime);
    if (remainingHorizon <= Duration.zero) {
      return const GraceEvaluation(
        status: GraceStatus.hardLocked,
        remainingGrace: Duration.zero,
        blockReason: "Horizon expired prior to device reboot",
      );
    }

    final effectiveGrace = policy.maxRebootGrace < remainingHorizon
        ? policy.maxRebootGrace
        : remainingHorizon;

    // Elapsed since reboot is measured by the kernel monotonic counter
    // directly (uptime since boot). After reboot, the monotonic clock
    // resets, so currentMonoNanos IS the elapsed time. Do NOT subtract
    // checkpoint.monotonicNanos — that would compare two different
    // boot clocks. Matches Go fieldtrust.go: rebootElapsed = time.Duration(currentMonoNanos).
    final rebootElapsed = Duration(microseconds: currentMonoNanos ~/ 1000);

    if (rebootElapsed > effectiveGrace) {
      return const GraceEvaluation(
        status: GraceStatus.hardLocked,
        remainingGrace: Duration.zero,
        blockReason: "Reboot grace window expired; re-anchor to edge appliance required",
      );
    }

    return GraceEvaluation(
      status: GraceStatus.graceActive,
      remainingGrace: effectiveGrace - rebootElapsed,
    );
  }
}
