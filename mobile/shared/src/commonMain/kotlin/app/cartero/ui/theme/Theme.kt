package app.cartero.ui.theme

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.MaterialExpressiveTheme
import androidx.compose.material3.MotionScheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

@Composable
expect fun platformColorScheme(): ColorScheme

@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun CarteroTheme(content: @Composable () -> Unit) {
    MaterialExpressiveTheme(
        colorScheme = platformColorScheme(),
        motionScheme = MotionScheme.expressive(),
        typography = rememberCarteroTypography(),
        content = content,
    )
}

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
    background = Color(0xFF191817),
    onBackground = Color(0xFFEBE8E0),
    surface = Color(0xFF191817),
    onSurface = Color(0xFFEBE8E0),
    surfaceVariant = Color(0xFF423E37),
    onSurfaceVariant = Color(0xFFB0ACA2),
    surfaceTint = Color(0xFFA9BCDD),
    inverseSurface = Color(0xFFEBE8E0),
    inverseOnSurface = Color(0xFF2C2A28),
    outline = Color(0xFF827E74),
    outlineVariant = Color(0xFF33302A),
    surfaceBright = Color(0xFF3A3835),
    surfaceDim = Color(0xFF151413),
    surfaceContainerLowest = Color(0xFF121110),
    surfaceContainerLow = Color(0xFF1E1D1B),
    surfaceContainer = Color(0xFF232120),
    surfaceContainerHigh = Color(0xFF2C2A28),
    surfaceContainerHighest = Color(0xFF373432),
)
