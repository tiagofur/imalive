package com.imalive.tranqui.feature.onboarding.component

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.imalive.tranqui.core.designsystem.TranquiTheme
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// The ritual progress (brief): a thin 4-segment line whose accessible name
// announces "Paso N de 4" — the count lives in the semantics, not on screen.
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class OnboardingProgressTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun theAccessibleNameAnnouncesTheStepOfFour() {
        composeRule.setContent {
            TranquiTheme {
                OnboardingProgress(currentStep = 2)
            }
        }

        composeRule.onNodeWithContentDescription("Paso 2 de 4").assertIsDisplayed()
    }

    @Test
    fun theLastStepAnnouncesFourOfFour() {
        composeRule.setContent {
            TranquiTheme {
                OnboardingProgress(currentStep = 4)
            }
        }

        composeRule.onNodeWithContentDescription("Paso 4 de 4").assertIsDisplayed()
    }
}
