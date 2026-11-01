package com.imalive.tranqui.feature.onboarding

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.imalive.tranqui.core.network.ApiError
import com.imalive.tranqui.core.network.ApiErrorCode
import com.imalive.tranqui.feature.checkin.AuthRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.io.IOException
import javax.inject.Inject

// The linear steps of the onboarding (plan of issue #75): no Navigation
// Compose for five lineal destinations without deep links — a single-VM
// step machine keeps every draft alive across back and rotation for free.
// RITUAL and CONTACT land in slice 4; the gate (slice 2) already leaves
// onboarding for home the moment the OTP verify persists the session.
enum class OnboardingStep {
    WELCOME,
    PHONE,
    CODE,
    RITUAL,
    CONTACT,
}

// Which long operation is running: the copy differs per phase (brief paso 2:
// "Enviando tu código…" / "Verificando…").
enum class OnboardingBusy {
    SENDING_CODE,
    VERIFYING,
}

enum class PhoneError {
    // The E.164 check (local or server PHONE_INVALID) failed.
    INVALID,

    // No network: the draft stays, the user retries.
    OFFLINE,

    // An unexpected business answer: human copy, never the raw detail.
    RETRY,
}

enum class CodeError {
    // OTP_INVALID: the written code stays, selected for rewrite.
    INVALID,

    // OTP_EXPIRED: a new code was sent automatically.
    EXPIRED,

    OFFLINE,
    RETRY,
}

// Immutable, unidirectional: the screens render this and answer back through
// the VM only. Phone/code drafts survive back and rotation because they live
// here (plan of issue #75).
data class OnboardingUiState(
    val step: OnboardingStep = OnboardingStep.WELCOME,
    val phone: String = "",
    val code: String = "",
    val busy: OnboardingBusy? = null,
    val phoneError: PhoneError? = null,
    val codeError: CodeError? = null,
    // Live resend countdown: > 0 disables "Enviarlo de nuevo" and shows the
    // rate-limit copy with the seconds (brief paso 2). Fed by Retry-After.
    val rateLimitSeconds: Int = 0,
)

@HiltViewModel
class OnboardingViewModel
    @Inject
    constructor(
        private val auth: AuthRepository,
    ) : ViewModel() {
        private val _uiState = MutableStateFlow(OnboardingUiState())
        val uiState: StateFlow<OnboardingUiState> = _uiState.asStateFlow()

        private var cooldownJob: Job? = null

        fun onStartClicked() {
            _uiState.update { it.copy(step = OnboardingStep.PHONE, phoneError = null) }
        }

        fun onPhoneChanged(raw: String) {
            // The field accepts the display characters a keyboard inserts;
            // only digits and the leading + reach the draft.
            val phone = raw.filter { it.isDigit() || it == '+' }
            _uiState.update { it.copy(phone = phone, phoneError = null) }
        }

        fun onRequestCodeClicked() {
            val state = _uiState.value
            if (state.busy != null || state.rateLimitSeconds > 0) return
            if (!E164.matches(state.phone)) {
                _uiState.update { it.copy(phoneError = PhoneError.INVALID) }
                return
            }
            sendCode()
        }

        fun onResendClicked() {
            val state = _uiState.value
            if (state.busy != null || state.rateLimitSeconds > 0) return
            sendCode()
        }

        fun onCodeChanged(raw: String) {
            val code = raw.filter(Char::isDigit).take(CODE_LENGTH)
            _uiState.update { it.copy(code = code, codeError = null) }
            // Auto-verify at the sixth digit (brief paso 2); the "Entrar"
            // button is the backup for when autofill never arrives.
            if (code.length == CODE_LENGTH) verify()
        }

        fun onVerifyClicked() {
            if (_uiState.value.busy != null) return
            verify()
        }

        // Predictive back walks the linear machine backwards; the phone draft
        // survives, a half-written code does not (a stale code can only fail).
        fun onBack() {
            when (_uiState.value.step) {
                OnboardingStep.CODE ->
                    _uiState.update {
                        it.copy(
                            step = OnboardingStep.PHONE,
                            code = "",
                            codeError = null,
                        )
                    }

                OnboardingStep.PHONE -> _uiState.update { it.copy(step = OnboardingStep.WELCOME) }

                else -> Unit // WELCOME leaves the app; RITUAL/CONTACT belong to slice 4
            }
        }

        private fun sendCode(markExpiredOnSuccess: Boolean = false) {
            val phone = _uiState.value.phone
            viewModelScope.launch {
                _uiState.update {
                    it.copy(busy = OnboardingBusy.SENDING_CODE, phoneError = null, codeError = null)
                }
                try {
                    auth.requestOtp(phone)
                    _uiState.update {
                        it.copy(
                            busy = null,
                            step = OnboardingStep.CODE,
                            // The brief's expiry flow: when the fresh code of
                            // the auto-reenvío lands, the copy says so.
                            codeError = if (markExpiredOnSuccess) CodeError.EXPIRED else null,
                        )
                    }
                } catch (error: ApiError) {
                    onApiError(error) {
                        _uiState.update { it.copy(busy = null, phoneError = PhoneError.RETRY) }
                    }
                } catch (_: IOException) {
                    _uiState.update { it.copy(busy = null, phoneError = PhoneError.OFFLINE) }
                }
            }
        }

        private fun verify() {
            val state = _uiState.value
            if (state.busy != null || state.code.length < CODE_LENGTH) return
            viewModelScope.launch {
                _uiState.update { it.copy(busy = OnboardingBusy.VERIFYING, codeError = null) }
                try {
                    auth.verifyOtp(state.phone, state.code)
                    // The session is persisted; the reactive gate swaps
                    // onboarding for home — nothing to navigate from here.
                    _uiState.update { it.copy(busy = null) }
                } catch (error: ApiError) {
                    onApiError(error) {
                        _uiState.update { it.copy(busy = null, codeError = CodeError.RETRY) }
                    }
                } catch (_: IOException) {
                    _uiState.update { it.copy(busy = null, codeError = CodeError.OFFLINE) }
                }
            }
        }

        // The brief: the app switches on the stable Problem.code and never
        // shows the raw detail (ux-brief §6). The fallback belongs to the
        // phase that was in flight — a non-mapped code during the request
        // must not write into the code slot where the user can't see it.
        private fun onApiError(
            error: ApiError,
            fallback: () -> Unit,
        ) {
            when (error.code) {
                ApiErrorCode.PHONE_INVALID ->
                    _uiState.update { it.copy(busy = null, phoneError = PhoneError.INVALID) }

                ApiErrorCode.OTP_INVALID ->
                    _uiState.update { it.copy(busy = null, codeError = CodeError.INVALID) }

                ApiErrorCode.OTP_EXPIRED -> {
                    // Auto-reenvío (brief paso 2): a fresh code goes out; when
                    // it lands, the copy confirms it ("El código caducó. Te
                    // enviamos uno nuevo.").
                    sendCode(markExpiredOnSuccess = true)
                }

                ApiErrorCode.RATE_LIMITED -> {
                    _uiState.update { it.copy(busy = null) }
                    // Real countdown from the Retry-After header (or the
                    // 60 s default when the header is missing or invalid,
                    // plan of issue #75): it disables the resend while live.
                    cooldownJob?.cancel()
                    cooldownJob =
                        viewModelScope.launch {
                            for (remaining in (error.retryAfterSeconds ?: DEFAULT_COOLDOWN_SECONDS) downTo 1) {
                                _uiState.update { it.copy(rateLimitSeconds = remaining) }
                                delay(COOLDOWN_TICK_MS)
                            }
                            _uiState.update { it.copy(rateLimitSeconds = 0) }
                        }
                }

                else -> fallback()
            }
        }

        private companion object {
            const val CODE_LENGTH = 6
            const val DEFAULT_COOLDOWN_SECONDS = 60
            const val COOLDOWN_TICK_MS = 1_000L

            // E.164: + followed by 8–15 digits, country code not starting 0.
            val E164 = Regex("""^\+[1-9]\d{7,14}$""")
        }
    }
