import Foundation
import Testing
@testable import Tranqui

/// The Keychain round-trip of plan issue #76 (real Keychain: the test target
/// is hosted in the app, so SecItem works on the simulator). A unique service
/// per test keeps the runs hermetic.
@Suite("KeychainSessionStore")
struct KeychainSessionStoreTests {
    private func freshStore() -> KeychainSessionStore {
        KeychainSessionStore(service: "com.imalive.tranqui.tests.\(UUID().uuidString)")
    }

    private func session(
        accessToken: String = "access-token-1",
        refreshToken: String = "refresh-token-1"
    ) -> StoredSession {
        StoredSession(
            accessToken: accessToken,
            refreshToken: refreshToken,
            expiresAt: Date(timeIntervalSince1970: 1_799_900_000)
        )
    }

    @Test func roundTripsTheSessionThroughTheKeychain() {
        let store = freshStore()
        let stored = session()

        store.save(stored)

        #expect(store.load() == stored)
    }

    @Test func aNewInstanceReadsWhatThePreviousOnePersisted() {
        let service = "com.imalive.tranqui.tests.\(UUID().uuidString)"
        let stored = session(accessToken: "rotated-access")

        KeychainSessionStore(service: service).save(stored)
        let relaunched = KeychainSessionStore(service: service)

        #expect(relaunched.load() == stored)
    }

    @Test func clearWipesTheItemFromTheKeychain() {
        let service = "com.imalive.tranqui.tests.\(UUID().uuidString)"
        let store = KeychainSessionStore(service: service)
        store.save(session())

        store.clear()

        #expect(store.load() == nil)
        #expect(KeychainSessionStore(service: service).load() == nil)
    }

    @Test func anEmptyStoreLoadsAsNoSession() {
        #expect(freshStore().load() == nil)
    }

    @Test func savingTwiceReplacesInsteadOfDuplicating() {
        let store = freshStore()

        store.save(session(accessToken: "first"))
        store.save(session(accessToken: "second"))

        #expect(store.load()?.accessToken == "second")
    }
}
