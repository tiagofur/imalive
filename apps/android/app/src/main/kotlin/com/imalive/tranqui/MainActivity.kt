package com.imalive.tranqui

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.imalive.tranqui.core.designsystem.TranquiTheme
import com.imalive.tranqui.feature.checkin.HomeGate
import com.imalive.tranqui.feature.checkin.HomeScreen
import com.imalive.tranqui.feature.checkin.HomeViewModel
import com.imalive.tranqui.feature.onboarding.OnboardingScreen
import com.imalive.tranqui.feature.onboarding.OnboardingViewModel
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    private val homeViewModel: HomeViewModel by viewModels()
    private val onboardingViewModel: OnboardingViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            TranquiTheme {
                val uiState by homeViewModel.uiState.collectAsStateWithLifecycle()
                // The session gate (issue #75 slice 2) decides what renders:
                // the onboarding flow without a session, home with one. The
                // OTP verify saves the session and the gate swaps the flow
                // for home reactively — no navigation call involved.
                when (uiState.gate) {
                    HomeGate.LOADING -> Unit // first DataStore read: a blank frame, no spinner
                    HomeGate.NEEDS_ONBOARDING -> OnboardingScreen(viewModel = onboardingViewModel)
                    HomeGate.READY ->
                        HomeScreen(
                            uiState = uiState,
                            onCheckInClicked = homeViewModel::onCheckInClicked,
                        )
                }
            }
        }
    }
}
