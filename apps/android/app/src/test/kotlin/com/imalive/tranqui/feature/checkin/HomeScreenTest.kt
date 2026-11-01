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

// Criteria of issues #3 and #72: the "Estoy OK" button, the next-checkin
// status and the streak the user is protecting, running inside
// testDebugUnitTest (Robolectric, no emulator).
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

    @Test
    fun homeShowsTheStreakTheUserIsProtecting() {
        composeRule.setContent {
            MaterialTheme {
                HomeScreen(
                    uiState = HomeUiState(nextCheckinLabel = "hoy 21:00", streakDays = 12),
                    onCheckInClicked = {},
                )
            }
        }

        composeRule.onNodeWithText("12 días seguidos 🔥").assertIsDisplayed()
    }

    @Test
    fun theArrivalFromOnboardingReadsTheZeroAsABeginning() {
        composeRule.setContent {
            MaterialTheme {
                HomeScreen(
                    uiState = HomeUiState(nextCheckinLabel = "hoy 21:00", streakDays = 0),
                    onCheckInClicked = {},
                )
            }
        }

        composeRule.onNodeWithText("Tu racha empieza hoy.").assertIsDisplayed()
    }

    @Test
    fun aFrozenStreakSaysSo() {
        composeRule.setContent {
            MaterialTheme {
                HomeScreen(
                    uiState =
                        HomeUiState(
                            nextCheckinLabel = "hoy 21:00",
                            streakDays = 7,
                            streakFrozen = true,
                        ),
                    onCheckInClicked = {},
                )
            }
        }

        composeRule.onNodeWithText("7 días seguidos 🔥").assertIsDisplayed()
        composeRule.onNodeWithText("Pausa activa · racha congelada").assertIsDisplayed()
    }
}
