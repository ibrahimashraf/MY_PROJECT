package com.integin.field.security

import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.SystemClock
import android.provider.Settings
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.security.keystore.StrongBoxUnavailableException
import android.security.keystore.UserNotAuthenticatedException
import android.security.keystore.KeyPermanentlyInvalidatedException
import android.util.Base64
import io.flutter.embedding.engine.plugins.FlutterPlugin
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import io.flutter.plugin.common.MethodChannel.MethodCallHandler
import io.flutter.plugin.common.MethodChannel.Result
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.PrivateKey
import java.security.ProviderException
import java.security.Signature
import java.security.spec.ECGenParameterSpec

class SecurityPlugin : FlutterPlugin, MethodCallHandler {
    private lateinit var channel: MethodChannel
    private lateinit var context: Context

    override fun onAttachedToEngine(binding: FlutterPlugin.FlutterPluginBinding) {
        context = binding.applicationContext
        channel = MethodChannel(binding.binaryMessenger, "com.integin.field/security")
        channel.setMethodCallHandler(this)
    }

    override fun onDetachedFromEngine(binding: FlutterPlugin.FlutterPluginBinding) {
        channel.setMethodCallHandler(null)
    }

    override fun onMethodCall(call: MethodCall, result: Result) {
        when (call.method) {
            "generateAttestedKey" -> {
                val alias = call.argument<String>("alias") ?: return result.error("BAD_ARGS", "Missing alias", null)
                val challenge = call.argument<ByteArray>("challenge") ?: return result.error("BAD_ARGS", "Missing challenge", null)
                val requireUserAuth = call.argument<Boolean>("requireUserAuth") ?: false
                val authValidity = call.argument<Int>("authValidityDurationSeconds") ?: 0

                try {
                    val attestation = generateKey(alias, challenge, requireUserAuth, authValidity)
                    result.success(
                        mapOf(
                            "alias" to attestation.alias,
                            "publicKeyDer" to attestation.publicKeyDer,
                            "keyOrigin" to attestation.keyOrigin,
                            "certificateChainPem" to attestation.certificateChainPem
                        )
                    )
                } catch (e: Exception) {
                    result.error("KEYGEN_FAILED", e.message, e.stackTraceToString())
                }
            }
            "signDigest" -> {
                val alias = call.argument<String>("alias") ?: return result.error("BAD_ARGS", "Missing alias", null)
                val digest = call.argument<ByteArray>("digest") ?: return result.error("BAD_ARGS", "Missing digest", null)

                // Offload to background thread: StrongBox secure elements
                // have 150-400ms latency over low-bandwidth serial buses.
                // Blocking the Flutter UI thread causes jank during rapid
                // inspection bursts.
                Thread {
                    val mainHandler = android.os.Handler(android.os.Looper.getMainLooper())
                    try {
                        val signature = signPrecomputedDigest(alias, digest)
                        // result() must be called on the main thread
                        mainHandler.post { result.success(signature) }
                    } catch (e: Exception) {
                        val isAuthError = isAuthError(e)
                        val code = if (isAuthError) "AUTH_REQUIRED" else "SIGN_FAILED"
                        val message = if (isAuthError) "User authentication required or expired" else e.message
                        mainHandler.post { result.error(code, message, e.stackTraceToString()) }
                    }
                }.start()
            }
            "getBootSession" -> {
                try {
                    val session = readBootSession()
                    result.success(
                        mapOf(
                            "sessionID" to session.first,
                            "monoNanos" to session.second
                        )
                    )
                } catch (e: Exception) {
                    result.error("BOOT_SESSION_FAILED", e.message, e.stackTraceToString())
                }
            }
            "com.integin.field/battery" -> {
                when (call.method) {
                    "requestExemption" -> {
                        try {
                            val granted = requestBatteryOptimizationExemption()
                            result.success(granted)
                        } catch (e: Exception) {
                            result.error("BATTERY_EXEMPTION_FAILED", e.message, e.stackTraceToString())
                        }
                    }
                    else -> result.notImplemented()
                }
            }
            else -> result.notImplemented()
        }
    }

    private fun isAuthError(e: Exception): Boolean {
        var cur: Throwable? = e
        while (cur != null) {
            if (cur is UserNotAuthenticatedException ||
                cur is KeyPermanentlyInvalidatedException ||
                cur.message?.contains("UserNotAuthenticated", ignoreCase = true) == true ||
                cur.message?.contains("BiometricPrompt", ignoreCase = true) == true ||
                cur.message?.contains("AUTHENTICATION", ignoreCase = true) == true ||
                cur.message?.contains("not authenticated", ignoreCase = true) == true
            ) {
                return true
            }
            cur = cur.cause
        }
        return false
    }

    private fun generateKey(
        alias: String,
        challenge: ByteArray,
        requireUserAuth: Boolean,
        authValidity: Int
    ): AttestationPayload {
        val keyStore = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
        if (keyStore.containsAlias(alias)) {
            keyStore.deleteEntry(alias)
        }

        return try {
            buildKeyPair(alias, challenge, requireUserAuth, authValidity, isStrongBox = true)
        } catch (e: Exception) {
            when (e) {
                is StrongBoxUnavailableException,
                is ProviderException -> {
                    buildKeyPair(alias, challenge, requireUserAuth, authValidity, isStrongBox = false)
                }
                else -> throw e
            }
        }
    }

    private fun buildKeyPair(
        alias: String,
        challenge: ByteArray,
        requireUserAuth: Boolean,
        authValidity: Int,
        isStrongBox: Boolean
    ): AttestationPayload {
        val kpg = KeyPairGenerator.getInstance(KeyProperties.KEY_ALGORITHM_EC, "AndroidKeyStore")
        val builder = KeyGenParameterSpec.Builder(alias, KeyProperties.PURPOSE_SIGN)
            .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
            .setDigests(KeyProperties.DIGEST_NONE)
            .setAttestationChallenge(challenge)

        if (requireUserAuth) {
            builder.setUserAuthenticationRequired(true)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
                builder.setUserAuthenticationParameters(
                    authValidity,
                    KeyProperties.AUTH_BIOMETRIC_STRONG or KeyProperties.AUTH_DEVICE_CREDENTIAL
                )
            } else {
                @Suppress("DEPRECATION")
                builder.setUserAuthenticationValidityDurationSeconds(authValidity)
            }
        } else {
            builder.setUserAuthenticationRequired(false)
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P && isStrongBox) {
            builder.setIsStrongBoxBacked(true)
        }

        kpg.initialize(builder.build())
        val keyPair = kpg.generateKeyPair()

        val keyStore = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
        val certChain = keyStore.getCertificateChain(alias)
            ?: throw IllegalStateException("KeyStore emitted null certificate chain for $alias")

        val pemChain = certChain.map { cert ->
            val b64 = Base64.encodeToString(cert.encoded, Base64.NO_WRAP)
            "-----BEGIN CERTIFICATE-----\n$b64\n-----END CERTIFICATE-----"
        }

        val origin = if (isStrongBox) "STRONGBOX" else "TEE"
        return AttestationPayload(alias, keyPair.public.encoded, origin, pemChain)
    }

    private fun signPrecomputedDigest(alias: String, digest: ByteArray): ByteArray {
        val keyStore = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
        val privateKey = keyStore.getKey(alias, null) as? PrivateKey
            ?: throw IllegalStateException("Private key not found for alias: $alias")

        val signer = Signature.getInstance("NONEwithECDSA").apply {
            initSign(privateKey)
            update(digest)
        }
        return signer.sign()
    }

    private fun readBootSession(): Pair<String, Long> {
        val bootCount = Settings.Global.getInt(
            context.contentResolver,
            Settings.Global.BOOT_COUNT,
            -1
        )
        val monoNanos = SystemClock.elapsedRealtimeNanos()

        val sessionID = if (bootCount != -1) {
            "android:boot_count:$bootCount"
        } else {
            "android:mono_only:unavailable"
        }

        return Pair(sessionID, monoNanos)
    }

    private fun requestBatteryOptimizationExemption(): Boolean {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            val powerManager = context.getSystemService(Context.POWER_SERVICE) as android.os.PowerManager
            if (!powerManager.isIgnoringBatteryOptimizations(context.packageName)) {
                val intent = Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS).apply {
                    data = android.net.Uri.parse("package:${context.packageName}")
                }
                // Cannot start activity from this context without FLAG_ACTIVITY_NEW_TASK
                intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                context.startActivity(intent)
                return true
            }
        }
        return true // Already exempted or pre-M SDK
    }

    private data class AttestationPayload(
        val alias: String,
        val publicKeyDer: ByteArray,
        val keyOrigin: String,
        val certificateChainPem: List<String>
    )
}
