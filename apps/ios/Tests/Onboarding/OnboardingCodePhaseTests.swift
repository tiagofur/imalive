import Foundation
import Testing
@testable import Tranqui

/// Criteria of issue #76 slice 3 (the code phase): auto-verify at the sixth
/// digit, the contract's codes mapped to human states — OTP_INVALID keeps
/// the written code for rewrite, OTP_EXPIRED auto-requests a fresh one —
/// and the successful verify resets the machine for the reactive gate.
@MainActor
struct OnboardingCodePhaseTests {
    private func waitUntil(
        deadline: Duration = .seconds(5),
        _ condition: @MainActor () -> Bool
    ) async -> Bool {
        await OnboardingHarness.waitUntil(deadline: deadline, condition)
    }

    @Test func verifySendsThePhoneAndTheCodeThroughTheRepository() async {
        let auth = FakeAuth()
        let model = OnboardingHarness.model(auth: auth)
        await OnboardingHarness.reachCodePhase(model, auth: auth)

        model.codeChanged("482913")
        await waitUntil { model.state.busy == nil }

        #expect(auth.verifyCount == 1)
        #expect(auth.verifiedPhone == "+5215512345678")
        #expect(auth.verifiedCode == "482913")
    }

    @Test func onlyTheSixthDigitAutoVerifies() async {
        let auth = FakeAuth()
        let model = OnboardingHarness.model(auth: auth)
        await OnboardingHarness.reachCodePhase(model, auth: auth)

        model.codeChanged("482")
        #expect(auth.verifyCount == 0)

        model.codeChanged("482913")
        await waitUntil { model.state.busy == nil }
        #expect(auth.verifyCount == 1)
    }

    @Test func otpInvalidKeepsTheWrittenCodeForRewriteAndRecoversOnTheNextTry() async {
        let auth = FakeAuth()
        auth.verifyError = OnboardingHarness.apiError(.otpInvalid)
        let model = OnboardingHarness.model(auth: auth)
        await OnboardingHarness.reachCodePhase(model, auth: auth)

        model.codeChanged("111111")
        await waitUntil { model.state.busy == nil }
        #expect(model.state.codeError == .invalid)
        #expect(model.state.code == "111111") // kept, not wiped

        auth.verifyError = nil
        model.codeChanged("482913")
        await waitUntil { model.state.busy == nil && model.state.step == .welcome }

        #expect(model.state.codeError == nil)
        #expect(auth.verifyCount == 2)
    }

    @Test func otpExpiredAutoRequestsAFreshCode() async {
        let auth = FakeAuth()
        auth.verifyError = OnboardingHarness.apiError(.otpExpired)
        let model = OnboardingHarness.model(auth: auth)
        await OnboardingHarness.reachCodePhase(model, auth: auth)

        model.codeChanged("482913")
        await waitUntil { model.state.codeError == .expired }

        #expect(auth.requestCount == 2) // the auto-reenvío fired
        #expect(model.state.step == .code)
    }

    @Test func anUnexpectedBusinessAnswerAsksToRetryInHumanWords() async {
        let auth = FakeAuth()
        auth.verifyError = OnboardingHarness.apiError(.unauthorized, status: 401)
        let model = OnboardingHarness.model(auth: auth)
        await OnboardingHarness.reachCodePhase(model, auth: auth)

        model.codeChanged("482913")
        await waitUntil { model.state.busy == nil }

        #expect(model.state.codeError == .retry)
    }

    @Test func anUnexpectedBusinessAnswerInThePhonePhaseWritesThePhoneSlotNotTheCodeSlot() async {
        let auth = FakeAuth()
        auth.requestError = OnboardingHarness.apiError(.unauthorized, status: 401)
        let model = OnboardingHarness.model(auth: auth)
        model.startTapped()
        model.phoneChanged("+5215512345678")

        model.requestCodeTapped()
        await waitUntil { model.state.busy == nil }

        #expect(model.state.phoneError == .retry)
        #expect(model.state.codeError == nil)
    }

    @Test func anOfflineVerifyKeepsTheCodeAndAsksToRetry() async {
        let auth = FakeAuth()
        auth.verifyError = URLError(.notConnectedToInternet)
        let model = OnboardingHarness.model(auth: auth)
        await OnboardingHarness.reachCodePhase(model, auth: auth)

        model.codeChanged("482913")
        await waitUntil { model.state.busy == nil }

        #expect(model.state.codeError == .offline)
        #expect(model.state.code == "482913")
    }

    @Test func onlySixDigitsOfWhateverArrivesEverReachTheDraft() async {
        let auth = FakeAuth()
        // The verify fails so the draft stays observable after the attempt.
        auth.verifyError = OnboardingHarness.apiError(.otpInvalid)
        let model = OnboardingHarness.model(auth: auth)
        await OnboardingHarness.reachCodePhase(model, auth: auth)

        model.codeChanged("4-8 2a913777")
        await waitUntil { model.state.busy == nil }

        #expect(model.state.code == "482913")
        #expect(auth.verifyCount == 1) // the sixth digit still auto-verifies
    }

    @Test func aSuccessfulVerifyResetsTheMachineSoAWipedSessionReopensTheFlowFresh() async {
        let auth = FakeAuth()
        let model = OnboardingHarness.model(auth: auth)
        await OnboardingHarness.reachCodePhase(model, auth: auth)

        model.codeChanged("482913")
        await waitUntil { model.state.step == .welcome }

        #expect(model.state.phone == "")
        #expect(model.state.code == "")
        #expect(model.state.busy == nil)
    }
}
