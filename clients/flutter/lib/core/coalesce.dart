// CoalescingController batches rapid interactive input (e.g. sliders, gestures)
// into single journaled command dispatches, preventing event flood.
class CoalescingController {
  final Duration debounceWindow;
  String? _activeKey;
  Map<String, dynamic>? _pendingPayload;
  int _timerCounter = 0;

  CoalescingController({
    this.debounceWindow = const Duration(milliseconds: 64), // ~15-16fps throttle
  });

  /// Submit an interactive update under a specific coalesce key (e.g. "asset.slider.load")
  void emit({
    required String coalesceKey,
    required Map<String, dynamic> payload,
    required void Function(String key, Map<String, dynamic> batchedPayload) onCommit,
  }) {
    _activeKey = coalesceKey;
    _pendingPayload = payload;
    final int currentGen = ++_timerCounter;

    Future.delayed(debounceWindow, () {
      if (_timerCounter == currentGen && _activeKey != null && _pendingPayload != null) {
        onCommit(_activeKey!, _pendingPayload!);
        _activeKey = null;
        _pendingPayload = null;
      }
    });
  }

  /// Force flush any pending coalesced payload immediately (e.g. on pointer up)
  void flush(void Function(String key, Map<String, dynamic> batchedPayload) onCommit) {
    if (_activeKey != null && _pendingPayload != null) {
      onCommit(_activeKey!, _pendingPayload!);
      _activeKey = null;
      _pendingPayload = null;
      _timerCounter++;
    }
  }
}
