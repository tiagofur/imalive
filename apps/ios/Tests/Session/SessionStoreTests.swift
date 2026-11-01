import Foundation
import Testing
@testable import Tranqui

/// The reactive gate of plan issue #76: the store's state survives a
/// relaunch through the persisting seam, and a clear() — the path a rejected
/// refresh takes — walks the user back to onboarding.
@MainActor
@Suite("SessionStore")
struct SessionStoreTests {
    private static func storedSession(accessToken: String = "access-token-1") -> StoredSession {
        StoredSession(
            accessToken: accessToken,
            refreshToken: "refresh-token-1",
            expiresAt: Date(timeIntervalSince1970: 1_799_900_000)
        )
    }

    @Test func startsWithoutASessionOnTheOnboardingGate() {
        let store = SessionStore(persisting: InMemoryPersisting())

        #expect(store.gate == .needsOnboarding)
        #expect(store.session == nil)
    }

    @Test func restoresThePersistedSessionAndLandsReady() {
        let persisted = InMemoryPersisting(initial: Self.storedSession())
        persisted.setOnboardingComplete(true) // a user on home finished the flow

        let store = SessionStore(persisting: persisted)

        #expect(store.gate == .ready)
        #expect(store.session == Self.storedSession())
        #expect(store.cachedSession() == Self.storedSession())
    }

    @Test func savePersistsAndOpensTheGate() {
        let persisting = InMemoryPersisting()
        let store = SessionStore(persisting: persisting)

        store.save(Self.storedSession(accessToken: "verified"))
        store.completeOnboarding()

        #expect(store.gate == .ready)
        #expect(store.session?.accessToken == "verified")
        #expect(persisting.load()?.accessToken == "verified")
    }

    @Test func clearWipesPersistenceAndRevertsTheGateToOnboarding() {
        let persisting = InMemoryPersisting()
        let store = SessionStore(persisting: persisting)
        store.save(Self.storedSession())

        store.clear()

        #expect(store.gate == .needsOnboarding)
        #expect(store.session == nil)
        #expect(store.cachedSession() == nil)
        #expect(persisting.load() == nil)
    }

    // MARK: the onboarding-finished flag (issue #76 slice 4)

    @Test func aSessionAloneKeepsTheOnboardingGateUntilTheFlowFinishes() {
        let store = SessionStore(persisting: InMemoryPersisting())

        store.save(Self.storedSession()) // what the OTP verify persists

        #expect(store.gate == .needsOnboarding)
        #expect(store.onboardingComplete == false)
    }

    @Test func completingTheOnboardingOpensTheReadyGateAndSurvivesARestart() {
        let persisting = InMemoryPersisting()
        let store = SessionStore(persisting: persisting)
        store.save(Self.storedSession())

        store.completeOnboarding()

        #expect(store.gate == .ready)
        #expect(store.onboardingComplete == true)
        #expect(persisting.isOnboardingComplete() == true)

        // The flag is persisted: a relaunch lands home directly.
        let relaunched = SessionStore(persisting: persisting)
        #expect(relaunched.gate == .ready)
    }

    @Test func completingTheOnboardingWithoutASessionNeverOpensHome() {
        let store = SessionStore(persisting: InMemoryPersisting())

        store.completeOnboarding()

        #expect(store.gate == .needsOnboarding)
    }

    @Test func clearResetsTheOnboardingFlagSoARejectingRefreshWalksBack() {
        let store = SessionStore(persisting: InMemoryPersisting())
        store.save(Self.storedSession())
        store.completeOnboarding()

        store.clear() // the rejected-refresh path

        #expect(store.gate == .needsOnboarding)
        #expect(store.onboardingComplete == false)
    }
}
