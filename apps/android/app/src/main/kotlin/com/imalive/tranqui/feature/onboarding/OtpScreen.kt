package com.imalive.tranqui.feature.onboarding

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.imalive.tranqui.R
import com.imalive.tranqui.core.designsystem.HeroButton
import com.imalive.tranqui.feature.onboarding.component.OtpInput

// Paso 2 (brief): "tu teléfono → tu código", una pantalla con dos fases —
// una por composable, montadas por el paso de la máquina. La única acción
// protagonista de cada fase vive anclada abajo; carga y errores usan el copy
// del brief conmutando sobre Problem.code — nunca el detail crudo.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun OtpRequestScreen(
    uiState: OnboardingUiState,
    onPhoneChanged: (String) -> Unit,
    onRequestCodeClicked: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier.fillMaxSize().padding(horizontal = 24.dp),
        verticalArrangement = Arrangement.SpaceBetween,
    ) {
        Column {
            Text(
                text = stringResource(R.string.onboarding_phone_title),
                style = MaterialTheme.typography.headlineSmall,
            )
            Text(
                text = stringResource(R.string.onboarding_phone_sub),
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 8.dp),
            )
            OutlinedTextField(
                value = uiState.phone,
                onValueChange = onPhoneChanged,
                modifier = Modifier.fillMaxWidth().padding(top = 24.dp),
                label = { Text(stringResource(R.string.onboarding_phone_field_label)) },
                placeholder = { Text(stringResource(R.string.onboarding_phone_field_hint)) },
                singleLine = true,
                isError = uiState.phoneError != null,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
                supportingText = {
                    phoneErrorCopy(uiState.phoneError)?.let { res ->
                        Text(
                            text = stringResource(res),
                            color = MaterialTheme.colorScheme.error,
                        )
                    }
                },
            )
        }
        Column(Modifier.fillMaxWidth()) {
            if (uiState.rateLimitSeconds > 0) {
                RateLimitCopy(uiState.rateLimitSeconds)
            }
            HeroButton(
                text = stringResource(R.string.onboarding_phone_cta),
                onClick = onRequestCodeClicked,
                enabled = uiState.busy == null && uiState.rateLimitSeconds == 0,
            )
            if (uiState.busy == OnboardingBusy.SENDING_CODE) {
                BusyCopy(R.string.onboarding_busy_sending_code)
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun OtpVerifyScreen(
    uiState: OnboardingUiState,
    onCodeChanged: (String) -> Unit,
    onVerifyClicked: () -> Unit,
    onResendClicked: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier.fillMaxSize().padding(horizontal = 24.dp),
        verticalArrangement = Arrangement.SpaceBetween,
    ) {
        Column {
            Text(
                text = stringResource(R.string.onboarding_code_title),
                style = MaterialTheme.typography.headlineSmall,
            )
            Text(
                text = stringResource(R.string.onboarding_code_sub, uiState.phone),
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 8.dp),
            )
            OtpInput(
                value = uiState.code,
                onValueChange = onCodeChanged,
                isError = uiState.codeError != null,
                modifier = Modifier.fillMaxWidth().padding(top = 24.dp),
            )
            codeErrorCopy(uiState.codeError)?.let { res ->
                Text(
                    text = stringResource(res),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(top = 12.dp),
                )
            }
        }
        CodeActions(
            busy = uiState.busy,
            codeReady = uiState.code.length == CODE_LENGTH,
            rateLimitSeconds = uiState.rateLimitSeconds,
            onVerifyClicked = onVerifyClicked,
            onResendClicked = onResendClicked,
        )
    }
}

// The bottom block of the verify phase: the backup "Entrar" (the
// auto-verify already ran at the sixth digit; this is the manual path),
// the busy copy or the resend with its live cooldown.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun CodeActions(
    busy: OnboardingBusy?,
    codeReady: Boolean,
    rateLimitSeconds: Int,
    onVerifyClicked: () -> Unit,
    onResendClicked: () -> Unit,
) {
    Column(Modifier.fillMaxWidth()) {
        if (rateLimitSeconds > 0) {
            RateLimitCopy(rateLimitSeconds)
        }
        HeroButton(
            text = stringResource(R.string.onboarding_code_enter),
            onClick = onVerifyClicked,
            enabled = busy == null && codeReady,
        )
        if (busy != null) {
            BusyCopy(
                if (busy == OnboardingBusy.SENDING_CODE) {
                    R.string.onboarding_busy_sending_code
                } else {
                    R.string.onboarding_busy_verifying
                },
            )
        } else {
            TextButton(
                onClick = onResendClicked,
                enabled = rateLimitSeconds == 0,
                modifier = Modifier.align(Alignment.CenterHorizontally),
            ) {
                Text(stringResource(R.string.onboarding_code_resend))
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun RateLimitCopy(seconds: Int) {
    Text(
        text = stringResource(R.string.onboarding_error_rate_limited, seconds),
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.error,
        modifier = Modifier.padding(bottom = 8.dp),
    )
}

@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun BusyCopy(labelRes: Int) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.padding(top = 12.dp),
    ) {
        CircularProgressIndicator(
            modifier = Modifier.size(16.dp),
            strokeWidth = 2.dp,
        )
        Text(
            text = stringResource(labelRes),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(start = 8.dp),
        )
    }
}

private const val CODE_LENGTH = 6

// The stable Problem.code of the contract maps to the brief's copy; the raw
// detail never reaches the user.
private fun phoneErrorCopy(error: PhoneError?): Int? =
    when (error) {
        PhoneError.INVALID -> R.string.onboarding_error_phone_invalid
        PhoneError.OFFLINE -> R.string.onboarding_error_offline
        PhoneError.RETRY -> R.string.onboarding_error_retry
        null -> null
    }

private fun codeErrorCopy(error: CodeError?): Int? =
    when (error) {
        CodeError.INVALID -> R.string.onboarding_error_otp_invalid
        CodeError.EXPIRED -> R.string.onboarding_error_otp_expired
        CodeError.OFFLINE -> R.string.onboarding_error_offline
        CodeError.RETRY -> R.string.onboarding_error_retry
        null -> null
    }
