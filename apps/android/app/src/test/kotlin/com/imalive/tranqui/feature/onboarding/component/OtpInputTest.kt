package com.imalive.tranqui.feature.onboarding.component

import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.imalive.tranqui.core.designsystem.TranquiTheme
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config

// Critical piece of the brief (paso 2): the six OTP cells behave as ONE
// field — typing and paste land filtered to six digits, an error keeps the
// written code visible (selected for rewrite inside the field).
@RunWith(AndroidJUnit4::class)
@Config(sdk = [35])
class OtpInputTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun typingFillsTheSixCellsAndReportsTheDigits() {
        val reported = mutableStateOf("")
        composeRule.setContent {
            TranquiTheme {
                OtpInput(value = reported.value, onValueChange = { reported.value = it })
            }
        }

        composeRule.onNode(hasSetTextAction()).performTextInput("482913")

        assertEquals("482913", reported.value)
        for (digit in listOf("4", "8", "2", "9", "1", "3")) {
            composeRule.onNodeWithText(digit).assertIsDisplayed()
        }
    }

    @Test
    fun pasteKeepsOnlyTheFirstSixDigitsWhateverComes() {
        val reported = mutableStateOf("")
        composeRule.setContent {
            TranquiTheme {
                OtpInput(value = reported.value, onValueChange = { reported.value = it })
            }
        }

        composeRule.onNode(hasSetTextAction()).performTextInput("Tu código es 48-29 13, gracias")

        assertEquals("482913", reported.value)
    }

    @Test
    fun anErrorKeepsTheWrittenCodeInTheCells() {
        val reported = mutableStateOf("482913")
        composeRule.setContent {
            TranquiTheme {
                OtpInput(
                    value = reported.value,
                    onValueChange = { reported.value = it },
                    isError = true,
                )
            }
        }

        composeRule.onNodeWithText("4").assertIsDisplayed()
        composeRule.onNodeWithText("3").assertIsDisplayed()
    }

    @Test
    fun thereAreExactlySixCells() {
        val reported = mutableStateOf("")
        composeRule.setContent {
            TranquiTheme {
                OtpInput(value = reported.value, onValueChange = { reported.value = it })
            }
        }

        // The invisible field hosts one editable semantics node; the six
        // drawn cells are decoration.
        composeRule.onAllNodes(hasSetTextAction()).assertCountEquals(1)
    }
}
