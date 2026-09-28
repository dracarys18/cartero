package app.cartero.ui.theme

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext

@Composable
actual fun platformColorScheme(): ColorScheme {
    val context = LocalContext.current
    return remember(context) { dynamicDarkColorScheme(context) }
}
