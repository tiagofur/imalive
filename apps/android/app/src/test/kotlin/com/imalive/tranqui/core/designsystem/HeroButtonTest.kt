package com.imalive.tranqui.core.designsystem

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// The hero CTA (OkButton DNA): pill, one per screen, honors enabled and
// answers the tap — rendered under the provisional TranquiTheme.
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class HeroButtonTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun rendersItsLabelInsideTheThemeAndAnswersTaps() {
        var taps = 0
        composeRule.setContent {
            TranquiTheme {
                HeroButton(text = "Empezar", onClick = { taps++ })
            }
        }

        composeRule.onNodeWithText("Empezar").assertIsDisplayed()
        composeRule.onNodeWithText("Empezar").performClick()
        assertEquals(1, taps)
    }

    @Test
    fun aDisabledHeroButtonSaysSo() {
        composeRule.setContent {
            TranquiTheme {
                HeroButton(text = "Enviarme el código", onClick = {}, enabled = false)
            }
        }

        composeRule.onNodeWithText("Enviarme el código").assertIsNotEnabled()
    }

    @Test
    fun anEnabledHeroButtonSaysSo() {
        composeRule.setContent {
            TranquiTheme {
                HeroButton(text = "Entrar", onClick = {}, enabled = true)
            }
        }

        composeRule.onNodeWithText("Entrar").assertIsEnabled()
    }
}
