package com.imalive.tranqui.feature.onboarding

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.imalive.tranqui.feature.onboarding.component.OnboardingProgress

// Root of the onboarding flow (plan of issue #75): the VM's step machine
// owns the navigation — the BackHandler walks it backwards so predictive
// back keeps the drafts. The OTP verify persists the session and the
// reactive gate (slice 2) swaps this flow for home; no navigation call
// happens here. Edge-to-edge (enforced on API 35+): the progress clears
// the status bar and the CTA clears the gesture area (brief, Android note).
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun OnboardingScreen(
    viewModel: OnboardingViewModel,
    modifier: Modifier = Modifier,
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    BackHandler(enabled = uiState.step != OnboardingStep.WELCOME) { viewModel.onBack() }

    Column(
        modifier =
            modifier
                .fillMaxSize()
                .statusBarsPadding()
                .navigationBarsPadding()
                .imePadding(),
    ) {
        OnboardingProgress(currentStep = uiState.step.progressSegment())
        when (uiState.step) {
            OnboardingStep.WELCOME -> WelcomeScreen(onStartClicked = viewModel::onStartClicked)
            OnboardingStep.PHONE ->
                OtpRequestScreen(
                    uiState = uiState,
                    onPhoneChanged = viewModel::onPhoneChanged,
                    onRequestCodeClicked = viewModel::onRequestCodeClicked,
                )
            OnboardingStep.CODE ->
                OtpVerifyScreen(
                    uiState = uiState,
                    onCodeChanged = viewModel::onCodeChanged,
                    onVerifyClicked = viewModel::onVerifyClicked,
                    onResendClicked = viewModel::onResendClicked,
                )

            // Slice 4 lands the ritual and the "añade a tu mamá" moment.
            OnboardingStep.RITUAL, OnboardingStep.CONTACT -> Unit
        }
    }
}

// Welcome is segment 1; the two OTP phases share segment 2 (brief: 4 pasos
// visibles, teléfono y código son el paso 2).
private const val SEGMENT_WELCOME = 1
private const val SEGMENT_AUTH = 2
private const val SEGMENT_RITUAL = 3
private const val SEGMENT_CONTACT = 4

private fun OnboardingStep.progressSegment(): Int =
    when (this) {
        OnboardingStep.WELCOME -> SEGMENT_WELCOME
        OnboardingStep.PHONE, OnboardingStep.CODE -> SEGMENT_AUTH
        OnboardingStep.RITUAL -> SEGMENT_RITUAL
        OnboardingStep.CONTACT -> SEGMENT_CONTACT
    }
