import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';

import 'camera_picker.dart';

class InAppCameraPhotoService implements PhotoPickerService {
  final BuildContext context;
  InAppCameraPhotoService(this.context);

  @override
  Future<Uint8List?> pickImage() async {
    final picker = ImagePicker();
    final XFile? file = await picker.pickImage(
      source: ImageSource.camera,
      imageQuality: 50,
      maxWidth: 1280,
      maxHeight: 720,
    );
    if (file != null) {
      return await file.readAsBytes();
    }
    return null;
  }
}
