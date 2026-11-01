package com.imalive.tranqui.feature.checkin

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import java.time.Instant
import javax.inject.Inject

// The home state (R-04 + R-12): the next-checkin schedule and the streak
// the user is protecting. nextCheckinLabel keeps the default copy until the
// profile read answers with the real slot (GET /v1/me, issue #75 slice 2).
data class HomeUiState(
    val nextCheckinLabel: String = "hoy 21:00",
    val streakDays: Int = 0,
    val streakFrozen: Boolean = false,
)

@HiltViewModel
class HomeViewModel
    @Inject
    constructor(
        private val profiles: ProfileRepository,
    ) : ViewModel() {
        private val _uiState = MutableStateFlow(HomeUiState())
        val uiState: StateFlow<HomeUiState> = _uiState.asStateFlow()

        init {
            // The streak is decoration: a failed read keeps the defaults and
            // never sinks the home (the identity is the button, not the fire).
            viewModelScope.launch {
                val defaults = _uiState.value
                _uiState.value =
                    try {
                        val profile = profiles.profile()
                        defaults.copy(
                            nextCheckinLabel = "hoy ${profile.checkinTime}",
                            streakDays = profile.streakDays,
                            // The pause (R-11) freezes the streak while live;
                            // an expired pauseUntil reads as no pause.
                            streakFrozen = profile.pauseUntil?.isAfter(Instant.now()) == true,
                        )
                    } catch (_: Exception) {
                        defaults // offline or business error: home intact
                    }
            }
        }

        @Suppress("EmptyFunctionBlock") // intentional seam: wired to POST /v1/checkins in the MVP
        fun onCheckInClicked() {
            // R-04: the tap will hit the contract's POST /v1/checkins (issue #2);
            // until the check-in flow lands, the tap stays a visual seam.
        }
    }
