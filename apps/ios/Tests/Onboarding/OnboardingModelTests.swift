import Foundation
import Testing
@testable import Tranqui

/// Criteria of issue #76 slice 3 (the machine and the phone phase): the
/// onboarding is a linear step machine on one model — steps advance and go
/// back with drafts intact, and the contract's Problem.codes map to human
/// states (never the raw detail).
@MainActor
struct OnboardingModelTests {
    private func model(auth: FakeAuth = FakeAuth(), tick: Duration = OnboardingHarness.frozenTick) -> OnboardingModel {
        OnboardingHarness.model(auth: auth, tick: tick)
    }

    private func waitUntil(
        deadline: Duration = .seconds(5),
        _ condition: @MainActor () -> Bool
    ) async -> Bool {
        await OnboardingHarness.waitUntil(deadline: deadline, condition)
    }

    // MARK: - the linear machine

    @Test func startingMovesTheMachineFromWelcomeToPhone() {
        let model = model()

        #expect(model.state.step == .welcome)
        model.startTapped()

        #expect(model.state.step == .phone)
    }

    @Test func requestingTheCodeAdvancesToTheCodePhaseAndClearsTheBusyFlag() async {
        let auth = FakeAuth()
        let model = model(auth: auth)

        model.startTapped()
        model.phoneChanged("+5215512345678")
        model.requestCodeTapped()

        await waitUntil { model.state.busy == nil }
        #expect(model.state.step == .code)
        #expect(model.state.busy == nil)
        #expect(auth.requestCount == 1)
    }

    @Test func backFromTheCodePhaseKeepsThePhoneDraftButNotTheStaleCode() async {
        let auth = FakeAuth()
        auth.verifyError = OnboardingHarness.apiError(.otpInvalid)
        let model = model(auth: auth)
        await OnboardingHarness.reachCodePhase(model, auth: auth)
        model.codeChanged("123456")
        await waitUntil { model.state.busy == nil }

        model.navigationPathChanged([.phone])

        #expect(model.state.step == .phone)
        #expect(model.state.phone == "+5215512345678")
        #expect(model.state.code == "")
        #expect(model.state.codeError == nil)
    }

    @Test func backFromThePhonePhaseReturnsToWelcome() {
        let model = model()
        model.startTapped()

        model.navigationPathChanged([])

        #expect(model.state.step == .welcome)
        // The phone draft survives: the number is still valid if the user
        // moves forward again.
        #expect(model.state.phone == "")
    }

    @Test func theNavigationPathMirrorsTheMachineForwards() async {
        let model = model()

        #expect(model.navigationPath() == [])
        model.startTapped()
        #expect(model.navigationPath() == [.phone])
        model.phoneChanged("+5215512345678")
        model.requestCodeTapped()
        await waitUntil { model.state.step == .code }

        #expect(model.navigationPath() == [.phone, .code])
    }

    @Test func aForwardPathChangeNeverPushesTheMachine() {
        let model = model()

        model.navigationPathChanged([.phone, .code])

        #expect(model.state.step == .welcome)
    }

    // MARK: - the phone phase and its errors

    @Test func aMalformedPhoneNeverReachesTheApiAndAsksToReviewIt() {
        let auth = FakeAuth()
        let model = model(auth: auth)
        model.startTapped()
        model.phoneChanged("55 1234 5678") // no country code

        model.requestCodeTapped()

        #expect(model.state.phoneError == .invalid)
        #expect(auth.requestCount == 0)
    }

    @Test func phoneInvalidFromTheApiAsksToReviewTheNumberAndStays() async {
        let auth = FakeAuth()
        auth.requestError = OnboardingHarness.apiError(.phoneInvalid)
        let model = model(auth: auth)
        model.startTapped()
        model.phoneChanged("+5215512345678")

        model.requestCodeTapped()
        await waitUntil { model.state.busy == nil }

        #expect(model.state.phoneError == .invalid)
        #expect(model.state.step == .phone)
        #expect(model.state.busy == nil)
    }

    @Test func anOfflineRequestKeepsThePhoneDraftAndAsksToRetry() async {
        let auth = FakeAuth()
        auth.requestError = URLError(.notConnectedToInternet)
        let model = model(auth: auth)
        model.startTapped()
        model.phoneChanged("+5215512345678")

        model.requestCodeTapped()
        await waitUntil { model.state.busy == nil }

        #expect(model.state.phoneError == .offline)
        #expect(model.state.phone == "+5215512345678")
    }

    @Test func aBusyMachineIgnoresExtraTaps() async {
        let auth = FakeAuth()
        auth.holdNextRequest = true
        let model = model(auth: auth)
        model.startTapped()
        model.phoneChanged("+5215512345678")

        model.requestCodeTapped()
        model.requestCodeTapped()
        model.requestCodeTapped()

        await waitUntil { auth.requestCount == 1 }
        #expect(model.state.busy == .sendingCode)
        auth.release()
        await waitUntil { model.state.busy == nil }
        #expect(auth.requestCount == 1)
    }

    // MARK: - the rate limit (Retry-After)

    @Test func the429StartsTheRetryAfterCountdownAndBlocksTheResend() async {
        let auth = FakeAuth()
        auth.requestError = OnboardingHarness.apiError(.rateLimited, status: 429, retryAfter: 47)
        let model = model(auth: auth)
        model.startTapped()
        model.phoneChanged("+5215512345678")

        model.requestCodeTapped()
        await waitUntil { model.state.rateLimitSeconds > 0 }

        // The countdown starts exactly at the Retry-After value (the tick is
        // frozen, so nothing decrements it under the assertion).
        #expect(model.state.rateLimitSeconds == 47)

        model.requestCodeTapped() // blocked while cooling down
        model.resendTapped()
        #expect(auth.requestCount == 1)
    }

    @Test func the429WithoutRetryAfterFallsBackToTheDefaultCooldown() async {
        let auth = FakeAuth()
        auth.requestError = OnboardingHarness.apiError(.rateLimited, status: 429)
        let model = model(auth: auth)
        model.startTapped()
        model.phoneChanged("+5215512345678")

        model.requestCodeTapped()
        await waitUntil { model.state.rateLimitSeconds > 0 }

        #expect(model.state.rateLimitSeconds == 60)
    }

    @Test func theCooldownCountsDownLiveAndUnblocksTheResend() async {
        let auth = FakeAuth()
        auth.requestError = OnboardingHarness.apiError(.rateLimited, status: 429, retryAfter: 3)
        let model = model(auth: auth, tick: .milliseconds(2))
        model.startTapped()
        model.phoneChanged("+5215512345678")

        model.requestCodeTapped()
        await waitUntil { model.state.rateLimitSeconds > 0 } // the error landed
        await waitUntil { model.state.rateLimitSeconds == 0 } // the live drain

        auth.requestError = nil
        model.requestCodeTapped()
        await waitUntil { model.state.step == .code }
        #expect(auth.requestCount == 2)
    }
}
