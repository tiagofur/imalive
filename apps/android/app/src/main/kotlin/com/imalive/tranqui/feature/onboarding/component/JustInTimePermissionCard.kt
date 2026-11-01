package com.imalive.tranqui.feature.onboarding.component

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.imalive.tranqui.R
import com.imalive.tranqui.feature.onboarding.hhmm
import java.time.LocalTime

// The just-in-time notification pre-permission (brief paso 3, "nunca en
// frío"): it speaks only after the hour exists, names the exact slot and
// promises calm. The system prompt fires only through [onAllow]; "Ahora no"
// continues the flow without it — denying never blocks (ux-movil.md).
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun JustInTimePermissionCard(
    checkinTime: LocalTime,
    onAllow: () -> Unit,
    onDecline: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Surface(
        shape = RoundedCornerShape(16.dp),
        color = MaterialTheme.colorScheme.surface,
        modifier = modifier.fillMaxWidth(),
    ) {
        Column(Modifier.padding(16.dp)) {
            Text(
                text = stringResource(R.string.onboarding_permission_title, checkinTime.hhmm()),
                style = MaterialTheme.typography.bodyMedium,
            )
            Row(Modifier.padding(top = 4.dp)) {
                TextButton(onClick = onAllow) {
                    Text(stringResource(R.string.onboarding_permission_allow))
                }
                TextButton(onClick = onDecline) {
                    Text(stringResource(R.string.onboarding_permission_decline))
                }
            }
        }
    }
}
