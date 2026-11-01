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
