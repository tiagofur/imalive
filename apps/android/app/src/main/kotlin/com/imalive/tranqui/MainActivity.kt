package com.imalive.tranqui

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.imalive.tranqui.feature.checkin.HomeScreen
import com.imalive.tranqui.feature.checkin.HomeViewModel
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    private val viewModel: HomeViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                val uiState by viewModel.uiState.collectAsStateWithLifecycle()
                HomeScreen(
                    uiState = uiState,
                    onCheckInClicked = viewModel::onCheckInClicked,
                )
            }
        }
    }
}
