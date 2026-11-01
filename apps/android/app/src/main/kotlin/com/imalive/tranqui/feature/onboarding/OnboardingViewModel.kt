package com.imalive.tranqui.feature.onboarding

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.imalive.tranqui.core.network.ApiError
import com.imalive.tranqui.core.network.ApiErrorCode
import com.imalive.tranqui.core.network.dto.ProfilePatch
import com.imalive.tranqui.core.session.SessionStore
import com.imalive.tranqui.feature.checkin.AuthRepository
import com.imalive.tranqui.feature.checkin.ContactsRepository
import com.imalive.tranqui.feature.checkin.ProfileRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.io.IOException
import java.time.LocalTime
import java.time.ZoneId
import java.util.Locale
import javax.inject.Inject

// The linear steps of the onboarding (plan of issue #75): no Navigation
// Compose for five lineal destinations without deep links — a single-VM
// step machine keeps every draft alive across back and rotation for free.
// The verify persists the session and walks into the ritual; leaving the
// flow for home is the completion signal (slice 4), never a step here.
enum class OnboardingStep {
    WELCOME,
    PHONE,
    CODE,
    RITUAL,
    CONTACT,
}

// Which long operation is running: the copy differs per phase (brief paso 2:
// "Enviando tu código…" / "Verificando…"; paso 3 guarda el ritual; paso 4
// envía la invitación).
enum class OnboardingBusy {
    SENDING_CODE,
    VERIFYING,
    SAVING_RITUAL,
    SENDING_INVITE,
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

// The ritual's (paso 3) failures: the empty name is caught locally (the brief
// asks for it kindly); the wire failures reuse the human offline/retry copy.
enum class RitualError {
    EMPTY_NAME,
    OFFLINE,
    RETRY,
}

// The contact step's (paso 4) failures: the phone format is caught locally
// (the brief's inline copy); the wire failures reuse the human offline/retry
// copy. The quota (CONTACT_LIMIT_REACHED) is not an error — it resolves with
// copy and a way home, never a paywall (R-13).
enum class ContactError {
    PHONE,
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
    // The ritual (paso 3): the name the welcome message will carry and the
    // daily slot — the native picker proposes 21:00, the timezone is the
    // device's (IANA) and never edited here (brief supuesto).
    val name: String = "",
    val checkinTime: LocalTime = DEFAULT_CHECKIN_TIME,
    val ritualError: RitualError? = null,
    // The just-in-time notification prompt (paso 3, API 33+) is answered
    // once; the card never re-asks (brief paso 3).
    val permissionAnswered: Boolean = false,
    // The contact moment (paso 4): the draft survives back, rotation and
    // error because it lives here.
    val contactName: String = "",
    val contactPhone: String = "",
    val contactError: ContactError? = null,
    // 201: the backend is sending the real welcome — the confirmation
    // replaces the form.
    val contactSent: Boolean = false,
    // CONTACT_LIMIT_REACHED (free plan = 1 contact): copy, not a paywall.
    val limitReached: Boolean = false,
    val busy: OnboardingBusy? = null,
    val phoneError: PhoneError? = null,
    val codeError: CodeError? = null,
    // Live resend countdown: > 0 disables "Enviarlo de nuevo" and shows the
    // rate-limit copy with the seconds (brief paso 2). Fed by Retry-After.
    val rateLimitSeconds: Int = 0,
)

// The contract renders the slot as "HH:MM" (checkinTime pattern). ROOT keeps
// the digits ASCII whatever locale the device runs.
internal fun LocalTime.hhmm(): String = String.format(Locale.ROOT, "%02d:%02d", hour, minute)

// The picker's proposal (brief paso 3): "una hora tranquila para avisar que
// el día cerró bien".
private const val SUGGESTED_CHECKIN_HOUR = 21
private val DEFAULT_CHECKIN_TIME: LocalTime = LocalTime.of(SUGGESTED_CHECKIN_HOUR, 0)

@HiltViewModel
// The plan of issue #75 mandates ONE step machine for the five linear
// destinations; every user intent is a small named function on it, so the
// function count is the design, not a smell.
@Suppress("TooManyFunctions")
class OnboardingViewModel
    @Inject
    constructor(
        private val auth: AuthRepository,
        private val profiles: ProfileRepository,
        private val sessions: SessionStore,
        private val contacts: ContactsRepository,
    ) : ViewModel() {
        private val _uiState = MutableStateFlow(OnboardingUiState())
        val uiState: StateFlow<OnboardingUiState> = _uiState.asStateFlow()

        private var cooldownJob: Job? = null

        init {
            // A process death mid-onboarding (the OTP verify already saved
            // the session) resumes at the ritual: the auth phases are done.
            viewModelScope.launch {
                if (sessions.session.first() != null && _uiState.value.step == OnboardingStep.WELCOME) {
                    _uiState.update { it.copy(step = OnboardingStep.RITUAL) }
                }
            }
        }

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

        // --- the ritual (paso 3) ---

        fun onNameChanged(raw: String) {
            // The server accepts 1..80 characters; the field stops there
            // instead of failing later (the code's six-digit discipline).
            _uiState.update { it.copy(name = raw.take(MAX_NAME_LENGTH), ritualError = null) }
        }

        fun onCheckinTimeChanged(time: LocalTime) {
            _uiState.update { it.copy(checkinTime = time, ritualError = null) }
        }

        fun onNotificationsAnswered() {
            _uiState.update { it.copy(permissionAnswered = true) }
        }

        fun onRitualContinueClicked() {
            val state = _uiState.value
            if (state.busy != null) return
            if (state.name.isBlank()) {
                _uiState.update { it.copy(ritualError = RitualError.EMPTY_NAME) }
                return
            }
            saveRitual()
        }

        // --- the contact moment (paso 4) ---

        fun onContactNameChanged(raw: String) {
            _uiState.update { it.copy(contactName = raw.take(MAX_NAME_LENGTH), contactError = null) }
        }

        fun onContactPhoneChanged(raw: String) {
            // Same discipline as the user's own phone field.
            val phone = raw.filter { it.isDigit() || it == '+' }
            _uiState.update { it.copy(contactPhone = phone, contactError = null) }
        }

        fun onInviteAnotherClicked() {
            _uiState.update {
                it.copy(
                    contactName = "",
                    contactPhone = "",
                    contactError = null,
                    contactSent = false,
                    limitReached = false,
                )
            }
        }

        fun onSendInviteClicked() {
            val state = _uiState.value
            if (state.busy != null || state.contactName.isBlank()) return
            if (!E164.matches(state.contactPhone)) {
                _uiState.update { it.copy(contactError = ContactError.PHONE) }
                return
            }
            sendInvite()
        }

        // The honest ways out of the flow — the skip ("Lo haré después") and
        // the post-send "Ir al inicio" — flip the persisted completion flag;
        // the reactive gate then lands home.
        fun onFinishClicked() {
            viewModelScope.launch { sessions.completeOnboarding() }
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

                OnboardingStep.RITUAL -> _uiState.update { it.copy(step = OnboardingStep.CODE) }

                OnboardingStep.CONTACT -> _uiState.update { it.copy(step = OnboardingStep.RITUAL) }

                else -> Unit // WELCOME leaves the app
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
                    // The session is persisted; since slice 4 the machine
                    // walks into the ritual — the gate leaves onboarding only
                    // when the flow completes (ritual + contact or skip).
                    _uiState.update { it.copy(busy = null, step = OnboardingStep.RITUAL) }
                } catch (error: ApiError) {
                    onApiError(error) {
                        _uiState.update { it.copy(busy = null, codeError = CodeError.RETRY) }
                    }
                } catch (_: IOException) {
                    _uiState.update { it.copy(busy = null, codeError = CodeError.OFFLINE) }
                }
            }
        }

        // The ritual persists through PATCH /v1/me (brief paso 3): the name
        // the welcome message will carry, the chosen slot and the device's
        // IANA timezone (the engine escalates on the user's local slot, not
        // on the server's clock).
        private fun saveRitual() {
            val state = _uiState.value
            viewModelScope.launch {
                _uiState.update { it.copy(busy = OnboardingBusy.SAVING_RITUAL, ritualError = null) }
                try {
                    profiles.update(
                        ProfilePatch(
                            name = state.name.trim(),
                            checkinTime = state.checkinTime.hhmm(),
                            timezone = ZoneId.systemDefault().id,
                        ),
                    )
                    _uiState.update { it.copy(busy = null, step = OnboardingStep.CONTACT) }
                } catch (_: IOException) {
                    _uiState.update { it.copy(busy = null, ritualError = RitualError.OFFLINE) }
                } catch (_: ApiError) {
                    // The locally validated payload cannot fail validation
                    // (name ≤ 80, HH:MM, a valid IANA zone); anything else
                    // asks for a retry in human words.
                    _uiState.update { it.copy(busy = null, ritualError = RitualError.RETRY) }
                }
            }
        }

        // The send hits POST /v1/contacts: the backend messages the contact
        // (R-03) — the app never double-sends. The quota is a state, not an
        // error; the offline draft stays for the retry and the manual
        // WhatsApp fallback.
        private fun sendInvite() {
            val state = _uiState.value
            viewModelScope.launch {
                _uiState.update { it.copy(busy = OnboardingBusy.SENDING_INVITE, contactError = null) }
                try {
                    contacts.create(name = state.contactName.trim(), phone = state.contactPhone)
                    _uiState.update { it.copy(busy = null, contactSent = true, limitReached = false) }
                } catch (error: ApiError) {
                    when (error.code) {
                        ApiErrorCode.CONTACT_LIMIT_REACHED ->
                            _uiState.update { it.copy(busy = null, limitReached = true) }

                        else ->
                            _uiState.update { it.copy(busy = null, contactError = ContactError.RETRY) }
                    }
                } catch (_: IOException) {
                    _uiState.update { it.copy(busy = null, contactError = ContactError.OFFLINE) }
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

            // The server's name limit (POST/PATCH validate 1..80).
            const val MAX_NAME_LENGTH = 80

            // E.164: + followed by 8–15 digits, country code not starting 0.
            val E164 = Regex("""^\+[1-9]\d{7,14}$""")
        }
    }
