import Foundation
import Testing
@testable import Tranqui

@MainActor
struct HomeModelTests {
    /// Repos the tests control: fixed values, an explosion or a counter.
    private struct FixedRepo: ProfileRepository {
        let profile: Profile

        func profile() async throws -> Profile {
            profile
        }

        func update(_: ProfilePatch) async throws {}
    }

    private struct ExplodingRepo: ProfileRepository {
        func profile() async throws -> Profile {
            throw URLError(.notConnectedToInternet)
        }

        func update(_: ProfilePatch) async throws {}
    }

    /// Counts the profile reads: proves what the gate does and does not read.
    /// @unchecked: the lock guards the only mutable state.
    private final class CountingRepo: ProfileRepository, @unchecked Sendable {
        let profile: Profile
        private let lock = NSLock()
        private var count = 0

        init(profile: Profile) {
            self.profile = profile
        }

        var calls: Int {
            lock.withLock { count }
        }

        func profile() async throws -> Profile {
            lock.withLock { count += 1 }
            return profile
        }

        func update(_: ProfilePatch) async throws {}
    }

    private func profile(
        streakDays: Int,
        checkinTime: String = "21:00",
        pauseUntil: Date? = nil
    ) -> Profile {
        Profile(streakDays: streakDays, checkinTime: checkinTime, pauseUntil: pauseUntil)
    }

    private func session() -> StoredSession {
        StoredSession(
            accessToken: "access-token-1",
            refreshToken: "refresh-token-1",
            expiresAt: Date(timeIntervalSince1970: 0)
        )
    }

    private func signedInStore() -> SessionStore {
        // A user on home has a session AND a finished onboarding (slice 4:
        // the gate requires both).
        let store = SessionStore(persisting: InMemoryPersisting(initial: session()))
        store.completeOnboarding()
        return store
    }

    @Test func stateStartsPendingAtNineToday() {
        let model = HomeModel(profiles: FixedRepo(profile: profile(streakDays: 0)), sessions: signedInStore())

        #expect(model.state == .pending(nextCheckin: "hoy 21:00"))
        #expect(model.nextCheckinLabel == "hoy 21:00")
    }

    // MARK: - session gate (issue #76 slice 2)

    @Test func withoutASessionTheGateAsksForOnboardingAndNeverReadsTheProfile() {
        let store = SessionStore(persisting: InMemoryPersisting())
        let repo = CountingRepo(profile: profile(streakDays: 12))
        let model = HomeModel(profiles: repo, sessions: store)

        #expect(model.gate == .needsOnboarding)
        #expect(model.streakDays == 0)
        #expect(repo.calls == 0)
    }

    @Test func withASessionTheHomeIsReadyAndLoadsTheRealStreak() async {
        let model = HomeModel(profiles: FixedRepo(profile: profile(streakDays: 12)), sessions: signedInStore())
        await model.loadStreak()

        #expect(model.gate == .ready)
        #expect(model.streakDays == 12)
        #expect(model.nextCheckinLabel == "hoy 21:00")
    }

    @Test func aSessionSavedWhileTheAppIsUpLandsTheUserOnHome() async {
        // The OTP verify of the onboarding (slice 3) saves the session with
        // the app already running: the reaction must land home, not just at
        // the next launch.
        let store = SessionStore(persisting: InMemoryPersisting())
        let model = HomeModel(profiles: FixedRepo(profile: profile(streakDays: 5)), sessions: store)
        #expect(model.gate == .needsOnboarding)

        // The OTP verify saves the session: the flow continues at the
        // ritual (slice 4) — a session alone no longer lands home.
        store.save(session())
        model.reactToSession()
        #expect(model.gate == .needsOnboarding)

        store.completeOnboarding()
        model.reactToSession()
        #expect(model.gate == .ready)
        await model.loadStreak()
        #expect(model.streakDays == 5)
    }

    @Test func theObservationWiringFiresTheReactionOnAStoredChange() async throws {
        let store = SessionStore(persisting: InMemoryPersisting())
        let model = HomeModel(profiles: FixedRepo(profile: profile(streakDays: 5)), sessions: store)
        #expect(model.gate == .needsOnboarding)

        store.save(session())
        // The reaction re-arms itself on every fire; a short sleep lets the
        // @MainActor task run (there is no deterministic drain API).
        try await Task.sleep(for: .milliseconds(100))
        // A session alone is the ritual, not home (slice 4).
        #expect(model.gate == .needsOnboarding)

        store.completeOnboarding()
        try await Task.sleep(for: .milliseconds(100))
        #expect(model.gate == .ready)

        store.clear()
        try await Task.sleep(for: .milliseconds(100))
        #expect(model.gate == .needsOnboarding)
    }

    @Test func aSessionWipedByTheRefresherWalksTheUserBackToOnboarding() async {
        let store = signedInStore()
        let model = HomeModel(profiles: FixedRepo(profile: profile(streakDays: 12)), sessions: store)
        await model.loadStreak()
        #expect(model.gate == .ready)
        #expect(model.streakDays == 12)

        store.clear()
        model.reactToSession()

        #expect(model.gate == .needsOnboarding)
        // The previous user's data does not leak into the entry point.
        #expect(model.streakDays == 0)
        #expect(model.streakFrozen == false)
        #expect(model.nextCheckinLabel == "hoy 21:00")
    }

    @Test func aTokenRotationDoesNotReloadTheStreak() async throws {
        let store = signedInStore()
        let repo = CountingRepo(profile: profile(streakDays: 12))
        let model = HomeModel(profiles: repo, sessions: store)
        await model.loadStreak()
        #expect(model.gate == .ready)
        // Drain the transition's own load (the init fires it as a Task)
        // before taking the snapshot.
        try await Task.sleep(for: .milliseconds(50))
        let readsBefore = repo.calls

        // The refresher rotates the pair (slice 1) while home is up: the
        // gate stays ready and no new profile read happens.
        store.save(StoredSession(
            accessToken: "access-token-2",
            refreshToken: "refresh-token-2",
            expiresAt: Date(timeIntervalSince1970: 0)
        ))
        model.reactToSession()
        try await Task.sleep(for: .milliseconds(50))

        #expect(model.gate == .ready)
        #expect(repo.calls == readsBefore)
        // The loaded values are untouched by the rotation.
        #expect(model.streakDays == 12)
    }

    // MARK: - streak (R-12, real shape of #76 slice 2)

    @Test func loadsTheStreakFromTheRepository() async {
        let model = HomeModel(profiles: FixedRepo(profile: profile(streakDays: 12)), sessions: signedInStore())
        await model.loadStreak()

        #expect(model.streakDays == 12)
        #expect(model.streakFrozen == false)
    }

    @Test func theNextCheckinLabelFollowsTheProfileSlot() async {
        let model = HomeModel(
            profiles: FixedRepo(profile: profile(streakDays: 3, checkinTime: "08:30")),
            sessions: signedInStore()
        )
        await model.loadStreak()

        #expect(model.nextCheckinLabel == "hoy 08:30")
    }

    @Test func marksTheStreakFrozenWhileThePauseIsLive() async {
        let livePause = Date().addingTimeInterval(3600)
        let model = HomeModel(
            profiles: FixedRepo(profile: profile(streakDays: 7, pauseUntil: livePause)),
            sessions: signedInStore()
        )
        await model.loadStreak()

        #expect(model.streakDays == 7)
        #expect(model.streakFrozen == true)
    }

    @Test func anExpiredPauseDoesNotFreezeTheStreak() async {
        let pastPause = Date().addingTimeInterval(-60)
        let model = HomeModel(
            profiles: FixedRepo(profile: profile(streakDays: 7, pauseUntil: pastPause)),
            sessions: signedInStore()
        )
        await model.loadStreak()

        #expect(model.streakFrozen == false)
    }

    @Test func aFailedProfileReadKeepsTheHomeIntact() async {
        let model = HomeModel(profiles: ExplodingRepo(), sessions: signedInStore())
        await model.loadStreak()

        #expect(model.gate == .ready)
        #expect(model.streakDays == 0)
        #expect(model.streakFrozen == false)
        #expect(model.nextCheckinLabel == "hoy 21:00")
    }

    @Test(
        arguments: [
            ("es", "Estoy OK"),
            ("en", "I'm OK"),
            ("pt", "Estou bem")
        ]
    )
    func callToActionIsLocalized(language: String, expected: String) {
        #expect(localized("checkin.cta", in: language) == expected)
    }

    @Test(
        arguments: [
            ("es", "Siguiente check-in: hoy 21:00"),
            ("en", "Next check-in: hoy 21:00"),
            ("pt", "Próximo check-in: hoy 21:00")
        ]
    )
    func nextCheckinLineIsLocalized(language: String, expected: String) {
        let format = localized("home.next_checkin", in: language)

        #expect(String(format: format, "hoy 21:00") == expected)
    }

    @Test(
        arguments: [
            ("es", "12 días seguidos 🔥"),
            ("en", "12 days in a row 🔥"),
            ("pt", "12 dias seguidos 🔥")
        ]
    )
    func streakLineIsLocalized(language: String, expected: String) {
        let format = localized("home.streak_days", in: language)

        #expect(String(format: format, 12) == expected)
    }

    @Test(
        arguments: [
            ("es", "Pausa activa · racha congelada"),
            ("en", "Pause on · streak frozen"),
            ("pt", "Pausa ativa · ofensiva congelada")
        ]
    )
    func frozenLineIsLocalized(language: String, expected: String) {
        #expect(localized("home.streak_frozen", in: language) == expected)
    }

    /// Reads a key from the app bundle's Localizable.strings for one language.
    /// The test target is hosted in the app, so Bundle.main is the app bundle.
    private func localized(_ key: String, in language: String) -> String {
        guard let path = Bundle.main.path(forResource: language, ofType: "lproj"),
              let languageBundle = Bundle(path: path)
        else {
            return ""
        }
        return languageBundle.localizedString(forKey: key, value: nil, table: nil)
    }
}
