import SwiftUI

@main
struct TranquiApp: App {
    @State private var sessions: SessionStore
    @State private var home: HomeModel
    @State private var onboarding: OnboardingModel

    init() {
        // One stack for the whole app: the Keychain-backed session store
        // feeds the API's bearer attach and refresh AND the reactive gate
        // both surfaces render from (issue #76 slices 2-3). The onboarding's
        // verify saves the session through the same store the home reads.
        let sessions = SessionStore(persisting: Self.sessionPersisting())
        let api = TranquiAPI(sessions: sessions)
        _sessions = State(initialValue: sessions)
        _home = State(initialValue: HomeModel(
            profiles: NetworkProfileRepository(api: api),
            sessions: sessions
        ))
        _onboarding = State(initialValue: OnboardingModel(
            auth: NetworkAuthRepository(api: api, sessions: sessions)
        ))
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(home)
                .environment(onboarding)
        }
    }

    /// Production persistence: the Keychain. The UITest smoke override
    /// `-tranqui-session-stub signed-in|signed-out` boots the gate over an
    /// ephemeral store (seeded with a canned session, or empty) without
    /// touching the Keychain — test tokens are not real sessions. The full
    /// stubbed API mode lands with the onboarding flow (slice 4); with
    /// only this override the home shows its defaults when the profile read
    /// cannot reach the network, which is exactly the resilience it must have.
    private static func sessionPersisting() -> any SessionPersisting {
        let arguments = ProcessInfo.processInfo.arguments
        guard let index = arguments.firstIndex(of: "-tranqui-session-stub") else {
            return KeychainSessionStore()
        }
        let persisting = EphemeralSessionPersisting()
        if arguments.dropFirst(index + 1).first == "signed-in" {
            persisting.save(StoredSession(
                accessToken: "smoke-access-token",
                refreshToken: "smoke-refresh-token",
                expiresAt: Date().addingTimeInterval(900)
            ))
        }
        return persisting
    }
}

/// In-memory persistence for the smoke mode only: process-local, nothing
/// reaches the Keychain.
private final class EphemeralSessionPersisting: SessionPersisting, @unchecked Sendable {
    private let lock = NSLock()
    private var stored: StoredSession?

    func load() -> StoredSession? {
        lock.withLock { stored }
    }

    func save(_ session: StoredSession) {
        lock.withLock { stored = session }
    }

    func clear() {
        lock.withLock { stored = nil }
    }
}
