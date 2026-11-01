package com.imalive.tranqui

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.imalive.tranqui.feature.checkin.HomeGate
import com.imalive.tranqui.feature.checkin.HomeScreen
import com.imalive.tranqui.feature.checkin.HomeViewModel
import com.imalive.tranqui.feature.onboarding.OnboardingPlaceholderScreen
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    private val viewModel: HomeViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                val uiState by viewModel.uiState.collectAsStateWithLifecycle()
                // The session gate (issue #75 slice 2): onboarding entry point
                // without a session, home with one. Slice 3 mounts the real
                // onboarding flow under the same gate.
                when (uiState.gate) {
                    HomeGate.LOADING -> Unit // first DataStore read: a blank frame, no spinner
                    HomeGate.NEEDS_ONBOARDING ->
                        OnboardingPlaceholderScreen(
                            onStartClicked = {}, // real flow lands with slice 3
                        )
                    HomeGate.READY ->
                        HomeScreen(
                            uiState = uiState,
                            onCheckInClicked = viewModel::onCheckInClicked,
                        )
                }
            }
        }
    }
}
