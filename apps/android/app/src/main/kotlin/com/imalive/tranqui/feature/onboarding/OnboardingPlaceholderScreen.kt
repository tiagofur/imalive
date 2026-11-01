package com.imalive.tranqui.feature.onboarding

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.imalive.tranqui.R

// Entry point of the onboarding while the session gate has no session
// (issue #75 slice 2). Slice 3 replaces this placeholder with the real
// welcome step; the strings are already the brief's copy (ux-brief, paso 1),
// so only the body of the flow changes under the same gate.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun OnboardingPlaceholderScreen(
    onStartClicked: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier.fillMaxSize(),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(R.string.onboarding_welcome_title),
            style = MaterialTheme.typography.headlineSmall,
        )
        Button(
            onClick = onStartClicked,
            modifier = Modifier.padding(top = 24.dp),
        ) {
            Text(text = stringResource(R.string.onboarding_welcome_cta))
        }
    }
}
