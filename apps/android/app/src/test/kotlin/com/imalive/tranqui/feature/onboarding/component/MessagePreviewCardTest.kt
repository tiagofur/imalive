package com.imalive.tranqui.feature.onboarding.component

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.imalive.tranqui.core.designsystem.TranquiTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// The memorable moment (brief paso 4, issue #75 slice 4) — the bubble is the
// product's honesty test: it must show the exact message the backend sends,
// interpolated with the names as they are typed. Three checks: the pure
// interpolation equals the server-rendered sentence, the composed bubble
// shows it, and the empty state uses the neutral placeholders.
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class MessagePreviewCardTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun thePreviewInterpolatesExactlyLikeTheServerDoes() {
        val template =
            WelcomeTemplate(
                text = MIRRORED_TEMPLATE,
                inviterFallback = "[tu nombre]",
                contactFallback = "[su nombre]",
            )

        val rendered = welcomePreview(template, inviterName = "Ana", contactName = "María", appName = "Tranqui")

        assertEquals(MIRRORED_SENTENCE, rendered)
    }

    @Test
    fun theBubbleMirrorsTheExactMessageTheContactWillReceive() {
        composeRule.setContent {
            TranquiTheme {
                MessagePreviewCard(inviterName = "Ana", contactName = "María")
            }
        }

        composeRule.onNodeWithText(MIRRORED_SENTENCE).assertIsDisplayed()
    }

    @Test
    fun blankNamesRenderTheNeutralPlaceholdersOfTheEmptyState() {
        composeRule.setContent {
            TranquiTheme {
                MessagePreviewCard(inviterName = "", contactName = "")
            }
        }

        composeRule.onNodeWithText(MIRRORED_PLACEHOLDERS).assertIsDisplayed()
    }

    private companion object {
        // welcomeMessageV1 (backend/internal/circle/http.go) rendered for
        // inviter "Ana", contact "María" and the app name.
        private const val MIRRORED_SENTENCE =
            "Hola María: Ana te ha añadido como persona de confianza en Tranqui, " +
                "la app con la que confirma a diario que está bien. No tienes que hacer nada. " +
                "Solo te escribiremos si Ana pasa muchas horas sin confirmar, para que puedas " +
                "comprobar cómo está. Tranqui es un servicio de avisos entre personas; " +
                "no es un servicio de emergencias."

        private const val MIRRORED_PLACEHOLDERS =
            "Hola [su nombre]: [tu nombre] te ha añadido como persona de confianza en Tranqui, " +
                "la app con la que confirma a diario que está bien. No tienes que hacer nada. " +
                "Solo te escribiremos si [tu nombre] pasa muchas horas sin confirmar, para que " +
                "puedas comprobar cómo está. Tranqui es un servicio de avisos entre personas; " +
                "no es un servicio de emergencias."

        // The same sentence shape with markers, as the resource stores it.
        private const val MIRRORED_TEMPLATE =
            "%1\$s: %2\$s te ha añadido como persona de confianza en %3\$s, " +
                "la app con la que confirma a diario que está bien. No tienes que hacer nada. " +
                "Solo te escribiremos si %2\$s pasa muchas horas sin confirmar, para que puedas " +
                "comprobar cómo está. %3\$s es un servicio de avisos entre personas; " +
                "no es un servicio de emergencias."
    }
}
