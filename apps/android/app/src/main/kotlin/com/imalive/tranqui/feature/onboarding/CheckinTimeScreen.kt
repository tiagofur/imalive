package com.imalive.tranqui.feature.onboarding

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TimePicker
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import com.imalive.tranqui.R
import com.imalive.tranqui.core.designsystem.HeroButton
import com.imalive.tranqui.feature.onboarding.component.JustInTimePermissionCard
import java.time.LocalTime

// Paso 3 (brief): the ritual — the user's name (the welcome message will
// carry it) and the daily hour, with the Material 3 time picker proposing
// 21:00. The timezone is the device's, never edited. The just-in-time
// permission card speaks between the hour and the CTA; continuing PATCHes
// /v1/me and walks into the contact moment.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun CheckinTimeScreen(
    uiState: OnboardingUiState,
    onNameChanged: (String) -> Unit,
    onCheckinTimeChanged: (LocalTime) -> Unit,
    onContinueClicked: () -> Unit,
    onAllowNotifications: () -> Unit,
    onPermissionAnswered: () -> Unit,
    showPermissionCard: Boolean,
    modifier: Modifier = Modifier,
) {
    var showTimePicker by rememberSaveable { mutableStateOf(false) }
    Column(modifier = modifier.fillMaxSize().padding(horizontal = 24.dp)) {
        // Scrollable content with the CTA pinned: the copy must survive the
        // biggest font scale without breaking the layout (brief, WCAG AC).
        Column(
            modifier =
                Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState()),
        ) {
            Text(
                text = stringResource(R.string.onboarding_ritual_title),
                style = MaterialTheme.typography.headlineSmall,
            )
            NameField(
                name = uiState.name,
                error = uiState.ritualError,
                onNameChanged = onNameChanged,
            )
            TimeSection(
                checkinTime = uiState.checkinTime,
                onTimeClick = { showTimePicker = true },
            )
            if (showPermissionCard) {
                JustInTimePermissionCard(
                    checkinTime = uiState.checkinTime,
                    onAllow = onAllowNotifications,
                    onDecline = onPermissionAnswered,
                    modifier = Modifier.padding(top = 12.dp),
                )
            } else if (uiState.permissionAnswered) {
                Text(
                    text = stringResource(R.string.onboarding_permission_answered),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 12.dp),
                )
            }
        }
        RitualActions(
            busy = uiState.busy,
            ritualError = uiState.ritualError,
            onContinueClicked = onContinueClicked,
        )
    }
    if (showTimePicker) {
        RitualTimePicker(
            initial = uiState.checkinTime,
            onConfirm = { time ->
                onCheckinTimeChanged(time)
                showTimePicker = false
            },
            onDismiss = { showTimePicker = false },
        )
    }
}

@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun NameField(
    name: String,
    error: RitualError?,
    onNameChanged: (String) -> Unit,
) {
    OutlinedTextField(
        value = name,
        onValueChange = onNameChanged,
        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
        label = { Text(stringResource(R.string.onboarding_ritual_name_label)) },
        singleLine = true,
        isError = error == RitualError.EMPTY_NAME,
        supportingText = {
            if (error == RitualError.EMPTY_NAME) {
                Text(
                    text = stringResource(R.string.onboarding_ritual_name_error),
                    color = MaterialTheme.colorScheme.error,
                )
            }
        },
    )
}

@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun TimeSection(
    checkinTime: LocalTime,
    onTimeClick: () -> Unit,
) {
    val slot = checkinTime.hhmm()
    val timeRowA11y = stringResource(R.string.onboarding_ritual_time_a11y, slot)
    Text(
        text = stringResource(R.string.onboarding_ritual_time_title),
        style = MaterialTheme.typography.titleMedium,
        modifier = Modifier.padding(top = 24.dp),
    )
    Text(
        text = stringResource(R.string.onboarding_ritual_time_sub),
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(top = 4.dp),
    )
    Surface(
        onClick = onTimeClick,
        shape = RoundedCornerShape(16.dp),
        color = MaterialTheme.colorScheme.surface,
        modifier =
            Modifier
                .fillMaxWidth()
                .padding(top = 12.dp)
                .semantics { contentDescription = timeRowA11y },
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(16.dp),
        ) {
            Column(Modifier.weight(1f)) {
                Text(
                    text = stringResource(R.string.onboarding_ritual_time_label),
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Text(text = slot, style = MaterialTheme.typography.titleMedium)
            }
            Text(
                text = stringResource(R.string.onboarding_ritual_time_change),
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.primary,
            )
        }
    }
}

@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun RitualActions(
    busy: OnboardingBusy?,
    ritualError: RitualError?,
    onContinueClicked: () -> Unit,
) {
    Column(Modifier.fillMaxWidth()) {
        if (ritualError == RitualError.OFFLINE || ritualError == RitualError.RETRY) {
            Text(
                text = stringResource(ritualErrorCopy(ritualError)),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.error,
                modifier = Modifier.padding(bottom = 8.dp),
            )
        }
        HeroButton(
            text = stringResource(R.string.onboarding_ritual_cta),
            onClick = onContinueClicked,
            enabled = busy == null,
        )
        if (busy == OnboardingBusy.SAVING_RITUAL) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.padding(top = 12.dp),
            ) {
                CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                Text(
                    text = stringResource(R.string.onboarding_busy_saving),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(start = 8.dp),
                )
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

// The Material 3 picker (brief, Android note) in the locale's 24h shape the
// ritual promised; the slot travels as a LocalTime into the PATCH.
@Composable
@OptIn(ExperimentalMaterial3Api::class) // TimePicker is still experimental in this BOM
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun RitualTimePicker(
    initial: LocalTime,
    onConfirm: (LocalTime) -> Unit,
    onDismiss: () -> Unit,
) {
    val state =
        rememberTimePickerState(
            initialHour = initial.hour,
            initialMinute = initial.minute,
            is24Hour = true,
        )
    AlertDialog(
        onDismissRequest = onDismiss,
        confirmButton = {
            TextButton(onClick = { onConfirm(LocalTime.of(state.hour, state.minute)) }) {
                Text(stringResource(R.string.onboarding_ritual_time_ok))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.onboarding_ritual_time_cancel))
            }
        },
        text = { TimePicker(state) },
    )
}

// The wire failures reuse the paso-2 human copy; the empty name speaks
// inline, under its own field.
private fun ritualErrorCopy(error: RitualError?): Int =
    when (error) {
        RitualError.OFFLINE -> R.string.onboarding_error_offline
        else -> R.string.onboarding_error_retry
    }
