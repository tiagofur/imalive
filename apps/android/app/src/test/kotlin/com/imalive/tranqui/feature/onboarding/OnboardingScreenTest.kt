package com.imalive.tranqui.feature.onboarding

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.performTextReplacement
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.imalive.tranqui.core.designsystem.TranquiTheme
import com.imalive.tranqui.core.network.ApiError
import com.imalive.tranqui.core.network.ApiErrorCode
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// The critical UI path of issue #75 slice 3: the whole auth flow walks
// welcome → phone → code through the real step machine, and an invalid code
// recovers in place. The verify success itself lands on home through the
// reactive gate, covered by HomeViewModelTest (slice 2).
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class OnboardingScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val mainDispatcher = UnconfinedTestDispatcher()

    @Before
    fun setUp() {
        Dispatchers.setMain(mainDispatcher)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private fun setContent(auth: FakeAuth): OnboardingViewModel {
        val viewModel = OnboardingViewModel(auth)
        composeRule.setContent {
            TranquiTheme {
                OnboardingScreen(viewModel = viewModel)
            }
        }
        return viewModel
    }

    private fun walkToCodePhase(auth: FakeAuth) {
        setContent(auth)
        composeRule.onNodeWithText("Empezar").performClick()
        composeRule.onNodeWithText("¿A qué teléfono te enviamos el código?").assertIsDisplayed()
        composeRule.onNode(hasSetTextAction()).performTextInput("+5215512345678")
        composeRule.onNodeWithText("Enviarme el código").performClick()
        composeRule.onNodeWithText("Escribe el código de 6 dígitos").assertIsDisplayed()
    }

    @Test
    fun theAuthFlowWalksWelcomePhoneAndCodeWithVisibleProgress() {
        val auth = FakeAuth()
        setContent(auth)

        composeRule.onNodeWithContentDescription("Paso 1 de 4").assertIsDisplayed()
        composeRule.onNodeWithText("Empezar").performClick()
        composeRule.onNodeWithContentDescription("Paso 2 de 4").assertIsDisplayed()

        composeRule.onNode(hasSetTextAction()).performTextInput("+5215512345678")
        composeRule.onNodeWithText("Enviarme el código").performClick()

        composeRule.onNodeWithText("Escribe el código de 6 dígitos").assertIsDisplayed()
        composeRule.onNodeWithText("Te lo enviamos al +5215512345678.").assertIsDisplayed()
        assertEquals(1, auth.requestCalls)
    }

    @Test
    fun anInvalidCodeRecoversInPlaceWithoutLosingTheDraft() {
        val auth = FakeAuth()
        auth.verifyError = ApiError(ApiErrorCode.OTP_INVALID, httpStatus = 400)
        walkToCodePhase(auth)

        composeRule.onNode(hasSetTextAction()).performTextInput("111111")
        composeRule.onNodeWithText("Ese código no es. Revísalo.").assertIsDisplayed()

        auth.verifyError = null
        composeRule.onNode(hasSetTextAction()).performTextReplacement("482913")

        assertEquals(2, auth.verifyCalls)
        assertEquals("482913", auth.lastVerifyCode)
    }

    @Test
    fun theSixthDigitAutoVerifiesThroughTheRealField() {
        val auth = FakeAuth()
        walkToCodePhase(auth)

        composeRule.onNode(hasSetTextAction()).performTextInput("482913")

        assertEquals(1, auth.verifyCalls)
        assertEquals("+5215512345678", auth.lastVerifyPhone)
    }
}
