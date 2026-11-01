package com.imalive.tranqui.feature.onboarding.component

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import com.imalive.tranqui.R

// The ritual progress of the brief ("línea fina de 4 segmentos arriba"):
// not bureaucracy — no big numbers, the accessible name carries the count
// ("Paso N de 4"). Bienvenida is segment 1; the two OTP phases share
// segment 2.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun OnboardingProgress(
    currentStep: Int,
    modifier: Modifier = Modifier,
) {
    val label = stringResource(R.string.onboarding_progress_a11y, currentStep)
    Row(
        modifier =
            modifier
                .fillMaxWidth()
                .padding(horizontal = 24.dp, vertical = 16.dp)
                .semantics { contentDescription = label },
    ) {
        repeat(TOTAL_SEGMENTS) { segment ->
            Box(
                modifier =
                    Modifier
                        .weight(1f)
                        .padding(horizontal = 3.dp)
                        .height(3.dp)
                        .clip(RoundedCornerShape(2.dp))
                        .background(
                            if (segment < currentStep) {
                                MaterialTheme.colorScheme.primary
                            } else {
                                MaterialTheme.colorScheme.surfaceVariant
                            },
                        ),
            )
        }
    }
}

private const val TOTAL_SEGMENTS = 4
