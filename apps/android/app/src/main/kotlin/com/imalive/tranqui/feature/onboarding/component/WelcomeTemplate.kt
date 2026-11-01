package com.imalive.tranqui.feature.onboarding.component

import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import com.imalive.tranqui.R
import java.util.Locale

// The parts of the welcome mirror that live in resources, bundled so the
// pure interpolation below stays honest without dragging Context around.
data class WelcomeTemplate(
    // The exact string the server renders (see welcomePreview).
    val text: String,
    // Neutral placeholders of the empty preview ("[tu nombre]"); the real
    // send never carries blank names (the API requires 1..80 characters).
    val inviterFallback: String,
    val contactFallback: String,
)

// The mirror of the server's welcome (R-03): [template] is the exact string
// welcomeMessageV1 renders in backend/internal/circle/http.go, interpolated
// the same way — greeting (hardcoded "Hola" in the server too), inviter and
// app. Three guards pin it: the copy-guard test pins the resource, this pure
// function is unit-tested against the server-rendered sentence, and the
// composable test asserts the bubble. Follow-up: an additive endpoint that
// hands the app the template and deletes the mirror.
internal fun welcomePreview(
    template: WelcomeTemplate,
    inviterName: String,
    contactName: String,
    appName: String,
): String {
    val inviter = inviterName.ifBlank { template.inviterFallback }
    val contact = contactName.ifBlank { template.contactFallback }
    return String.format(Locale.ROOT, template.text, "Hola $contact", inviter, appName)
}

// Resolves the mirrored template from resources; screens that need the same
// text (the bubble, the manual share fallback) build it here.
@Composable
internal fun welcomeTemplate(): WelcomeTemplate =
    WelcomeTemplate(
        text = stringResource(R.string.onboarding_contact_welcome_message),
        inviterFallback = stringResource(R.string.onboarding_contact_preview_you),
        contactFallback = stringResource(R.string.onboarding_contact_preview_them),
    )
