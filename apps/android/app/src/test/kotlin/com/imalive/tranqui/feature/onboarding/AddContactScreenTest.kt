package com.imalive.tranqui.feature.onboarding

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.hasScrollAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.v2.createComposeRule
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

// The brief's critical states of paso 4 — "añade a tu mamá" (issue #75
// slice 4): the question that opens the step, the personalized CTA and
// bubble, the honest skip, the quota without paywall and the confirmation
// that speaks to the other. Composed against hand-built states, as the rest
// of the onboarding screen tests do.
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class AddContactScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private var sendCalls = 0
    private var inviteAnotherCalls = 0
    private var finishCalls = 0

    private fun setContent(state: OnboardingUiState) {
        sendCalls = 0
        inviteAnotherCalls = 0
        finishCalls = 0
        composeRule.setContent {
            TranquiTheme {
                AddContactScreen(
                    uiState = state,
                    onContactNameChanged = {},
                    onContactPhoneChanged = {},
                    onSendInviteClicked = { sendCalls++ },
                    onInviteAnotherClicked = { inviteAnotherCalls++ },
                    onFinishClicked = { finishCalls++ },
                )
            }
        }
    }

    private val emptyState =
        OnboardingUiState(
            step = OnboardingStep.CONTACT,
            name = "Ana",
        )

    @Test
    fun theMomentAsksTheBriefQuestionAndPreviewsTheBubble() {
        setContent(emptyState)

        composeRule.onNodeWithText("¿Quién te pregunta si llegaste bien?").assertIsDisplayed()
        composeRule
            .onNodeWithText("Invita a esa persona de confianza. Le llega hoy un mensaje como este:")
            .assertIsDisplayed()

        // The bubble may live below the fold of the smallest font scale: the
        // test scrolls to it the way a thumb would. The ritual's name is
        // already written, so the bubble carries it (mirror, not placeholder).
        composeRule.onNode(hasScrollAction()).performScrollToNode(hasText(BUBBLE_INVITER, substring = true))
        composeRule
            .onNodeWithText(BUBBLE_INVITER, substring = true)
            .assertIsDisplayed()

        composeRule.onNodeWithText("Enviar invitación por WhatsApp").assertIsDisplayed()
        composeRule.onNodeWithText("Lo haré después").assertIsDisplayed()
        composeRule.onNodeWithText(DISCLAIMER).assertIsDisplayed()
    }

    @Test
    fun writingTheNamePersonalizesTheCta() {
        setContent(emptyState.copy(contactName = "María"))

        composeRule.onNodeWithText("Avisar a María por WhatsApp").assertIsDisplayed()
    }

    @Test
    fun theCtaStaysDisabledUntilTheNameExists() {
        setContent(emptyState)

        composeRule.onNodeWithText("Enviar invitación por WhatsApp").assertIsNotEnabled()
    }

    @Test
    fun aPhoneWithoutCountryAsksToCompleteItInline() {
        setContent(
            emptyState.copy(
                contactName = "María",
                contactPhone = "55 9876 5432",
                contactError = ContactError.PHONE,
            ),
        )

        composeRule
            .onNodeWithText("El teléfono debe incluir el país, por ejemplo +52 55 1234 5678.")
            .assertIsDisplayed()
    }

    @Test
    fun sendingShowsTheBusyCopyAndDisablesTheCta() {
        setContent(
            emptyState.copy(
                contactName = "María",
                contactPhone = "+521559876543",
                busy = OnboardingBusy.SENDING_INVITE,
            ),
        )

        composeRule.onNodeWithText("Enviando tu invitación…").assertIsDisplayed()
        composeRule.onNodeWithText("Avisar a María por WhatsApp").assertIsNotEnabled()
    }

    @Test
    fun anOfflineSendKeepsTheDraftAndOffersTheManualFallback() {
        setContent(
            emptyState.copy(
                contactName = "María",
                contactPhone = "+521559876543",
                contactError = ContactError.OFFLINE,
            ),
        )

        composeRule
            .onNodeWithText("No se envió. Tu invitación queda lista — reintenta cuando tengas internet.")
            .assertIsDisplayed()
        composeRule.onNodeWithText("Compartir la invitación yo mismo").assertIsDisplayed()
        composeRule.onNodeWithText("María").assertIsDisplayed()
    }

    @Test
    fun theConfirmationSpeaksToTheOtherAndKeepsTheDoorOpen() {
        setContent(
            emptyState.copy(contactName = "María", contactSent = true),
        )

        composeRule.onNodeWithText("Listo. María ya sabe que desde hoy cuentas con ella.").assertIsDisplayed()
        composeRule.onNodeWithText("Invitar a otra persona").assertIsDisplayed()
        composeRule.onNodeWithText("Ir al inicio").assertIsDisplayed()
    }

    @Test
    fun theLimitResolvesWithCopyNotWithAPaywall() {
        setContent(emptyState.copy(limitReached = true))

        composeRule
            .onNodeWithText("Por ahora tu plan guarda 1 contacto. Añade más cuando quieras desde Tu círculo.")
            .assertIsDisplayed()
        composeRule.onNodeWithText("Ir al inicio").assertIsDisplayed()
    }

    @Test
    fun theSkipIsHonestAndCompletesTheFlow() {
        setContent(emptyState)

        composeRule.onNodeWithText("Lo haré después").performClick()

        assertEquals(1, finishCalls)
        assertEquals(0, sendCalls)
    }

    @Test
    fun invitingAnotherResetsWithoutLeaving() {
        setContent(emptyState.copy(contactName = "María", contactSent = true))

        composeRule.onNodeWithText("Invitar a otra persona").performClick()

        assertEquals(1, inviteAnotherCalls)
        assertEquals(0, finishCalls)
    }

    private companion object {
        private const val BUBBLE_INVITER = "Ana te ha añadido"
        private const val DISCLAIMER = "No es un servicio de emergencia y no sustituye al 911/112."
    }
}
