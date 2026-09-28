package app.cartero.ui.theme

import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import app.cartero.resources.Res
import app.cartero.resources.manrope_bold
import app.cartero.resources.manrope_extrabold
import app.cartero.resources.manrope_medium
import app.cartero.resources.manrope_regular
import app.cartero.resources.manrope_semibold
import org.jetbrains.compose.resources.Font

@Composable
fun rememberCarteroTypography(): Typography {
    val manrope = FontFamily(
        Font(Res.font.manrope_regular, FontWeight.Normal),
        Font(Res.font.manrope_medium, FontWeight.Medium),
        Font(Res.font.manrope_semibold, FontWeight.SemiBold),
        Font(Res.font.manrope_bold, FontWeight.Bold),
        Font(Res.font.manrope_extrabold, FontWeight.ExtraBold),
    )
    return remember(manrope) { carteroTypography(manrope) }
}

@OptIn(ExperimentalMaterial3ExpressiveApi::class)
private fun carteroTypography(family: FontFamily): Typography = Typography().run {
    copy(
        displayLarge = displayLarge.with(family),
        displayMedium = displayMedium.with(family),
        displaySmall = displaySmall.with(family),
        headlineLarge = headlineLarge.with(family),
        headlineMedium = headlineMedium.with(family),
        headlineSmall = headlineSmall.with(family),
        titleLarge = titleLarge.with(family),
        titleMedium = titleMedium.with(family),
        titleSmall = titleSmall.with(family),
        bodyLarge = bodyLarge.with(family),
        bodyMedium = bodyMedium.with(family),
        bodySmall = bodySmall.with(family),
        labelLarge = labelLarge.with(family),
        labelMedium = labelMedium.with(family),
        labelSmall = labelSmall.with(family),
        displayLargeEmphasized = displayLargeEmphasized.with(family),
        displayMediumEmphasized = displayMediumEmphasized.with(family),
        displaySmallEmphasized = displaySmallEmphasized.with(family),
        headlineLargeEmphasized = headlineLargeEmphasized.with(family),
        headlineMediumEmphasized = headlineMediumEmphasized.with(family),
        headlineSmallEmphasized = headlineSmallEmphasized.with(family),
        titleLargeEmphasized = titleLargeEmphasized.with(family),
        titleMediumEmphasized = titleMediumEmphasized.with(family),
        titleSmallEmphasized = titleSmallEmphasized.with(family),
        bodyLargeEmphasized = bodyLargeEmphasized.with(family),
        bodyMediumEmphasized = bodyMediumEmphasized.with(family),
        bodySmallEmphasized = bodySmallEmphasized.with(family),
        labelLargeEmphasized = labelLargeEmphasized.with(family),
        labelMediumEmphasized = labelMediumEmphasized.with(family),
        labelSmallEmphasized = labelSmallEmphasized.with(family),
    )
}

private fun TextStyle.with(family: FontFamily) = copy(fontFamily = family)
