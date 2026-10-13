import Flutter
import Foundation
import Security
import Darwin

public class SecurityPlugin: NSObject, FlutterPlugin {
    public static func register(with registrar: FlutterPluginRegistrar) {
        let channel = FlutterMethodChannel(name: "com.integin.field/security", binaryMessenger: registrar.messenger())
        let instance = SecurityPlugin()
        registrar.addMethodCallDelegate(instance, channel: channel)
    }

    public func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        switch call.method {
        case "generateAttestedKey":
            guard let args = call.arguments as? [String: Any],
                  let alias = args["alias"] as? String else {
                result(FlutterError(code: "BAD_ARGS", message: "Missing alias", details: nil))
                return
            }

            do {
                let keyResult = try generateSecureEnclaveKey(alias: alias)
                result([
                    "alias": keyResult.alias,
                    "publicKeyDer": FlutterStandardTypedData(bytes: keyResult.publicKeyDer),
                    "keyOrigin": "SECURE_ENCLAVE",
                    "certificateChainPem": [String]()
                ])
            } catch {
                result(FlutterError(code: "KEYGEN_FAILED", message: error.localizedDescription, details: nil))
            }

        case "signDigest":
            guard let args = call.arguments as? [String: Any],
                  let alias = args["alias"] as? String,
                  let digestTyped = args["digest"] as? FlutterStandardTypedData else {
                result(FlutterError(code: "BAD_ARGS", message: "Missing alias or digest", details: nil))
                return
            }

            do {
                let signature = try signPrecomputedDigest(alias: alias, digest: digestTyped.data)
                result(FlutterStandardTypedData(bytes: signature))
            } catch let error as NSError where error.domain == NSOSStatusErrorDomain {
                if error.code == errSecInteractionNotAllowed {
                    result(FlutterError(code: "AUTH_REQUIRED", message: "User authentication required or expired", details: nil))
                } else {
                    result(FlutterError(code: "SIGN_FAILED", message: error.localizedDescription, details: nil))
                }
            } catch {
                result(FlutterError(code: "SIGN_FAILED", message: error.localizedDescription, details: nil))
            }

        case "getBootSession":
            do {
                let session = try readAppleBootSession()
                result([
                    "sessionID": session.sessionID,
                    "monoNanos": session.monoNanos
                ])
            } catch {
                result(FlutterError(code: "BOOT_SESSION_FAILED", message: error.localizedDescription, details: nil))
            }

        default:
            result(FlutterMethodNotImplemented)
        }
    }

    private func generateSecureEnclaveKey(alias: String) throws -> (alias: String, publicKeyDer: Data) {
        let tag = alias.data(using: .utf8)!

        let deleteQuery: [String: Any] = [
            kSecClass as String: kSecClassKey,
            kSecAttrApplicationTag as String: tag,
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom
        ]
        SecItemDelete(deleteQuery as CFDictionary)

        var error: Unmanaged<CFError>?
        guard let access = SecAccessControlCreateWithFlags(
            kCFAllocatorDefault,
            kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
            [.privateKeyUsage, .userPresence],
            &error
        ) else {
            throw error!.takeRetainedValue() as Error
        }

        let attributes: [String: Any] = [
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrKeySizeInBits as String: 256,
            kSecAttrTokenID as String: kSecAttrTokenIDSecureEnclave,
            kSecPrivateKeyAttrs as String: [
                kSecAttrIsPermanent as String: true,
                kSecAttrApplicationTag as String: tag,
                kSecAttrAccessControl as String: access
            ]
        ]

        guard let privateKey = SecKeyCreateRandomKey(attributes as CFDictionary, &error) else {
            throw error!.takeRetainedValue() as Error
        }

        guard let publicKey = SecKeyCopyPublicKey(privateKey),
              let rawPoint = SecKeyCopyExternalRepresentation(publicKey, &error) as Data? else {
            throw error!.takeRetainedValue() as Error
        }

        // Prepend 26-byte ASN.1 SPKI header for P-256 so Go's
        // x509.ParsePKIXPublicKey can parse the key correctly.
        // SPKI: SEQUENCE { SEQUENCE { OID id-ecPublicKey, OID prime256v1 },
        //         BIT STRING { 0x00 || 0x04 || X || Y } }
        // Length: 26 bytes header + 65 bytes raw point = 91 bytes total
        let spkiHeader: [UInt8] = [
            0x30, 0x59,                       // SEQUENCE (91 bytes)
            0x30, 0x13,                       // SEQUENCE (19 bytes)
            0x06, 0x07, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x02, 0x01, // OID 1.2.840.10045.2.1 (id-ecPublicKey)
            0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07, // OID 1.2.840.10045.3.1.7 (prime256v1)
            0x03, 0x42, 0x00                  // BIT STRING (66 bytes, 0 padding bits)
        ]
        let fullSPKI = Data(spkiHeader) + rawPoint

        return (alias, fullSPKI)
    }

    private func signPrecomputedDigest(alias: String, digest: Data) throws -> Data {
        let tag = alias.data(using: .utf8)!
        let query: [String: Any] = [
            kSecClass as String: kSecClassKey,
            kSecAttrApplicationTag as String: tag,
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            kSecReturnRef as String: true
        ]

        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        guard status == errSecSuccess, let privateKey = item else {
            throw NSError(domain: NSOSStatusErrorDomain, code: Int(status), userInfo: [NSLocalizedDescriptionKey: "Key not found"])
        }

        var error: Unmanaged<CFError>?
        guard let signature = SecKeyCreateSignature(
            (privateKey as! SecKey),
            .ecdsaSignatureDigestX962,
            digest as CFData,
            &error
        ) as Data? else {
            throw error!.takeRetainedValue() as Error
        }

        return signature
    }

    private func readAppleBootSession() throws -> (sessionID: String, monoNanos: Int64) {
        var mib: [Int32] = [CTL_KERN, KERN_BOOTTIME]
        var bootTime = timeval()
        var size = MemoryLayout<timeval>.stride

        let status = mib.withUnsafeMutableBufferPointer { mibPtr -> Int32 in
            guard let base = mibPtr.baseAddress else { return -1 }
            return sysctl(base, 2, &bootTime, &size, nil, 0)
        }

        guard status == 0 else {
            throw NSError(domain: NSPOSIXErrorDomain, code: Int(errno), userInfo: nil)
        }

        let monoNanos = Int64(clock_gettime_nsec_np(CLOCK_UPTIME_RAW))
        let sessionID = "ios:boottime:\(bootTime.tv_sec).\(bootTime.tv_usec)"

        return (sessionID, monoNanos)
    }
}
