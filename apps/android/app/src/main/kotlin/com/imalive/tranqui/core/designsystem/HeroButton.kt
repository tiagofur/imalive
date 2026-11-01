package com.imalive.tranqui.core.designsystem

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

// The hero CTA — the OkButton DNA (design-system.md): one protagonist action
// per screen, pill radius, 56 dp tall (targets ≥ 48 dp). Home and onboarding
// share it.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun HeroButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    Button(
        onClick = onClick,
        modifier =
            modifier
                .fillMaxWidth()
                .height(56.dp),
        shape = RoundedCornerShape(percent = 50),
        enabled = enabled,
    ) {
        Text(text = text, style = MaterialTheme.typography.titleMedium)
    }
}
