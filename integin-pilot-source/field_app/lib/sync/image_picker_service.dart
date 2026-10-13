import 'dart:typed_data';

import 'package:image_picker/image_picker.dart';

import 'camera_picker.dart';

/// Captures a photo using the cross-platform image_picker package.
class ImagePickerPhotoService implements PhotoPickerService {
  final ImagePicker _picker = ImagePicker();

  @override
  Future<Uint8List?> pickImage() async {
    final XFile? photo = await _picker.pickImage(
      source: ImageSource.camera,
      imageQuality: 80,
    );
    if (photo == null) {
      return null;
    }
    return await photo.readAsBytes();
  }
}
