package app.cartero.ui.theme

import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.MaterialExpressiveTheme
import androidx.compose.material3.MotionScheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun CarteroTheme(content: @Composable () -> Unit) {
    MaterialExpressiveTheme(
        colorScheme = CarteroDarkColors,
        motionScheme = MotionScheme.expressive(),
        typography = rememberCarteroTypography(),
        content = content,
    )
}

val TopicColor = Color(0xFF5CB3FF)

val CarteroDarkColors = darkColorScheme(
    primary = Color(0xFFA9BCDD),
    onPrimary = Color(0xFF1B2E4B),
    primaryContainer = Color(0xFF34476A),
    onPrimaryContainer = Color(0xFFD5E0F5),
    inversePrimary = Color(0xFF4A5F82),
    secondary = Color(0xFFC9C6BC),
    onSecondary = Color(0xFF31302B),
    secondaryContainer = Color(0xFF423E37),
    onSecondaryContainer = Color(0xFFE6E2D8),
    tertiary = Color(0xFFE0876A),
    onTertiary = Color(0xFF4A1408),
    tertiaryContainer = Color(0xFF6E2A1C),
    onTertiaryContainer = Color(0xFFFFDAD2),
    background = Color(0xFF000000),
    onBackground = Color(0xFFE3DED4),
    surface = Color(0xFF000000),
    onSurface = Color(0xFFE3DED4),
    surfaceVariant = Color(0xFF2A2826),
    onSurfaceVariant = Color(0xFFA39E94),
    surfaceTint = Color(0xFFA9BCDD),
    inverseSurface = Color(0xFFE3DED4),
    inverseOnSurface = Color(0xFF1C1B1A),
    outline = Color(0xFF7A756C),
    outlineVariant = Color(0xFF2A2826),
    surfaceBright = Color(0xFF2E2D2B),
    surfaceDim = Color(0xFF000000),
    surfaceContainerLowest = Color(0xFF000000),
    surfaceContainerLow = Color(0xFF0F0F0E),
    surfaceContainer = Color(0xFF141413),
    surfaceContainerHigh = Color(0xFF1C1B1A),
    surfaceContainerHighest = Color(0xFF262523),
)
