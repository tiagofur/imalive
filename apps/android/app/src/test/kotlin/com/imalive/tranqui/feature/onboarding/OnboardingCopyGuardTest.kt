package com.imalive.tranqui.feature.onboarding

import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import com.imalive.tranqui.R
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// Criterion of issue #75 (brief §5, "la promesa honesta"): the literal
// wording is "no es un servicio de emergencia y no sustituye al 911/112" —
// "emergencia médica" and safety promises are forbidden everywhere. This
// guard scans every string resource so a future copy cannot regress it.
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
}
