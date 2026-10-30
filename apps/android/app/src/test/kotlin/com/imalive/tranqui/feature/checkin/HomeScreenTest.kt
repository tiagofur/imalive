package com.imalive.tranqui.feature.checkin

import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// Criterion of issue #3 (triaged): a Compose test asserting the "Estoy OK"
// button text, running inside testDebugUnitTest (Robolectric, no emulator).
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class HomeScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun homeShowsTheOkButtonAndTheNextCheckinStatus() {
        composeRule.setContent {
            MaterialTheme {
                HomeScreen(
                    uiState = HomeUiState(nextCheckinLabel = "hoy 21:00"),
                    onCheckInClicked = {},
                )
            }
        }

        composeRule.onNodeWithText("Estoy OK").assertIsDisplayed()
        composeRule.onNodeWithText("Siguiente check-in: hoy 21:00").assertIsDisplayed()
    }
}
