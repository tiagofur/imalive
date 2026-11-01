import Foundation
import Testing
@testable import Tranqui

/// Shared harness of the onboarding model suites: a fresh machine per call,
/// a drive to the code phase, a poll-based drain and compact ApiError
/// fixtures (the verbose memberwise shape only serves type completeness).
@MainActor
enum OnboardingHarness {
    /// The countdown freezes at its exact starting value under this tick:
    /// nothing decrements it inside the test's wall time.
    static let frozenTick = Duration.seconds(3600)

    static func model(auth: FakeAuth = FakeAuth(), tick: Duration = OnboardingHarness.frozenTick) -> OnboardingModel {
        let model = OnboardingModel(
            auth: auth,
            profiles: StubProfileRepo(),
            contacts: StubContactsRepo(),
            sessions: SessionStore(persisting: InMemoryPersisting())
        )
        model.cooldownTick = tick
        return model
    }

    /// A ritual build with explicit seams, for the ritual suites.
    static func ritualModel(
        auth: FakeAuth = FakeAuth(),
        profiles: StubProfileRepo = StubProfileRepo(),
        contacts: StubContactsRepo = StubContactsRepo(),
        sessions: SessionStore
    ) -> OnboardingModel {
        OnboardingModel(auth: auth, profiles: profiles, contacts: contacts, sessions: sessions)
    }

    /// Drives the machine to the code phase the way the UI does.
    static func reachCodePhase(_ model: OnboardingModel, auth: FakeAuth) async {
        model.startTapped()
        model.phoneChanged("+5215512345678")
        model.requestCodeTapped()
        await waitUntil { model.state.step == .code && model.state.busy == nil }
        #expect(auth.requestCount == 1)
    }

    /// Drains the model's in-flight tasks: polls until the condition holds
    /// or the deadline passes (Swift Testing has no deterministic drain).
    static func waitUntil(
        deadline: Duration = .seconds(5),
        _ condition: @MainActor () -> Bool
    ) async -> Bool {
        let clock = ContinuousClock()
        let start = clock.now
        while !condition() {
            guard clock.now - start < deadline else { return false }
            try? await Task.sleep(for: .milliseconds(2))
        }
        return true
    }

    /// A compact Problem envelope for the tests that only care about the
    /// code (and sometimes the Retry-After).
    static func apiError(_ code: ApiErrorCode, status: Int = 400, retryAfter: Int? = nil) -> ApiError {
        ApiError(code: code, httpStatus: status, retryAfterSeconds: retryAfter, type: nil, detail: nil)
    }
}
