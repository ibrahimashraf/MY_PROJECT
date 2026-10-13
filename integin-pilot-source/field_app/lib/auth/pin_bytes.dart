import 'dart:typed_data';

/// Overwrites [pin] with zeros so the plaintext does not linger in memory.
///
/// Caller owns the buffer lifetime; this only scrubs the bytes.
void zeroPinBytes(Uint8List pin) {
  for (var i = 0; i < pin.length; i++) {
    pin[i] = 0;
  }
}