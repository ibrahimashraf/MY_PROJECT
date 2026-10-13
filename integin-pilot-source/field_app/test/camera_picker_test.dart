import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/sync/camera_picker.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('INTEGIN.field_app/camera');

  Uint8List mockPhoto() => Uint8List.fromList(jpegHeaderBytes);

  void mockChannel(Object? Function(MethodCall) handler) {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
      expect(call.method, 'capturePhoto');
      return handler(call);
    });
    addTearDown(() => TestDefaultBinaryMessengerBinding.instance
        .defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null));
  }

  test('deterministic test photo picker returns non-empty JPEG header bytes',
      () async {
    final picker = DeterministicTestPhotoPicker();
    final bytes = await picker.pickImage();

    expect(bytes, isNotNull);
    expect(bytes!.isNotEmpty, isTrue);
    expect(bytes, orderedEquals(mockPhoto()));
    expect(bytes.sublist(0, 2), orderedEquals([0xFF, 0xD8]));
    expect(picker.result.contentType, 'image/jpeg');
    expect(picker.result.metadata, containsPair('source', 'deterministic_test'));
    expect(picker.result.capturedAt,
        DateTime.fromMillisecondsSinceEpoch(kDeterministicCaptureEpochMs));
  });

  test('mock photo picker honors injected bytes', () async {
    const injected = [0xFF, 0xD8, 0xFF, 0xC0, 0x00, 0x11];
    final picker = DeterministicTestPhotoPicker(
        bytes: Uint8List.fromList(injected));
    final bytes = await picker.pickImage();
    expect(bytes, orderedEquals(injected));
  });

  test('platform channel picker returns native capture bytes', () async {
    mockChannel((call) => mockPhoto());

    final bytes = await PlatformChannelPhotoPicker().pickImage();
    expect(bytes, orderedEquals(mockPhoto()));
    expect(bytes!.isNotEmpty, isTrue);
  });

  test('platform channel picker surfaces native capture errors', () async {
    mockChannel((call) =>
        throw PlatformException(code: 'camera_unavailable', message: 'denied'));

    await expectLater(
      PlatformChannelPhotoPicker().pickImage(),
      throwsA(isA<PlatformException>()
          .having((e) => e.code, 'code', 'camera_unavailable')),
    );
  });

  test('platform channel picker returns null when capture is cancelled',
      () async {
    mockChannel((call) => null);

    final bytes = await PlatformChannelPhotoPicker().pickImage();
    expect(bytes, isNull);
  });
}