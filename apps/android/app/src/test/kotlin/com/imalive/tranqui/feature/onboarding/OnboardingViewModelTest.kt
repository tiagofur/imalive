package com.imalive.tranqui.feature.onboarding

import com.imalive.tranqui.core.network.ApiError
import com.imalive.tranqui.core.network.ApiErrorCode
import com.imalive.tranqui.core.network.FakeSessionStore
import com.imalive.tranqui.core.network.dto.Contact
import com.imalive.tranqui.core.network.dto.ProfilePatch
import com.imalive.tranqui.core.session.StoredSession
import com.imalive.tranqui.feature.checkin.AuthRepository
import com.imalive.tranqui.feature.checkin.ContactsRepository
import com.imalive.tranqui.feature.checkin.Profile
import com.imalive.tranqui.feature.checkin.ProfileRepository
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.advanceTimeBy
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import java.io.IOException
import java.time.LocalTime
import java.time.ZoneId

// Shared by the onboarding test files of this package: an in-memory auth
// repository the tests program with errors and counters (the wire behavior
// is covered by NetworkRepositoriesTest).
internal class FakeAuth : AuthRepository {
    var requestCalls = 0
        private set

    var verifyCalls = 0
        private set

    var lastVerifyPhone: String? = null
        private set

    var lastVerifyCode: String? = null
        private set

    var requestError: Exception? = null
    var verifyError: Exception? = null
    var gateRequest: CompletableDeferred<Unit>? = null

    override suspend fun requestOtp(phone: String) {
        requestCalls++
        gateRequest?.await()
        requestError?.let { throw it }
    }

    override suspend fun verifyOtp(
        phone: String,
        code: String,
    ) {
        verifyCalls++
        lastVerifyPhone = phone
        lastVerifyCode = code
        verifyError?.let { throw it }
    }
}

// The profile seam the ritual step patches (PATCH /v1/me).
internal class FakeProfiles : ProfileRepository {
    var updateCalls = 0
        private set

    var lastPatch: ProfilePatch? = null
        private set

    var error: Exception? = null

    override suspend fun profile(): Profile = Profile(streakDays = 0, checkinTime = "21:00")

    override suspend fun update(patch: ProfilePatch): Profile {
        updateCalls++
        lastPatch = patch
        error?.let { throw it }
        return Profile(streakDays = 0, checkinTime = patch.checkinTime ?: "21:00")
    }
}

// The circle seam the contact moment posts to (POST /v1/contacts).
internal class FakeContacts : ContactsRepository {
    var createCalls = 0
        private set

    var lastName: String? = null
        private set

    var lastPhone: String? = null
        private set

    var error: Exception? = null

    override suspend fun contacts(): List<Contact> = emptyList()

    override suspend fun create(
        name: String,
        phone: String,
    ): Contact {
        createCalls++
        lastName = name
        lastPhone = phone
        error?.let { throw it }
        return Contact(id = "01934f2f", name = name, phone = phone, createdAt = "2026-11-01T00:00:00Z")
    }
}

// Criteria of issue #75 slice 3: the onboarding is a linear step machine on
// one VM — steps advance and go back with drafts intact, the errors of the
// contract map to human states (never the raw detail) and the resend
// cooldown counts down from Retry-After. The fakes stand in for the auth
// repository; the wire behavior is covered by NetworkRepositoriesTest.
class OnboardingViewModelTest {
    private val mainDispatcher = UnconfinedTestDispatcher()

    // The session the OTP verify saves (what a mid-onboarding process death
    // leaves behind).
    private val savedSession =
        StoredSession(
            accessToken = "access-token-1",
            refreshToken = "refresh-token-1",
            expiresAtEpochSeconds = 0L,
            userPhone = "+5215512345678",
        )

    @Before
    fun setUp() {
        Dispatchers.setMain(mainDispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    // The slice-3 machine ran on the auth seam alone; since slice 4 the VM
    // also owns the profile (the ritual), the session (resume + completion)
    // and the contacts. Tests default every seam to a fresh fake.
    private fun viewModel(
        auth: FakeAuth = FakeAuth(),
        profiles: FakeProfiles = FakeProfiles(),
        sessions: FakeSessionStore = FakeSessionStore(),
        contacts: FakeContacts = FakeContacts(),
    ): OnboardingViewModel = OnboardingViewModel(auth, profiles, sessions, contacts)

    // Drives the machine to the code phase the way the UI does.
    private fun OnboardingViewModel.reachCodePhase(auth: FakeAuth) {
        onStartClicked()
        onPhoneChanged("+5215512345678")
        onRequestCodeClicked()
        assertEquals(1, auth.requestCalls)
        assertEquals(OnboardingStep.CODE, uiState.value.step)
    }

    // --- the linear machine ---

    @Test
    fun `empezar moves the machine from welcome to phone`() =
        runTest {
            val viewModel = viewModel()

            assertEquals(OnboardingStep.WELCOME, viewModel.uiState.value.step)
            viewModel.onStartClicked()

            assertEquals(OnboardingStep.PHONE, viewModel.uiState.value.step)
        }

    @Test
    fun `requesting the code advances to the code phase and clears the busy flag`() =
        runTest {
            val auth = FakeAuth()
            val viewModel = viewModel(auth)

            viewModel.onStartClicked()
            viewModel.onPhoneChanged("+5215512345678")
            viewModel.onRequestCodeClicked()

            assertEquals(OnboardingStep.CODE, viewModel.uiState.value.step)
            assertNull(viewModel.uiState.value.busy)
        }

    @Test
    fun `back from the code phase keeps the phone draft but not the stale code`() =
        runTest {
            val auth = FakeAuth()
            auth.verifyError = ApiError(ApiErrorCode.OTP_INVALID, httpStatus = 400)
            val viewModel = viewModel(auth)
            viewModel.reachCodePhase(auth)
            viewModel.onCodeChanged("123456") // a successful verify walks into the ritual

            viewModel.onBack()

            assertEquals(OnboardingStep.PHONE, viewModel.uiState.value.step)
            assertEquals("+5215512345678", viewModel.uiState.value.phone)
            assertEquals("", viewModel.uiState.value.code)
            assertNull(viewModel.uiState.value.codeError)
        }

    @Test
    fun `back from the phone phase returns to welcome`() =
        runTest {
            val viewModel = viewModel()
            viewModel.onStartClicked()

            viewModel.onBack()

            assertEquals(OnboardingStep.WELCOME, viewModel.uiState.value.step)
        }

    // --- the phone phase and its errors ---

    @Test
    fun `a malformed phone never reaches the api and asks to review it`() =
        runTest {
            val auth = FakeAuth()
            val viewModel = viewModel(auth)
            viewModel.onStartClicked()
            viewModel.onPhoneChanged("55 1234 5678") // no country code

            viewModel.onRequestCodeClicked()

            assertEquals(PhoneError.INVALID, viewModel.uiState.value.phoneError)
            assertEquals(0, auth.requestCalls)
        }

    @Test
    fun `phone invalid from the api asks to review the number and stays`() =
        runTest {
            val auth = FakeAuth()
            auth.requestError = ApiError(ApiErrorCode.PHONE_INVALID, httpStatus = 400)
            val viewModel = viewModel(auth)
            viewModel.onStartClicked()
            viewModel.onPhoneChanged("+5215512345678")

            viewModel.onRequestCodeClicked()

            assertEquals(PhoneError.INVALID, viewModel.uiState.value.phoneError)
            assertEquals(OnboardingStep.PHONE, viewModel.uiState.value.step)
            assertNull(viewModel.uiState.value.busy)
        }

    @Test
    fun `an offline request keeps the phone draft and asks to retry`() =
        runTest {
            val auth = FakeAuth()
            auth.requestError = IOException("airplane mode")
            val viewModel = viewModel(auth)
            viewModel.onStartClicked()
            viewModel.onPhoneChanged("+5215512345678")

            viewModel.onRequestCodeClicked()

            assertEquals(PhoneError.OFFLINE, viewModel.uiState.value.phoneError)
            assertEquals("+5215512345678", viewModel.uiState.value.phone)
        }

    @Test
    fun `a busy machine ignores extra taps`() =
        runTest {
            val auth = FakeAuth()
            auth.gateRequest = CompletableDeferred()
            val viewModel = viewModel(auth)
            viewModel.onStartClicked()
            viewModel.onPhoneChanged("+5215512345678")

            viewModel.onRequestCodeClicked()
            viewModel.onRequestCodeClicked()

            assertEquals(1, auth.requestCalls)
            auth.gateRequest?.complete(Unit)
            runCurrent()
            assertNull(viewModel.uiState.value.busy)
        }

    // --- the rate limit (Retry-After) ---

    @Test
    fun `429 starts the retry-after countdown, decrements live and blocks the resend`() =
        runTest {
            val auth = FakeAuth()
            auth.requestError = ApiError(ApiErrorCode.RATE_LIMITED, httpStatus = 429, retryAfterSeconds = 47)
            val viewModel = viewModel(auth)
            viewModel.onStartClicked()
            viewModel.onPhoneChanged("+5215512345678")
            viewModel.onRequestCodeClicked()

            assertEquals(47, viewModel.uiState.value.rateLimitSeconds)

            viewModel.onRequestCodeClicked() // blocked while cooling down
            assertEquals(1, auth.requestCalls)

            advanceTimeBy(1_000)
            runCurrent()
            assertEquals(46, viewModel.uiState.value.rateLimitSeconds)

            advanceTimeBy(60_000)
            runCurrent()
            assertEquals(0, viewModel.uiState.value.rateLimitSeconds)
        }

    @Test
    fun `429 without retry-after falls back to the default cooldown`() =
        runTest {
            val auth = FakeAuth()
            auth.requestError = ApiError(ApiErrorCode.RATE_LIMITED, httpStatus = 429, retryAfterSeconds = null)
            val viewModel = viewModel(auth)
            viewModel.onStartClicked()
            viewModel.onPhoneChanged("+5215512345678")

            viewModel.onRequestCodeClicked()

            assertEquals(60, viewModel.uiState.value.rateLimitSeconds)
        }

    // --- the code phase and its errors ---

    @Test
    fun `verify persists the session through the repository`() =
        runTest {
            val auth = FakeAuth()
            val viewModel = viewModel(auth)
            viewModel.reachCodePhase(auth)

            viewModel.onCodeChanged("482913")

            assertEquals(1, auth.verifyCalls)
            assertEquals("+5215512345678", auth.lastVerifyPhone)
            assertEquals("482913", auth.lastVerifyCode)
            assertNull(viewModel.uiState.value.busy)
        }

    @Test
    fun `only the sixth digit auto-verifies`() =
        runTest {
            val auth = FakeAuth()
            val viewModel = viewModel(auth)
            viewModel.reachCodePhase(auth)

            viewModel.onCodeChanged("482")
            assertEquals(0, auth.verifyCalls)

            viewModel.onCodeChanged("482913")
            assertEquals(1, auth.verifyCalls)
        }

    @Test
    fun `otp invalid keeps the written code for rewrite and recovers on the next try`() =
        runTest {
            val auth = FakeAuth()
            auth.verifyError = ApiError(ApiErrorCode.OTP_INVALID, httpStatus = 400)
            val viewModel = viewModel(auth)
            viewModel.reachCodePhase(auth)

            viewModel.onCodeChanged("111111")

            assertEquals(CodeError.INVALID, viewModel.uiState.value.codeError)
            assertEquals("111111", viewModel.uiState.value.code) // selected, not wiped

            auth.verifyError = null
            viewModel.onCodeChanged("482913")

            assertNull(viewModel.uiState.value.codeError)
            assertEquals(2, auth.verifyCalls)
        }

    @Test
    fun `otp expired auto-requests a fresh code`() =
        runTest {
            val auth = FakeAuth()
            auth.verifyError = ApiError(ApiErrorCode.OTP_EXPIRED, httpStatus = 400)
            val viewModel = viewModel(auth)
            viewModel.reachCodePhase(auth)

            viewModel.onCodeChanged("482913")

            assertEquals(CodeError.EXPIRED, viewModel.uiState.value.codeError)
            assertEquals(2, auth.requestCalls) // the auto-reenvío fired
        }

    @Test
    fun `an unexpected business answer asks to retry in human words`() =
        runTest {
            val auth = FakeAuth()
            auth.verifyError = ApiError(ApiErrorCode.UNAUTHORIZED, httpStatus = 401)
            val viewModel = viewModel(auth)
            viewModel.reachCodePhase(auth)

            viewModel.onCodeChanged("482913")

            assertEquals(CodeError.RETRY, viewModel.uiState.value.codeError)
        }

    @Test
    fun `an unexpected business answer in the phone phase writes the phone slot, not the code slot`() =
        runTest {
            val auth = FakeAuth()
            auth.requestError = ApiError(ApiErrorCode.UNAUTHORIZED, httpStatus = 401)
            val viewModel = viewModel(auth)
            viewModel.onStartClicked()
            viewModel.onPhoneChanged("+5215512345678")

            viewModel.onRequestCodeClicked()

            assertEquals(PhoneError.RETRY, viewModel.uiState.value.phoneError)
            assertNull(viewModel.uiState.value.codeError)
        }

    @Test
    fun `an offline verify keeps the code and asks to retry`() =
        runTest {
            val auth = FakeAuth()
            auth.verifyError = IOException("network down")
            val viewModel = viewModel(auth)
            viewModel.reachCodePhase(auth)

            viewModel.onCodeChanged("482913")

            assertEquals(CodeError.OFFLINE, viewModel.uiState.value.codeError)
            assertEquals("482913", viewModel.uiState.value.code)
        }

    @Test
    fun `only six digits of whatever arrives ever reach the draft`() =
        runTest {
            val auth = FakeAuth()
            val viewModel = viewModel(auth)
            viewModel.reachCodePhase(auth)

            viewModel.onCodeChanged("4-8 2a913777")

            assertEquals("482913", viewModel.uiState.value.code)
            assertEquals(1, auth.verifyCalls)
        }

    // --- the ritual (paso 3, slice 4) ---

    // Drives the machine to the ritual the way the UI does: the verify
    // persists the session (the gate lands home), the flow continues here.
    private fun OnboardingViewModel.reachRitual() {
        onStartClicked()
        onPhoneChanged("+5215512345678")
        onRequestCodeClicked()
        onCodeChanged("482913")
        onNameChanged("Ana")
    }

    @Test
    fun `the ritual proposes 21 oclock and continues to the contact step`() =
        runTest {
            val profiles = FakeProfiles()
            val viewModel = viewModel(profiles = profiles)
            viewModel.onStartClicked()
            viewModel.onPhoneChanged("+5215512345678")
            viewModel.onRequestCodeClicked()
            viewModel.onCodeChanged("482913")

            assertEquals(LocalTime.of(21, 0), viewModel.uiState.value.checkinTime)

            viewModel.onNameChanged("Ana")
            viewModel.onRitualContinueClicked()

            assertEquals(OnboardingStep.CONTACT, viewModel.uiState.value.step)
            assertEquals(1, profiles.updateCalls)
        }

    @Test
    fun `continuing the ritual patches name, chosen hour and the device timezone`() =
        runTest {
            val profiles = FakeProfiles()
            val viewModel = viewModel(profiles = profiles)
            viewModel.reachRitual()
            viewModel.onCheckinTimeChanged(LocalTime.of(8, 30))

            viewModel.onRitualContinueClicked()

            val patch = profiles.lastPatch!!
            assertEquals("Ana", patch.name)
            assertEquals("08:30", patch.checkinTime)
            assertEquals(ZoneId.systemDefault().id, patch.timezone)
            assertNull(patch.escalationWindowMinutes)
        }

    @Test
    fun `an empty name never reaches the api and asks for it kindly`() =
        runTest {
            val profiles = FakeProfiles()
            val viewModel = viewModel(profiles = profiles)
            viewModel.reachRitual()
            viewModel.onNameChanged("   ")

            viewModel.onRitualContinueClicked()

            assertEquals(RitualError.EMPTY_NAME, viewModel.uiState.value.ritualError)
            assertEquals(0, profiles.updateCalls)
        }

    @Test
    fun `a name longer than the server limit stops at eighty characters`() =
        runTest {
            val viewModel = viewModel()
            viewModel.reachRitual()
            viewModel.onNameChanged("A".repeat(90))

            assertEquals(80, viewModel.uiState.value.name.length)
        }

    @Test
    fun `an offline ritual keeps the draft and asks to retry`() =
        runTest {
            val profiles = FakeProfiles()
            profiles.error = IOException("airplane mode")
            val viewModel = viewModel(profiles = profiles)
            viewModel.reachRitual()

            viewModel.onRitualContinueClicked()

            assertEquals(RitualError.OFFLINE, viewModel.uiState.value.ritualError)
            assertEquals(OnboardingStep.RITUAL, viewModel.uiState.value.step)
            assertEquals("Ana", viewModel.uiState.value.name)
            assertNull(viewModel.uiState.value.busy)
        }

    @Test
    fun `an unexpected ritual answer asks to retry in human words`() =
        runTest {
            val profiles = FakeProfiles()
            profiles.error = ApiError(ApiErrorCode.UNAUTHORIZED, httpStatus = 401)
            val viewModel = viewModel(profiles = profiles)
            viewModel.reachRitual()

            viewModel.onRitualContinueClicked()

            assertEquals(RitualError.RETRY, viewModel.uiState.value.ritualError)
        }

    @Test
    fun `answering the permission prompt once never asks again`() =
        runTest {
            val viewModel = viewModel()

            viewModel.onNotificationsAnswered()

            assertEquals(true, viewModel.uiState.value.permissionAnswered)
        }

    @Test
    fun `a process death mid-onboarding resumes at the ritual`() =
        runTest {
            // The OTP verify already persisted the session before the death.
            val sessions = FakeSessionStore(savedSession)
            val viewModel = viewModel(sessions = sessions)

            assertEquals(OnboardingStep.RITUAL, viewModel.uiState.value.step)
        }

    @Test
    fun `back from the ritual returns to the code phase`() =
        runTest {
            val viewModel = viewModel()
            viewModel.reachRitual()

            viewModel.onBack()

            assertEquals(OnboardingStep.CODE, viewModel.uiState.value.step)
        }

    // --- the contact moment (paso 4, slice 4) ---

    // Drives the machine to the contact step the way the UI does.
    private fun OnboardingViewModel.reachContact() {
        reachRitual()
        onRitualContinueClicked()
    }

    @Test
    fun `the invite posts the name and the e164 phone and confirms`() =
        runTest {
            val contacts = FakeContacts()
            val viewModel = viewModel(contacts = contacts)
            viewModel.reachContact()
            viewModel.onContactNameChanged("María")
            viewModel.onContactPhoneChanged("+52 155 987 6543")

            viewModel.onSendInviteClicked()

            assertEquals(1, contacts.createCalls)
            assertEquals("María", contacts.lastName)
            assertEquals("+521559876543", contacts.lastPhone)
            assertEquals(true, viewModel.uiState.value.contactSent)
            assertNull(viewModel.uiState.value.busy)
        }

    @Test
    fun `a phone without country never reaches the api and asks to complete it`() =
        runTest {
            val contacts = FakeContacts()
            val viewModel = viewModel(contacts = contacts)
            viewModel.reachContact()
            viewModel.onContactNameChanged("María")
            viewModel.onContactPhoneChanged("55 9876 5432")

            viewModel.onSendInviteClicked()

            assertEquals(ContactError.PHONE, viewModel.uiState.value.contactError)
            assertEquals(0, contacts.createCalls)
        }

    @Test
    fun `contact limit reached resolves with the copy state, not an error`() =
        runTest {
            val contacts = FakeContacts()
            contacts.error = ApiError(ApiErrorCode.CONTACT_LIMIT_REACHED, httpStatus = 422)
            val viewModel = viewModel(contacts = contacts)
            viewModel.reachContact()
            viewModel.onContactNameChanged("María")
            viewModel.onContactPhoneChanged("+521559876543")

            viewModel.onSendInviteClicked()

            assertEquals(true, viewModel.uiState.value.limitReached)
            assertEquals(false, viewModel.uiState.value.contactSent)
            assertNull(viewModel.uiState.value.contactError)
        }

    @Test
    fun `an offline invite keeps the draft and asks to retry`() =
        runTest {
            val contacts = FakeContacts()
            contacts.error = IOException("network down")
            val viewModel = viewModel(contacts = contacts)
            viewModel.reachContact()
            viewModel.onContactNameChanged("María")
            viewModel.onContactPhoneChanged("+521559876543")

            viewModel.onSendInviteClicked()

            assertEquals(ContactError.OFFLINE, viewModel.uiState.value.contactError)
            assertEquals("María", viewModel.uiState.value.contactName)
            assertEquals("+521559876543", viewModel.uiState.value.contactPhone)
            assertEquals(false, viewModel.uiState.value.contactSent)
        }

    @Test
    fun `an unexpected business answer in the contact phase writes the contact slot`() =
        runTest {
            val contacts = FakeContacts()
            contacts.error = ApiError(ApiErrorCode.VALIDATION_FAILED, httpStatus = 400)
            val viewModel = viewModel(contacts = contacts)
            viewModel.reachContact()
            viewModel.onContactNameChanged("María")
            viewModel.onContactPhoneChanged("+521559876543")

            viewModel.onSendInviteClicked()

            assertEquals(ContactError.RETRY, viewModel.uiState.value.contactError)
        }

    @Test
    fun `inviting another clears the confirmation and the draft`() =
        runTest {
            val viewModel = viewModel()
            viewModel.reachContact()
            viewModel.onContactNameChanged("María")
            viewModel.onContactPhoneChanged("+521559876543")
            viewModel.onSendInviteClicked()

            viewModel.onInviteAnotherClicked()

            assertEquals(false, viewModel.uiState.value.contactSent)
            assertEquals("", viewModel.uiState.value.contactName)
            assertEquals("", viewModel.uiState.value.contactPhone)
        }

    @Test
    fun `the skip completes the onboarding so the gate lands home`() =
        runTest {
            val sessions = FakeSessionStore(savedSession)
            val viewModel = viewModel(sessions = sessions)
            viewModel.reachContact()

            viewModel.onFinishClicked()

            assertEquals(true, sessions.onboardingComplete.first())
        }

    @Test
    fun `back from the contact step returns to the ritual`() =
        runTest {
            val viewModel = viewModel()
            viewModel.reachContact()

            viewModel.onBack()

            assertEquals(OnboardingStep.RITUAL, viewModel.uiState.value.step)
        }
}
