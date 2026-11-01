@file:Suppress("MagicNumber")
// The provisional theme (issue #75 slice 3): /design/tokens.json does not
// exist yet and brand/primary awaits the product name (#5), so the brief's
// supuesto applies — ok/primary stands in as the hero color. This file is
// the home of named tokens: its literals ARE the values (pending the
// generator), so MagicNumber is suppressed by design.

package com.imalive.tranqui.core.designsystem

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

// The token values live in this one file until the tokens
// generator replaces them (design-system.md; TODO #5 / tokens.json).
private val OkPrimaryLight = Color(0xFF2E7D5B) // ok/primary, AA sobre blanco
private val OkPrimaryDark = Color(0xFF8AD6AE) // ok/primary lifted for dark AA

private val OkContainerLight = Color(0xFFB9EFD2)
private val OkContainerDark = Color(0xFF17553A)

// Neutral cálidos, ligeramente sepia — no grises puros (design-system.md).
private val NeutralBgLight = Color(0xFFFAF8F4)
private val NeutralSurfaceLight = Color(0xFFFFFFFF)
private val NeutralTextLight = Color(0xFF1D1B17)
private val NeutralBgDark = Color(0xFF151310)
private val NeutralSurfaceDark = Color(0xFF201D19)
private val NeutralTextDark = Color(0xFFEAE4DB)

// incident/critical: errores inline; reservado, nunca decorativo.
private val IncidentCriticalLight = Color(0xFFC63B33)
private val IncidentCriticalDark = Color(0xFFF2B8B5)

private val LightColors =
    lightColorScheme(
        primary = OkPrimaryLight,
        onPrimary = Color.White,
        primaryContainer = OkContainerLight,
        background = NeutralBgLight,
        onBackground = NeutralTextLight,
        surface = NeutralSurfaceLight,
        onSurface = NeutralTextLight,
        surfaceVariant = NeutralBgLight,
        onSurfaceVariant = NeutralTextLight,
        error = IncidentCriticalLight,
        onError = Color.White,
    )

private val DarkColors =
    darkColorScheme(
        primary = OkPrimaryDark,
        onPrimary = Color(0xFF00391F),
        primaryContainer = OkContainerDark,
        background = NeutralBgDark,
        onBackground = NeutralTextDark,
        surface = NeutralSurfaceDark,
        onSurface = NeutralTextDark,
        surfaceVariant = NeutralSurfaceDark,
        onSurfaceVariant = NeutralTextDark,
        error = IncidentCriticalDark,
        onError = Color(0xFF601410),
    )

// The shared type scale (design-system.md): display 32/40, headline 24/32,
// title 18/26, body 16/24, label 13/18. Roboto (system font) only.
private val TranquiTypography =
    Typography(
        displaySmall =
            TextStyle(
                fontWeight = FontWeight.Normal,
                fontSize = 32.sp,
                lineHeight = 40.sp,
            ),
        headlineSmall =
            TextStyle(
                fontWeight = FontWeight.Normal,
                fontSize = 24.sp,
                lineHeight = 32.sp,
            ),
        titleMedium =
            TextStyle(
                fontWeight = FontWeight.Medium,
                fontSize = 18.sp,
                lineHeight = 26.sp,
            ),
        bodyLarge = TextStyle(fontSize = 16.sp, lineHeight = 24.sp),
        bodyMedium = TextStyle(fontSize = 16.sp, lineHeight = 24.sp),
        labelLarge =
            TextStyle(
                fontWeight = FontWeight.Medium,
                fontSize = 13.sp,
                lineHeight = 18.sp,
            ),
    )

// Dark mode is mandatory (the nightly check-in is real, design-system.md).
@Composable
@Suppress("ktlint:standard:function-naming") // PascalCase is the Compose convention
fun TranquiTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = if (darkTheme) DarkColors else LightColors,
        typography = TranquiTypography,
        content = content,
    )
}
