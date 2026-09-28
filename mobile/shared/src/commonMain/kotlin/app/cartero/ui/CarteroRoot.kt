package app.cartero.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import app.cartero.AppGraph
import app.cartero.platform.LocalPlatformActions
import app.cartero.platform.PlatformActions
import app.cartero.ui.navigation.AppNavigation
import app.cartero.ui.navigation.AppRequest
import app.cartero.ui.theme.CarteroTheme
import coil3.compose.setSingletonImageLoaderFactory

@Composable
fun CarteroRoot(
    graph: AppGraph,
    actions: PlatformActions,
    request: AppRequest?,
    onRequestHandled: () -> Unit,
) {
    setSingletonImageLoaderFactory { graph.imageLoader }
    CompositionLocalProvider(LocalGraph provides graph, LocalPlatformActions provides actions) {
        CarteroTheme {
            AppNavigation(request = request, onRequestHandled = onRequestHandled)
        }
    }
}
