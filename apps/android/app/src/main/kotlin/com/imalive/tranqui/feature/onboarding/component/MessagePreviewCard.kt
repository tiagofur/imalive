package com.imalive.tranqui.feature.onboarding.component

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.imalive.tranqui.R
import java.util.Locale

// The parts of the welcome mirror that live in resources, bundled so the
// pure interpolation below stays honest without dragging Context around.
internal data class WelcomeTemplate(
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

// The memorable moment (brief paso 4): a chat bubble showing, with the real
// names the user is typing, the exact message the trusted person will
// receive. Mirror, not promise — honesty is the feature.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun MessagePreviewCard(
    inviterName: String,
    contactName: String,
    modifier: Modifier = Modifier,
) {
    val message =
        welcomePreview(
            welcomeTemplate(),
            inviterName,
            contactName,
            stringResource(R.string.app_name),
        )
    // A received bubble: the small corner opens toward the sender.
    Surface(
        shape =
            RoundedCornerShape(
                topStart = 4.dp,
                topEnd = 16.dp,
                bottomStart = 16.dp,
                bottomEnd = 16.dp,
            ),
        color = MaterialTheme.colorScheme.surface,
        modifier = modifier.fillMaxWidth(),
    ) {
        Text(
            text = message,
            style = MaterialTheme.typography.bodyLarge,
            modifier = Modifier.padding(16.dp),
        )
    }
}
