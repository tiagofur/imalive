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

// R-04 (home = the button): one tap for "Estoy OK" plus the next-checkin
// status. F0 scaffold: static schedule, no navigation (docs/diseno/ux-movil.md).
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
        Text(
            text = stringResource(R.string.home_next_checkin, uiState.nextCheckinLabel),
            style = MaterialTheme.typography.bodyLarge,
        )
        Button(
            onClick = onCheckInClicked,
            modifier = Modifier.padding(top = 24.dp),
        ) {
            Text(text = stringResource(R.string.checkin_button_ok))
        }
    }
}
