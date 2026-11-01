package com.imalive.tranqui.feature.onboarding

import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// Criterion of issue #75 slice 2: without a session MainActivity shows the
// onboarding entry point — the promise of the brief (paso 1) and the start
// CTA, with the definitive copy already in strings.xml.
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class OnboardingPlaceholderScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun theEntryPointShowsThePromiseOfTheBriefAndTheStartCta() {
        composeRule.setContent {
            MaterialTheme {
                OnboardingPlaceholderScreen(onStartClicked = {})
            }
        }

        val promise = "Un toque al día para que los que te quieren sepan que estás bien."
        composeRule.onNodeWithText(promise).assertIsDisplayed()
        composeRule.onNodeWithText("Empezar").assertIsDisplayed()
    }
}
