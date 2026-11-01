package com.imalive.tranqui.feature.onboarding

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.imalive.tranqui.R
import com.imalive.tranqui.core.designsystem.HeroButton

// Paso 1 (brief, "la promesa"): the emotional promise, not features. One
// protagonist CTA anchored at the bottom; the honest disclaimer speaks in a
// low voice under it — the first true thing the product says about itself
// (voz baja, nunca un modal).
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun WelcomeScreen(
    onStartClicked: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val appName = stringResource(R.string.app_name)
    Column(
        modifier =
            modifier
                .fillMaxSize()
                .padding(horizontal = 24.dp),
        verticalArrangement = Arrangement.SpaceBetween,
    ) {
        Column {
            Spacer(Modifier.height(96.dp))
            Text(
                text = stringResource(R.string.onboarding_welcome_title),
                style = MaterialTheme.typography.displaySmall,
            )
            Text(
                text = stringResource(R.string.onboarding_welcome_sub),
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 16.dp),
            )
        }
        Column(Modifier.fillMaxWidth()) {
            HeroButton(
                text = stringResource(R.string.onboarding_welcome_cta),
                onClick = onStartClicked,
            )
            Text(
                text = stringResource(R.string.onboarding_welcome_disclaimer, appName),
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = 16.dp, bottom = 24.dp),
            )
        }
    }
}
