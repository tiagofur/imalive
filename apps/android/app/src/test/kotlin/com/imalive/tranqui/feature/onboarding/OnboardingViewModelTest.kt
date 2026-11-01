package com.imalive.tranqui.feature.onboarding

import com.imalive.tranqui.core.network.ApiError
import com.imalive.tranqui.core.network.ApiErrorCode
import com.imalive.tranqui.feature.checkin.AuthRepository
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.Dispatchers
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

// Criteria of issue #75 slice 3: the onboarding is a linear step machine on
// one VM — steps advance and go back with drafts intact, the errors of the
// contract map to human states (never the raw detail) and the resend
// cooldown counts down from Retry-After. The fakes stand in for the auth
// repository; the wire behavior is covered by NetworkRepositoriesTest.
class OnboardingViewModelTest {
    private val mainDispatcher = UnconfinedTestDispatcher()

    @Before
    fun setUp() {
        Dispatchers.setMain(mainDispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

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
            val viewModel = OnboardingViewModel(FakeAuth())

            assertEquals(OnboardingStep.WELCOME, viewModel.uiState.value.step)
            viewModel.onStartClicked()

            assertEquals(OnboardingStep.PHONE, viewModel.uiState.value.step)
        }

    @Test
    fun `requesting the code advances to the code phase and clears the busy flag`() =
        runTest {
            val auth = FakeAuth()
            val viewModel = OnboardingViewModel(auth)

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
            val viewModel = OnboardingViewModel(auth)
            viewModel.reachCodePhase(auth)
            viewModel.onCodeChanged("123456")

            viewModel.onBack()

            assertEquals(OnboardingStep.PHONE, viewModel.uiState.value.step)
            assertEquals("+5215512345678", viewModel.uiState.value.phone)
            assertEquals("", viewModel.uiState.value.code)
            assertNull(viewModel.uiState.value.codeError)
        }

    @Test
    fun `back from the phone phase returns to welcome`() =
        runTest {
            val viewModel = OnboardingViewModel(FakeAuth())
            viewModel.onStartClicked()

            viewModel.onBack()

            assertEquals(OnboardingStep.WELCOME, viewModel.uiState.value.step)
        }

    // --- the phone phase and its errors ---

    @Test
    fun `a malformed phone never reaches the api and asks to review it`() =
        runTest {
            val auth = FakeAuth()
            val viewModel = OnboardingViewModel(auth)
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
            val viewModel = OnboardingViewModel(auth)
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
            val viewModel = OnboardingViewModel(auth)
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
            val viewModel = OnboardingViewModel(auth)
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
            val viewModel = OnboardingViewModel(auth)
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
            val viewModel = OnboardingViewModel(auth)
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
            val viewModel = OnboardingViewModel(auth)
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
            val viewModel = OnboardingViewModel(auth)
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
            val viewModel = OnboardingViewModel(auth)
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
            val viewModel = OnboardingViewModel(auth)
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
            val viewModel = OnboardingViewModel(auth)
            viewModel.reachCodePhase(auth)

            viewModel.onCodeChanged("482913")

            assertEquals(CodeError.RETRY, viewModel.uiState.value.codeError)
        }

    @Test
    fun `an offline verify keeps the code and asks to retry`() =
        runTest {
            val auth = FakeAuth()
            auth.verifyError = IOException("network down")
            val viewModel = OnboardingViewModel(auth)
            viewModel.reachCodePhase(auth)

            viewModel.onCodeChanged("482913")

            assertEquals(CodeError.OFFLINE, viewModel.uiState.value.codeError)
            assertEquals("482913", viewModel.uiState.value.code)
        }

    @Test
    fun `only six digits of whatever arrives ever reach the draft`() =
        runTest {
            val auth = FakeAuth()
            val viewModel = OnboardingViewModel(auth)
            viewModel.reachCodePhase(auth)

            viewModel.onCodeChanged("4-8 2a913777")

            assertEquals("482913", viewModel.uiState.value.code)
            assertEquals(1, auth.verifyCalls)
        }
}
