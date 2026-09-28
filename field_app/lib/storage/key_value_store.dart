// Simple key‑value store wrapper using SharedPreferences
// Provides the minimal API used by the app (getInstance, getString, setString).
import 'package:shared_preferences/shared_preferences.dart';

class SharedPreferencesKeyValueStore {
  SharedPreferencesKeyValueStore._();

  static Future<SharedPreferencesKeyValueStore> getInstance() async {
    // Ensure SharedPreferences is initialized.
    await SharedPreferences.getInstance();
    return SharedPreferencesKeyValueStore._();
  }

  Future<void> setString(String key, String value) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(key, value);
  }

  Future<String?> getString(String key) async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(key);
  }
}
