package com.imalive.tranqui.feature.checkin

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

// R-04 (home = the button) + R-12 (the streak the user is protecting): one
// tap for "Estoy OK", the next-checkin status and the streak count — frozen
// while the pause mode is live (docs/diseno/ux-movil.md).
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun HomeScreen(
    uiState: HomeUiState,
    onCheckInClicked: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier.fillMaxSize(),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        if (uiState.streakDays > 0) {
            Text(
                text = stringResource(R.string.home_streak_days, uiState.streakDays),
                style = MaterialTheme.typography.headlineSmall,
            )
            if (uiState.streakFrozen) {
                Text(
                    text = stringResource(R.string.home_streak_frozen),
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.padding(top = 4.dp),
                )
            }
        }
        Text(
            text = stringResource(R.string.home_next_checkin, uiState.nextCheckinLabel),
            style = MaterialTheme.typography.bodyLarge,
            modifier = Modifier.padding(top = 16.dp),
        )
        Button(
            onClick = onCheckInClicked,
            modifier = Modifier.padding(top = 24.dp),
        ) {
            Text(text = stringResource(R.string.checkin_button_ok))
        }
    }
}
