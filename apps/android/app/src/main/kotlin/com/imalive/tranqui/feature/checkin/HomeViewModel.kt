package com.imalive.tranqui.feature.checkin

import androidx.lifecycle.ViewModel
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject

// F0 scaffold state: the next-checkin schedule is static copy until the
// backend client lands with the MVP (R-04, contract in /contracts).
data class HomeUiState(
    val nextCheckinLabel: String = "hoy 21:00",
)

@HiltViewModel
class HomeViewModel
    @Inject
    constructor() : ViewModel() {
        private val _uiState = MutableStateFlow(HomeUiState())
        val uiState: StateFlow<HomeUiState> = _uiState.asStateFlow()

        @Suppress("EmptyFunctionBlock") // intentional seam: wired to POST /v1/checkins in the MVP
        fun onCheckInClicked() {
            // R-04: the tap will hit the contract's POST /v1/checkins (issue #2).
            // F0 has no network client yet, so the tap stays a visual seam.
        }
    }
