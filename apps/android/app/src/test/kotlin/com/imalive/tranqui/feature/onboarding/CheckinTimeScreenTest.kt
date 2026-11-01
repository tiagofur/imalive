package com.imalive.tranqui.feature.onboarding

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.hasScrollAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollToNode
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.imalive.tranqui.core.designsystem.TranquiTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config
import java.time.LocalTime

// The brief's critical states of paso 3 (the ritual): the two questions with
// their copy, the just-in-time permission card whose system prompt fires
// only through the allow action, and the busy/wire-error states. Composed
// against hand-built states, as the OTP screen tests do.
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class CheckinTimeScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private var allowCalls = 0
    private var answeredCalls = 0

    private fun setContent(
        state: OnboardingUiState = OnboardingUiState(step = OnboardingStep.RITUAL),
        showPermissionCard: Boolean = false,
    ) {
        allowCalls = 0
        answeredCalls = 0
        composeRule.setContent {
            TranquiTheme {
                CheckinTimeScreen(
                    uiState = state,
                    onNameChanged = {},
                    onCheckinTimeChanged = {},
                    onContinueClicked = {},
                    onAllowNotifications = { allowCalls++ },
                    onPermissionAnswered = { answeredCalls++ },
                    showPermissionCard = showPermissionCard,
                )
            }
        }
    }

    @Test
    fun theRitualAsksTheNameAndTheHourWithTheBriefCopy() {
        setContent()

        composeRule.onNodeWithText("¿Cómo te llamas?").assertIsDisplayed()
        composeRule.onNodeWithText("¿A qué hora te recordamos cada día?").assertIsDisplayed()
        val timeSub = "Te sugerimos las 21:00: una hora tranquila para avisar que el día cerró bien. Elige la tuya."
        composeRule.onNodeWithText(timeSub).assertIsDisplayed()
        composeRule.onNodeWithText("21:00").assertIsDisplayed()
        composeRule.onNodeWithText("Continuar").assertIsDisplayed()
    }

    @Test
    fun theHourRowAnnouncesItsSlotForAccessibility() {
        setContent(state = OnboardingUiState(step = OnboardingStep.RITUAL, checkinTime = LocalTime.of(8, 30)))

        composeRule
            .onNodeWithContentDescription("Elegir la hora del recordatorio. Actualmente 08:30.")
            .assertIsDisplayed()
    }

    @Test
    fun anEmptyNameAsksForItKindly() {
        setContent(
            OnboardingUiState(step = OnboardingStep.RITUAL, ritualError = RitualError.EMPTY_NAME),
        )

        composeRule
            .onNodeWithText("Escríbenos tu nombre: así te llamará tu gente.")
            .assertIsDisplayed()
    }

    @Test
    fun anOfflineRitualKeepsTheHumanCopy() {
        setContent(
            OnboardingUiState(step = OnboardingStep.RITUAL, ritualError = RitualError.OFFLINE),
        )

        composeRule
            .onNodeWithText("No tienes internet. Revisa tu conexión y prueba otra vez.")
            .assertIsDisplayed()
    }

    @Test
    fun savingShowsTheBusyCopyAndDisablesTheCta() {
        setContent(
            OnboardingUiState(step = OnboardingStep.RITUAL, busy = OnboardingBusy.SAVING_RITUAL),
        )

        composeRule.onNodeWithText("Guardando tus datos…").assertIsDisplayed()
        composeRule.onNodeWithText("Continuar").assertIsNotEnabled()
    }

    // The card sits under the time section, below the fold of the smallest
    // viewport: the test scrolls to it the way a thumb would.
    private fun scrollToPermissionCard() {
        composeRule.onNode(hasScrollAction()).performScrollToNode(hasText("Permitir notificaciones"))
    }

    @Test
    fun thePermissionCardSpeaksBeforeTheCtaWithTheExactSlot() {
        setContent(
            OnboardingUiState(step = OnboardingStep.RITUAL, checkinTime = LocalTime.of(8, 30)),
            showPermissionCard = true,
        )

        scrollToPermissionCard()
        composeRule
            .onNodeWithText("Te avisaremos a las 08:30 con una notificación tranquila. Nada de alarma.")
            .assertIsDisplayed()
        composeRule.onNodeWithText("Permitir notificaciones").assertIsDisplayed()
        composeRule.onNodeWithText("Ahora no").assertIsDisplayed()
    }

    @Test
    fun onlyTheAllowActionFiresTheSystemPrompt() {
        setContent(showPermissionCard = true)

        scrollToPermissionCard()
        composeRule.onNodeWithText("Ahora no").performClick()

        assertEquals(0, allowCalls)
        assertEquals(1, answeredCalls)
    }

    @Test
    fun theAllowActionLaunchesTheSystemPrompt() {
        setContent(showPermissionCard = true)

        scrollToPermissionCard()
        composeRule.onNodeWithText("Permitir notificaciones").performClick()

        assertEquals(1, allowCalls)
        assertEquals(0, answeredCalls)
    }
}
