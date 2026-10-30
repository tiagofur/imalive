package com.imalive.tranqui.feature.checkin

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

// The home state (R-04 + R-12): the next-checkin schedule and the streak
// the user is protecting. nextCheckinLabel is static copy until the network
// client lands with the MVP (issue #75 wires the auth flow).
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
                _uiState.value =
                    try {
                        val profile = profiles.profile()
                        _uiState.value.copy(streakDays = profile.streakDays, streakFrozen = profile.streakFrozen)
                    } catch (_: Exception) {
                        _uiState.value // defaults: no streak shown, home intact
                    }
            }
        }

        @Suppress("EmptyFunctionBlock") // intentional seam: wired to POST /v1/checkins in the MVP
        fun onCheckInClicked() {
            // R-04: the tap will hit the contract's POST /v1/checkins (issue #2).
            // F0 has no network client yet, so the tap stays a visual seam.
        }
    }
