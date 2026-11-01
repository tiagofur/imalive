package com.imalive.tranqui.feature.checkin

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.imalive.tranqui.core.session.SessionStore
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.launch
import java.time.Instant
import javax.inject.Inject

// Which surface MainActivity renders, decided by the persisted session
// (issue #75 slice 2: the gate of the home).
enum class HomeGate {
    // The session store has not answered yet (first DataStore read).
    LOADING,

    // No session, or the session exists but the onboarding has not finished
    // (slice 4): the onboarding flow owns the screen.
    NEEDS_ONBOARDING,

    // Session present: the home, with the real streak.
    READY,
}

// The home state (R-04 + R-12): the session gate, the next-checkin schedule
// and the streak the user is protecting. nextCheckinLabel keeps the default
// copy until the profile read answers with the real slot (GET /v1/me).
data class HomeUiState(
    val gate: HomeGate = HomeGate.LOADING,
    val nextCheckinLabel: String = "hoy 21:00",
    val streakDays: Int = 0,
    val streakFrozen: Boolean = false,
)

@HiltViewModel
class HomeViewModel
    @Inject
    constructor(
        private val profiles: ProfileRepository,
        private val sessions: SessionStore,
    ) : ViewModel() {
        private val _uiState = MutableStateFlow(HomeUiState())
        val uiState: StateFlow<HomeUiState> = _uiState.asStateFlow()

        init {
            // The gate collects (not reads once): the first store answer
            // decides where the app starts. A session alone is not enough
            // (slice 4): the onboarding must finish first — the ritual saves
            // the profile and the trusted contact (or the honest skip) calls
            // completeOnboarding(). Then home lands; a session wiped by the
            // authenticator (rejected refresh) walks them back. Token
            // rotations also emit here — they must not reload anything.
            viewModelScope.launch {
                combine(sessions.session, sessions.onboardingComplete, ::Pair).collect { pair ->
                    val (session, complete) = pair
                    val gate = _uiState.value.gate
                    when {
                        (session == null || !complete) && gate != HomeGate.NEEDS_ONBOARDING ->
                            _uiState.value = HomeUiState(gate = HomeGate.NEEDS_ONBOARDING)

                        session != null && complete && gate != HomeGate.READY -> {
                            _uiState.value = _uiState.value.copy(gate = HomeGate.READY)
                            loadStreak()
                        }

                        // else: a token rotation rewrote the store while the
                        // home is up (or nothing changed) — nothing to reload.
                        else -> Unit
                    }
                }
            }
        }

        // The streak is decoration: a failed read keeps the defaults and
        // never sinks the home (the identity is the button, not the fire).
        private suspend fun loadStreak() {
            val defaults = _uiState.value
            _uiState.value =
                try {
                    val profile = profiles.profile()
                    defaults.copy(
                        nextCheckinLabel = "hoy ${profile.checkinTime}",
                        streakDays = profile.streakDays,
                        // The pause (R-11) freezes the streak while live; an
                        // expired pauseUntil reads as no pause.
                        streakFrozen = profile.pauseUntil?.isAfter(Instant.now()) == true,
                    )
                } catch (_: Exception) {
                    defaults // offline or business error: home intact
                }
        }

        @Suppress("EmptyFunctionBlock") // intentional seam: wired to POST /v1/checkins in the MVP
        fun onCheckInClicked() {
            // R-04: the tap will hit the contract's POST /v1/checkins (issue #2);
            // until the check-in flow lands, the tap stays a visual seam.
        }
    }
