package com.imalive.tranqui.feature.onboarding

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.imalive.tranqui.core.designsystem.TranquiTheme
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// The brief's critical states of paso 2, on the two phases of the OTP
// screen: the copy of every contract error and the rate-limit countdown
// disabling its action. Composed against hand-built states, as HomeScreen
// tests do.
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class OtpScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private fun setContent(state: OnboardingUiState) {
        composeRule.setContent {
            TranquiTheme {
                if (state.step == OnboardingStep.PHONE) {
                    OtpRequestScreen(
                        uiState = state,
                        onPhoneChanged = {},
                        onRequestCodeClicked = {},
                    )
                } else {
                    OtpVerifyScreen(
                        uiState = state,
                        onCodeChanged = {},
                        onVerifyClicked = {},
                        onResendClicked = {},
                    )
                }
            }
        }
    }

    @Test
    fun thePhonePhaseAsksTheBriefQuestionWithItsCta() {
        setContent(OnboardingUiState(step = OnboardingStep.PHONE))

        composeRule.onNodeWithText("¿A qué teléfono te enviamos el código?").assertIsDisplayed()
        composeRule
            .onNodeWithText("Solo para entrar. Tu número no se muestra a nadie.")
            .assertIsDisplayed()
        composeRule.onNodeWithText("Enviarme el código").assertIsDisplayed()
    }

    @Test
    fun phoneInvalidAsksToReviewTheNumberInline() {
        setContent(
            OnboardingUiState(
                step = OnboardingStep.PHONE,
                phone = "+52 55 12",
                phoneError = PhoneError.INVALID,
            ),
        )

        composeRule.onNodeWithText("Ese teléfono no parece completo. Revísalo.").assertIsDisplayed()
    }

    @Test
    fun theRateLimitShowsTheRetryAfterCountdownAndDisablesTheCta() {
        setContent(
            OnboardingUiState(
                step = OnboardingStep.PHONE,
                phone = "+5215512345678",
                rateLimitSeconds = 47,
            ),
        )

        composeRule
            .onNodeWithText("Demasiados intentos. Prueba otra vez en 47 s.")
            .assertIsDisplayed()
        composeRule.onNodeWithText("Enviarme el código").assertIsNotEnabled()
    }

    @Test
    fun theCodePhaseNamesThePhoneThatGotTheCode() {
        setContent(
            OnboardingUiState(step = OnboardingStep.CODE, phone = "+5215512345678"),
        )

        composeRule.onNodeWithText("Escribe el código de 6 dígitos").assertIsDisplayed()
        composeRule.onNodeWithText("Te lo enviamos al +5215512345678.").assertIsDisplayed()
        composeRule.onNodeWithText("Enviarlo de nuevo").assertIsDisplayed()
        composeRule.onNodeWithText("Entrar").assertIsDisplayed()
    }

    @Test
    fun otpInvalidShowsTheBriefCopy() {
        setContent(
            OnboardingUiState(
                step = OnboardingStep.CODE,
                phone = "+5215512345678",
                code = "111111",
                codeError = CodeError.INVALID,
            ),
        )

        composeRule.onNodeWithText("Ese código no es. Revísalo.").assertIsDisplayed()
        composeRule.onNodeWithText("1").assertIsDisplayed() // kept for rewrite
    }

    @Test
    fun otpExpiredTellsTheFreshCodeIsOut() {
        setContent(
            OnboardingUiState(
                step = OnboardingStep.CODE,
                phone = "+5215512345678",
                codeError = CodeError.EXPIRED,
            ),
        )

        composeRule.onNodeWithText("El código caducó. Te enviamos uno nuevo.").assertIsDisplayed()
    }

    @Test
    fun verifyingShowsTheSpinnerCopyAndDisablesBothActions() {
        setContent(
            OnboardingUiState(
                step = OnboardingStep.CODE,
                phone = "+5215512345678",
                code = "482913",
                busy = OnboardingBusy.VERIFYING,
            ),
        )

        composeRule.onNodeWithText("Verificando…").assertIsDisplayed()
        composeRule.onNodeWithText("Entrar").assertIsNotEnabled()
        composeRule.onNodeWithText("Enviarlo de nuevo").assertDoesNotExist()
    }

    @Test
    fun theCountdownDisablesTheResend() {
        setContent(
            OnboardingUiState(
                step = OnboardingStep.CODE,
                phone = "+5215512345678",
                rateLimitSeconds = 12,
            ),
        )

        composeRule.onNodeWithText("Demasiados intentos. Prueba otra vez en 12 s.").assertIsDisplayed()
        composeRule.onNodeWithText("Enviarlo de nuevo").assertIsNotEnabled()
    }

    @Test
    fun anOfflineCodeErrorKeepsTheHumanCopy() {
        setContent(
            OnboardingUiState(
                step = OnboardingStep.CODE,
                phone = "+5215512345678",
                code = "482913",
                codeError = CodeError.OFFLINE,
            ),
        )

        composeRule
            .onNodeWithText("No tienes internet. Revisa tu conexión y prueba otra vez.")
            .assertIsDisplayed()
    }
}
