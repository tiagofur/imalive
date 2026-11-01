package com.imalive.tranqui.feature.onboarding

import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.imalive.tranqui.R
import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// Criteria of issue #75 (brief §5, "la promesa honesta" + decisión abierta
// 9.1): the literal wording is "no es un servicio de emergencia y no
// sustituye al 911/112" — "emergencia médica" and safety promises are
// forbidden everywhere — and the preview bubble mirrors the backend's real
// welcome verbatim (welcomeMessageV1). These guards scan every string
// resource so a future copy cannot regress either.
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class OnboardingCopyGuardTest {
    @Test
    fun noStringResourcePromisesEmergencyCapabilities() {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val forbidden = listOf("emergencia médica", "te protege", "siempre a salvo")

        for (field in R.string::class.java.declaredFields) {
            val text = context.getString(field.getInt(null))
            for (phrase in forbidden) {
                check(!text.contains(phrase)) {
                    "R.string.${field.name} contains the forbidden phrase \"$phrase\": \"$text\""
                }
            }
        }
    }

    @Test
    fun theWelcomeMessageResourceMirrorsTheBackendTemplateVerbatim() {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val rendered =
            String.format(
                context.getString(R.string.onboarding_contact_welcome_message),
                "HOLA",
                "INVITER",
                "APP",
            )

        // The server (welcomeMessageV1 in backend/internal/circle/http.go)
        // renders: "Hola {contact}: {inviter} te ha añadido como persona de
        // confianza en {app}, …" — the mirrored string must stay 1:1.
        assertEquals(BACKEND_TEMPLATE, rendered)
    }

    private companion object {
        private const val BACKEND_TEMPLATE =
            "HOLA: INVITER te ha añadido como persona de confianza en APP, " +
                "la app con la que confirma a diario que está bien. No tienes que hacer nada. " +
                "Solo te escribiremos si INVITER pasa muchas horas sin confirmar, para que " +
                "puedas comprobar cómo está. APP es un servicio de avisos entre personas; " +
                "no es un servicio de emergencias."
    }
}
