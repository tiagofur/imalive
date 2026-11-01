import Foundation
import Testing
@testable import Tranqui

/// The ritual suites (issue #76 slice 4): the name+time PATCH, the contact
/// moment, the quota copy, the honest skip — every wire effect on fakes.
@MainActor
struct RitualTests {
    private func signedInStore() -> SessionStore {
        let store = SessionStore(persisting: InMemoryPersisting())
        store.save(StoredSession(
            accessToken: "access",
            refreshToken: "refresh",
            expiresAt: Date(timeIntervalSince1970: 0)
        ))
        return store
    }

    @Test func theRitualSavesNameTimeAndDeviceTimezone() async {
        let profiles = StubProfileRepo()
        let model = OnboardingHarness.ritualModel(profiles: profiles, sessions: signedInStore())
        model.ritualNameChanged("  Ana  ")
        model.ritualTimeChanged(minutesFromMidnight: 22 * 60 + 30)

        model.ritualContinueTapped()
        await OnboardingHarness.waitUntil { model.state.step == .contact && model.state.busy == nil }

        #expect(profiles.updates.count == 1)
        #expect(profiles.updates[0].name == "Ana")
        #expect(profiles.updates[0].checkinTime == "22:30")
        #expect(profiles.updates[0].timezone == TimeZone.current.identifier)
        #expect(model.state.step == .contact)
    }

    @Test func aBlankNameStaysInTheRitualWithError() async {
        let profiles = StubProfileRepo()
        let model = OnboardingHarness.ritualModel(profiles: profiles, sessions: signedInStore())

        model.ritualContinueTapped()
        await OnboardingHarness.waitUntil { model.state.busy == nil }

        #expect(model.state.step == .ritual)
        #expect(model.state.ritualError == .invalidName)
        #expect(profiles.updates.isEmpty)
    }

    @Test func anOfflineRitualKeepsTheDraftForRetry() async {
        let profiles = StubProfileRepo()
        profiles.updateError = URLError(.notConnectedToInternet)
        let model = OnboardingHarness.ritualModel(profiles: profiles, sessions: signedInStore())
        model.ritualNameChanged("Ana")
        model.ritualContinueTapped()
        await OnboardingHarness.waitUntil { model.state.busy == nil }

        #expect(model.state.ritualError == .offline)
        #expect(model.state.step == .ritual)
        #expect(model.state.name == "Ana")
    }

    @Test func theInvitationCreatesTheContactAndFinishesTheOnboarding() async {
        let contacts = StubContactsRepo()
        let sessions = signedInStore()
        let model = OnboardingHarness.ritualModel(contacts: contacts, sessions: sessions)
        model.jumpTo(.contact)
        model.ritualNameChanged("Ana") // the inviter name for the preview
        model.contactNameChanged("María")
        model.contactPhoneChanged("+5215511111111")

        model.inviteTapped()
        await OnboardingHarness.waitUntil { model.state.savedContactName != nil }

        #expect(contacts.created.count == 1)
        #expect(contacts.created[0].name == "María")
        #expect(contacts.created[0].phone == "+5215511111111")
        #expect(model.state.savedContactName == "María")
        #expect(sessions.onboardingComplete)
        #expect(sessions.gate == .ready)
    }

    @Test func theHonestSkipFinishesWithoutContact() {
        let sessions = signedInStore()
        let model = OnboardingHarness.ritualModel(sessions: sessions)

        model.skipContactTapped()

        #expect(sessions.onboardingComplete)
        #expect(sessions.gate == .ready)
    }

    @Test func theQuotaIsCopyNotError() async {
        let contacts = StubContactsRepo()
        contacts.limitReached = true
        let sessions = signedInStore()
        let model = OnboardingHarness.ritualModel(contacts: contacts, sessions: sessions)
        model.jumpTo(.contact)
        model.contactNameChanged("Otra")
        model.contactPhoneChanged("+5215522222222")

        model.inviteTapped()
        await OnboardingHarness.waitUntil { model.state.quotaNotice }

        #expect(model.state.contactError == nil)
        #expect(model.state.quotaNotice)
        #expect(sessions.onboardingComplete) // the flow still closes
    }

    @Test func aBadPhoneStaysInTheFormWithError() {
        let contacts = StubContactsRepo()
        let model = OnboardingHarness.ritualModel(contacts: contacts, sessions: signedInStore())
        model.jumpTo(.contact)
        model.contactNameChanged("María")
        model.contactPhoneChanged("551234") // no + country code

        model.inviteTapped()

        #expect(model.state.contactError == .invalidPhone)
        #expect(contacts.created.isEmpty)
    }

    @Test func aProcessDeathMidOnboardingResumesAtTheRitual() {
        // The store boots with the session the OTP verify persisted: the
        // machine must open at the ritual, never at welcome.
        let sessions = signedInStore()
        let model = OnboardingHarness.ritualModel(sessions: sessions)

        #expect(model.state.step == .ritual)
    }
}
