package com.imalive.tranqui.feature.onboarding

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.imalive.tranqui.R
import com.imalive.tranqui.core.designsystem.HeroButton
import com.imalive.tranqui.feature.onboarding.component.MessagePreviewCard
import com.imalive.tranqui.feature.onboarding.component.welcomePreview
import com.imalive.tranqui.feature.onboarding.component.welcomeTemplate

// Paso 4 (brief): "añade a tu mamá" — the trusted person enters the circle
// with a name and a phone. The bubble previews the exact welcome the backend
// sends on POST /v1/contacts; the send never duplicates the WhatsApp message
// (the intent is only the manual fallback of the failure states). The
// confirmation speaks to the other, not about the user; "Lo haré después"
// skips honestly.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun AddContactScreen(
    uiState: OnboardingUiState,
    onContactNameChanged: (String) -> Unit,
    onContactPhoneChanged: (String) -> Unit,
    onSendInviteClicked: () -> Unit,
    onInviteAnotherClicked: () -> Unit,
    onFinishClicked: () -> Unit,
    modifier: Modifier = Modifier,
) {
    // The soft haptic of the confirmation — the same family as the check-in
    // (brief paso 4: the first act of the product is care, not fireworks).
    val haptics = LocalHapticFeedback.current
    LaunchedEffect(uiState.contactSent) {
        if (uiState.contactSent) haptics.performHapticFeedback(HapticFeedbackType.LongPress)
    }
    Column(modifier = modifier.fillMaxSize().padding(horizontal = 24.dp)) {
        // Scrollable content with the actions pinned: the bubble and the form
        // must survive the biggest font scale (brief, WCAG AC).
        Column(
            modifier =
                Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState()),
        ) {
            when {
                uiState.limitReached -> LimitReachedBody()
                uiState.contactSent -> ConfirmationBody(uiState.contactName, onInviteAnotherClicked)
                else -> FormBody(uiState, onContactNameChanged, onContactPhoneChanged)
            }
        }
        BottomActions(uiState, onSendInviteClicked, onFinishClicked)
    }
}

@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun FormBody(
    uiState: OnboardingUiState,
    onContactNameChanged: (String) -> Unit,
    onContactPhoneChanged: (String) -> Unit,
) {
    Column {
        Text(
            text = stringResource(R.string.onboarding_contact_title),
            style = MaterialTheme.typography.headlineSmall,
        )
        Text(
            text = stringResource(R.string.onboarding_contact_sub),
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 8.dp),
        )
        MessagePreviewCard(
            inviterName = uiState.name,
            contactName = uiState.contactName,
            modifier = Modifier.padding(top = 16.dp),
        )
        OutlinedTextField(
            value = uiState.contactName,
            onValueChange = onContactNameChanged,
            modifier = Modifier.fillMaxWidth().padding(top = 16.dp),
            label = { Text(stringResource(R.string.onboarding_contact_name_label)) },
            singleLine = true,
        )
        OutlinedTextField(
            value = uiState.contactPhone,
            onValueChange = onContactPhoneChanged,
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
            label = { Text(stringResource(R.string.onboarding_contact_phone_label)) },
            placeholder = { Text(stringResource(R.string.onboarding_contact_phone_hint)) },
            singleLine = true,
            isError = uiState.contactError == ContactError.PHONE,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
            supportingText = {
                if (uiState.contactError == ContactError.PHONE) {
                    Text(
                        text = stringResource(R.string.onboarding_error_contact_phone),
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            },
        )
    }
}

// The confirmation replaces the form (brief paso 4): "Listo. María ya sabe
// que desde hoy cuentas con ella." + the door to invite someone else.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun ConfirmationBody(
    contactName: String,
    onInviteAnotherClicked: () -> Unit,
) {
    Column {
        Text(
            text = stringResource(R.string.onboarding_contact_sent, contactName.trim()),
            style = MaterialTheme.typography.headlineSmall,
        )
        TextButton(onClick = onInviteAnotherClicked, modifier = Modifier.padding(top = 8.dp)) {
            Text(stringResource(R.string.onboarding_contact_invite_another))
        }
    }
}

// CONTACT_LIMIT_REACHED resolves with copy — never a paywall (R-13) — and a
// way home; the circle waits in "Tu círculo".
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun LimitReachedBody() {
    Text(
        text = stringResource(R.string.onboarding_contact_limit),
        style = MaterialTheme.typography.headlineSmall,
    )
}

@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun BottomActions(
    uiState: OnboardingUiState,
    onSendInviteClicked: () -> Unit,
    onFinishClicked: () -> Unit,
) {
    Column(Modifier.fillMaxWidth()) {
        if (uiState.limitReached || uiState.contactSent) {
            HeroButton(
                text = stringResource(R.string.onboarding_contact_finish),
                onClick = onFinishClicked,
            )
        } else {
            WireFeedback(uiState.contactError, uiState.name, uiState.contactName)
            HeroButton(
                text = ctaLabel(uiState.contactName),
                onClick = onSendInviteClicked,
                // No name, no send: the disabled CTA points the eye to the
                // empty "Su nombre" field (the VM keeps the same guard).
                enabled = uiState.busy == null && uiState.contactName.isNotBlank(),
            )
            if (uiState.busy == OnboardingBusy.SENDING_INVITE) {
                BusyRow()
            } else {
                TextButton(
                    onClick = onFinishClicked,
                    modifier = Modifier.align(Alignment.CenterHorizontally),
                ) {
                    Text(stringResource(R.string.onboarding_contact_skip))
                }
            }
        }
        Text(
            text = stringResource(R.string.onboarding_contact_disclaimer),
            style = MaterialTheme.typography.labelLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 12.dp, bottom = 24.dp),
        )
    }
}

// The wire failures reuse the human copy; the offline state additionally
// offers the manual WhatsApp fallback with the exact same mirrored text.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun WireFeedback(
    error: ContactError?,
    inviterName: String,
    contactName: String,
) {
    val context = LocalContext.current
    if (error == null || error == ContactError.PHONE) return
    Text(
        text =
            stringResource(
                if (error == ContactError.OFFLINE) {
                    R.string.onboarding_error_contact_offline
                } else {
                    R.string.onboarding_error_retry
                },
            ),
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.error,
        modifier = Modifier.padding(bottom = 8.dp),
    )
    if (error == ContactError.OFFLINE) {
        val message =
            welcomePreview(welcomeTemplate(), inviterName, contactName, stringResource(R.string.app_name))
        TextButton(onClick = { WhatsAppInvitation.share(context, message) }) {
            Text(stringResource(R.string.onboarding_contact_share_fallback))
        }
    }
}

@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun BusyRow() {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.padding(top = 12.dp),
    ) {
        CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
        Text(
            text = stringResource(R.string.onboarding_busy_sending_invite),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(start = 8.dp),
        )
    }
}

// The brief's personalized CTA: the mamá's name when it exists, the honest
// generic otherwise.
@Composable
private fun ctaLabel(contactName: String): String =
    if (contactName.isBlank()) {
        stringResource(R.string.onboarding_contact_cta)
    } else {
        stringResource(R.string.onboarding_contact_cta_named, contactName.trim())
    }
