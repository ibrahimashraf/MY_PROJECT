// Minimal stub for TusClient used in dynamic_form_view.dart
// Provides no functionality – just satisfies the type reference.
class TusClient {
  // Upload placeholder – returns a dummy upload ID.
  Future<String> upload({required List<int> data, required int size, required String sha256, required String contentType}) async {
    // In a real implementation, this would start an upload and return an upload ID.
    return 'dummy-upload-id';
  }

  // Complete placeholder – returns a dummy completion result.
  Future<CompletionResult> complete(String uploadId) async {
    // Return a dummy key.
    return CompletionResult(key: 'dummy-key');
  }
}

class CompletionResult {
  final String key;
  CompletionResult({required this.key});
}
