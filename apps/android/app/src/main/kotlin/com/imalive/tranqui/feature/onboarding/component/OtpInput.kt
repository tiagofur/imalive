package com.imalive.tranqui.feature.onboarding.component

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.autofill.ContentType
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.contentType
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp
import com.imalive.tranqui.R

private const val CODE_LENGTH = 6

// The six OTP cells of the brief (paso 2): one real BasicTextField carries
// the input — invisible under the drawn cells — so the SMS autofill
// (contentType SmsOtpCode), paste and the keyboard all land in a single
// field. An error keeps the written digits selected for rewrite (the brief's
// OTP_INVALID state). The VM owns the digits; this piece is stateless.
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun OtpInput(
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    isError: Boolean = false,
) {
    val fieldLabel = stringResource(R.string.onboarding_code_field_a11y)
    val selection =
        if (isError && value.isNotEmpty()) {
            TextRange(0, value.length) // the whole code selected to rewrite
        } else {
            TextRange(value.length)
        }
    BasicTextField(
        value = TextFieldValue(text = value, selection = selection),
        onValueChange = { incoming ->
            onValueChange(incoming.text.filter(Char::isDigit).take(CODE_LENGTH))
        },
        modifier =
            modifier
                .fillMaxWidth()
                .height(56.dp)
                .semantics {
                    contentType = ContentType.SmsOtpCode
                    contentDescription = fieldLabel
                },
        textStyle = TextStyle(color = Color.Transparent),
        cursorBrush = SolidColor(Color.Transparent),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        decorationBox = {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                repeat(CODE_LENGTH) { index ->
                    OtpCell(
                        char = value.getOrNull(index),
                        active = index == value.length,
                        isError = isError,
                        modifier = Modifier.weight(1f),
                    )
                }
            }
        },
    )
}

@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
private fun OtpCell(
    char: Char?,
    active: Boolean,
    isError: Boolean,
    modifier: Modifier = Modifier,
) {
    val borderColor =
        when {
            isError -> MaterialTheme.colorScheme.error
            active -> MaterialTheme.colorScheme.primary
            else -> MaterialTheme.colorScheme.outline
        }
    Box(
        modifier =
            modifier
                .height(56.dp)
                .clip(RoundedCornerShape(8.dp))
                .background(MaterialTheme.colorScheme.surface)
                .border(1.dp, borderColor, RoundedCornerShape(8.dp)),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = char?.toString().orEmpty(),
            style = MaterialTheme.typography.headlineSmall,
        )
    }
}
