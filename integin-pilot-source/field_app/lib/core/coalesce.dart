import 'dart:async';

/// CoalescingController batches high-frequency user interactions (sliders, drags, typing)
/// into single journaled command payloads, eliminating event floods.
class CoalescingController {
  final Duration debounceWindow;
  String? _activeKey;
  Map<String, dynamic>? _pendingPayload;
  int _timerCounter = 0;

  CoalescingController({
    this.debounceWindow = const Duration(milliseconds: 64),
  });

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

  void flush(void Function(String key, Map<String, dynamic> batchedPayload) onCommit) {
    if (_activeKey != null && _pendingPayload != null) {
      onCommit(_activeKey!, _pendingPayload!);
      _activeKey = null;
      _pendingPayload = null;
      _timerCounter++;
    }
  }
}
