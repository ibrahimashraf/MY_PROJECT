package com.integin.integin_field_app

import android.content.Intent
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyInfo
import android.security.keystore.KeyProperties
import android.security.keystore.StrongBoxUnavailableException
import android.util.Log
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.security.KeyFactory
import java.security.KeyStore
import java.security.MessageDigest
import java.security.PrivateKey
import java.security.Signature
import java.security.cert.Certificate
import java.security.spec.ECGenParameterSpec
import java.util.Base64

class MainActivity : FlutterActivity() {
    private val mainHandler = Handler(Looper.getMainLooper())
    private val channelName = "integin.attestation/v1"

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName)
            .setMethodCallHandler { call, result ->
                if (call.method != "mintChain") {
                    result.error("unknown_method", "unknown method ${call.method}", null)
                    return@setMethodCallHandler
                }
                val nonceHex = call.arguments as? String
                if (nonceHex.isNullOrEmpty()) {
                    result.success(mintResult("mintChain requires a non-empty nonceHex string"))
                    return@setMethodCallHandler
                }
                Thread {
                    val res = mintChain(nonceHex)
                    mainHandler.post { result.success(res) }
                }.start()
            }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        maybeRunIntentAttestation(intent)
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        maybeRunIntentAttestation(intent)
    }

    private fun maybeRunIntentAttestation(intent: Intent?) {
        if (intent == null) return
        val nonceHex = intent.getStringExtra("attest_nonce") ?: return
        Thread {
            val res = mintChain(nonceHex)
            // small fields first so a truncated logcat line still carries them
            val probe = JSONObject()
                .put("nonce_hex", nonceHex)
                .put("device_model", Build.MODEL)
                .put("android_sdk", Build.VERSION.SDK_INT)
            putChain(probe, res)
            val json = probe.toString()
            logChunked("META", json)
            try {
                val dir = getExternalFilesDir(null) ?: File(filesDir, "attest-probe-dir")
                dir.mkdirs()
                val out = File(dir, "attest_probe.json")
                out.writeText(json)
                Log.i(TAG, "probe written: ${out.absolutePath}")
            } catch (e: Exception) {
                Log.e(TAG, "probe write failed (external): $e")
            }
            // Internal mirror: MIUI refuses to create the external Android/data
            // dir; pullable via `adb exec-out run-as` where SELinux allows.
            try {
                val internalOut = File(filesDir, "attest_probe.json")
                internalOut.writeText(json)
                Log.i(TAG, "probe written (internal): ${internalOut.absolutePath}")
            } catch (e: Exception) {
                Log.e(TAG, "internal probe write failed: $e")
            }
        }.start()
    }

    // logcat truncates long lines (~4k), so emit the probe in <=3600-char
    // chunks; the receiver concatenates ATTEST_PROBE_<SEQ> lines in order.
    private fun logChunked(tagSuffix: String, payload: String) {
        var off = 0
        var seq = 0
        while (off < payload.length) {
            val end = (off + 3600).coerceAtMost(payload.length)
            Log.i(TAG, "ATTEST_PROBE_$tagSuffix#$seq ${payload.substring(off, end)}")
            off = end
            seq++
        }
    }

    private fun putChain(json: JSONObject, res: Map<String, Any?>): JSONObject {
        @Suppress("UNCHECKED_CAST")
        val chain = res["chain_pem"] as? List<String> ?: emptyList()
        val arr = JSONArray()
        chain.forEach { arr.put(it) }
        if (arr.length() > 0) json.put("chain_pem", arr)
        (res["security_level"] as? String)?.let { json.put("security_level", it) }
        (res["key_alias"] as? String)?.let { json.put("key_alias", it) }
        (res["error"] as? String)?.let { json.put("error", it) }
        (res["public_key_spki_hex"] as? String)?.let { json.put("public_key_spki_hex", it) }
        (res["nonce_signature_hex"] as? String)?.let { json.put("nonce_signature_hex", it) }
        return json
    }

    private fun mintResult(error: String): Map<String, Any?> = mapOf("error" to error)

    /** Mints a P-256 key-attestation chain for [nonceHex] and returns a
     *  serializable summary. StrongBox-first with TEE fallback. Never throws;
     *  failures are reported in the returned map under "error". */
    private fun mintChain(nonceHex: String): Map<String, Any?> {
        val alias = "integin_probe_" + sha256(nonceHex).substring(0, 8)
        // load-bearing: the Keymaster attestationChallenge must be the UTF-8
        // bytes of the nonce hex string — the Go verifier compares them
        // byte-equal against expectedChallenge = []byte(cha.Nonce).
        val challenge = nonceHex.toByteArray(Charsets.UTF_8)
        try {
            val keyStore = KeyStore.getInstance("AndroidKeyStore")
            keyStore.load(null)
            val chain = generateAttestedChain(alias, challenge, keyStore)
            val securityLevel = securityLevelOf(alias, keyStore, chain.firstOrNull())
            return mapOf(
                "chain_pem" to chain.map { pemEncode(it.encoded) },
                "security_level" to securityLevel,
                "key_alias" to alias,
                "os_version" to (Build.VERSION.RELEASE ?: ""),
                "public_key_spki_hex" to (chain.firstOrNull()?.publicKey?.encoded?.let { bytesToHex(it) } ?: ""),
                "nonce_signature_hex" to signNonce(alias, nonceHex),
            )
        } catch (e: Exception) {
            Log.e(TAG, "mint failed", e)
            return mintResult(e.message ?: e.toString())
        }
    }

    private fun generateAttestedChain(
        alias: String,
        challenge: ByteArray,
        keyStore: KeyStore,
    ): Array<out Certificate> {
        try {
            keyStore.deleteEntry(alias) // same nonce re-probe: re-mint cleanly
        } catch (ignored: Exception) {
        }
        return try {
            generateKeyWithSetting(alias, challenge, keyStore, strongBox = true)
        } catch (e: Exception) {
            if (!causedByStrongBoxUnavailable(e)) throw e
            Log.i(TAG, "StrongBox unavailable, retrying with TEE: $e")
            generateKeyWithSetting(alias, challenge, keyStore, strongBox = false)
        }
    }

    private fun generateKeyWithSetting(
        alias: String,
        challenge: ByteArray,
        keyStore: KeyStore,
        strongBox: Boolean,
    ): Array<out Certificate> {
        val spec = KeyGenParameterSpec.Builder(
            alias,
            KeyProperties.PURPOSE_SIGN,
        )
            .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
            .setDigests(KeyProperties.DIGEST_SHA256)
            .setAttestationChallenge(challenge)
            .setIsStrongBoxBacked(strongBox)
            .build()
        val kpg = java.security.KeyPairGenerator.getInstance(
            KeyProperties.KEY_ALGORITHM_EC, "AndroidKeyStore"
        )
        kpg.initialize(spec)
        kpg.generateKeyPair()
        return keyStore.getCertificateChain(alias)
    }

    private fun causedByStrongBoxUnavailable(e: Throwable): Boolean {
        var cur: Throwable? = e
        while (cur != null) {
            if (cur is StrongBoxUnavailableException) return true
            if ((cur.message ?: "").contains("StrongBox")) return true
            cur = cur.cause
        }
        return false
    }

    private fun securityLevelOf(
        alias: String,
        keyStore: KeyStore,
        leaf: Certificate?,
    ): String {
        val alg = leaf?.publicKey?.algorithm ?: "EC"
        val key = try {
            keyStore.getKey(alias, null)
        } catch (e: Exception) {
            Log.w(TAG, "getKey failed: $e")
            return "UNKNOWN"
        }
        if (key !is PrivateKey) {
            Log.w(TAG, "key is not a PrivateKey: ${key?.javaClass?.name}")
            return "UNKNOWN"
        }
        return try {
            if (Build.VERSION.SDK_INT >= 31) {
                val info = KeyFactory.getInstance(alg, "AndroidKeyStore")
                    .getKeySpec(key, KeyInfo::class.java)
                when (info.securityLevel) {
                    KeyProperties.SECURITY_LEVEL_STRONGBOX -> "STRONGBOX"
                    KeyProperties.SECURITY_LEVEL_TRUSTED_ENVIRONMENT -> "TEE"
                    KeyProperties.SECURITY_LEVEL_SOFTWARE -> "SOFTWARE"
                    else -> "UNKNOWN"
                }
            } else {
                // below API 31 only the coarse hardware flag exists
                val info = KeyFactory.getInstance(alg, "AndroidKeyStore")
                    .getKeySpec(key, KeyInfo::class.java)
                if (info.isInsideSecureHardware) "TEE?" else "SOFTWARE?"
            }
        } catch (e: Exception) {
            Log.w(TAG, "KeyInfo unavailable: $e")
            "UNKNOWN"
        }
    }

    private fun signNonce(alias: String, nonceHex: String): String {
        val keyStore = KeyStore.getInstance("AndroidKeyStore")
        keyStore.load(null)
        val key = keyStore.getKey(alias, null) as? PrivateKey ?: return ""
        val sig = Signature.getInstance("SHA256withECDSA")
        sig.initSign(key)
        sig.update(nonceHex.toByteArray(Charsets.UTF_8))
        return bytesToHex(sig.sign())
    }

    private fun pemEncode(der: ByteArray): String {
        val b64 = Base64.getMimeEncoder(64, "\n".toByteArray()).encodeToString(der)
        return "-----BEGIN CERTIFICATE-----\n$b64\n-----END CERTIFICATE-----"
    }

    private fun sha256(input: String): String {
        return bytesToHex(MessageDigest.getInstance("SHA-256").digest(input.toByteArray(Charsets.UTF_8)))
    }

    private fun bytesToHex(bytes: ByteArray): String {
        val sb = StringBuilder(bytes.size * 2)
        for (b in bytes) sb.append("%02x".format(b.toInt() and 0xff))
        return sb.toString()
    }

    companion object {
        private const val TAG = "InteginAttest"
    }
}