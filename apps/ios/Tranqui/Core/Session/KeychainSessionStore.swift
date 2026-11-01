import Foundation
import Security

/// One session record in the Keychain: the JWT pair plus the absolute expiry
/// of the access token (contract v0.5.0). Tokens live only here — never in
/// UserDefaults, never in logs (docs/stack/ios.md, privacy: minimum data).
struct StoredSession: Codable, Equatable, Sendable {
    let accessToken: String
    let refreshToken: String
    /// Absolute expiry of the access token (issue instant + expiresInSeconds).
    let expiresAt: Date
}

/// The persistence seam behind `SessionStore` (Keychain in the app, memory
/// in tests). Synchronous on purpose: SecItem calls are atomic and
/// thread-safe, and the reactive store needs a warm cache at init.
protocol SessionPersisting: Sendable {
    func load() -> StoredSession?
    func save(_ session: StoredSession)
    func clear()
}

/// Keychain persistence for `StoredSession`: one `kSecClassGenericPassword`
/// item under a fixed service + account, with
/// `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly` so the pair never
/// leaves this device (excluded from iCloud backup and same-Apple-ID
/// syncing — plan issue #76).
///
/// The SecItem dictionaries are built by subscript (not multi-line
/// literals): swiftlint's `trailing_comma` and swiftformat's
/// `trailingCommas` default rules disagree on that shape, so the factory
/// style avoids it.
final class KeychainSessionStore: SessionPersisting, @unchecked Sendable {
    private let service: String
    private let account = "session"

    /// In-memory mirror: the hot path (bearer attach on every request)
    /// reads this, never the Keychain.
    private let lock = NSLock()
    private var cache: StoredSession?

    /// Loads the persisted session once; a corrupt record reads as no
    /// session (the user just signs in again — never a crash on launch).
    init(service: String = "com.imalive.tranqui.session") {
        self.service = service
        cache = Self.read(service: service, account: account)
    }

    func load() -> StoredSession? {
        lock.withLock { cache }
    }

    func save(_ session: StoredSession) {
        guard let data = try? JSONEncoder().encode(session) else { return }
        var attributesToUpdate: [String: Any] = [:]
        attributesToUpdate[kSecValueData as String] = data
        attributesToUpdate[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        if SecItemUpdate(baseQuery as CFDictionary, attributesToUpdate as CFDictionary) == errSecItemNotFound {
            var add = baseQuery
            add[kSecValueData as String] = data
            add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
            SecItemAdd(add as CFDictionary, nil)
        }
        lock.withLock { cache = session }
    }

    func clear() {
        SecItemDelete(baseQuery as CFDictionary)
        lock.withLock { cache = nil }
    }

    private var baseQuery: [String: Any] {
        var query: [String: Any] = [:]
        query[kSecClass as String] = kSecClassGenericPassword
        query[kSecAttrService as String] = service
        query[kSecAttrAccount as String] = account
        return query
    }

    private static func read(service: String, account: String) -> StoredSession? {
        var query: [String: Any] = [:]
        query[kSecClass as String] = kSecClassGenericPassword
        query[kSecAttrService as String] = service
        query[kSecAttrAccount as String] = account
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var result: AnyObject?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
              let data = result as? Data
        else {
            return nil
        }
        return try? JSONDecoder().decode(StoredSession.self, from: data)
    }
}
