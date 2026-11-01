package com.imalive.tranqui.feature.onboarding

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.imalive.tranqui.core.designsystem.TranquiTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// Criterion of issue #75 slice 3 (paso 1 del brief): the welcome is the
// emotional promise — title, sub, the hero CTA and the honest disclaimer in
// a low voice, with "Tranqui" interpolated as the product-name placeholder.
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class WelcomeScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun thePromiseEarnsTheFirstTapWithTheBriefCopy() {
        var started = false
        composeRule.setContent {
            TranquiTheme {
                WelcomeScreen(onStartClicked = { started = true })
            }
        }

        composeRule
            .onNodeWithText("Un toque al día para que los que te quieren sepan que estás bien.")
            .assertIsDisplayed()
        val sub =
            "Si un día no confirmas, tu gente lo sabe al instante " +
                "y alguien puede comprobar cómo estás."
        composeRule.onNodeWithText(sub).assertIsDisplayed()
        composeRule.onNodeWithText("Empezar").assertIsDisplayed()
        composeRule.onNodeWithText("Empezar").performClick()
        assertEquals(true, started)
    }

    @Test
    fun theDisclaimerSpeaksInALowVoiceWithTheAppNameInterpolated() {
        composeRule.setContent {
            TranquiTheme {
                WelcomeScreen(onStartClicked = {})
            }
        }

        val disclaimer =
            "Tranqui avisa a tu gente; " +
                "no es un servicio de emergencia y no sustituye al 911/112."
        composeRule.onNodeWithText(disclaimer).assertIsDisplayed()
    }
}
