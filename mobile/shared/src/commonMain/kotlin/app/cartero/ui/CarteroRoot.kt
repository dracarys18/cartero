package app.cartero.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.lifecycle.compose.LifecycleResumeEffect
import app.cartero.AppGraph
import app.cartero.platform.LocalPlatformActions
import app.cartero.platform.PlatformActions
import app.cartero.ui.navigation.AppNavigation
import app.cartero.ui.navigation.AppRequest
import app.cartero.ui.platform.rememberNotificationPermission
import app.cartero.ui.theme.CarteroTheme
import coil3.compose.setSingletonImageLoaderFactory
import kotlinx.coroutines.launch

@Composable
fun CarteroRoot(
    graph: AppGraph,
    actions: PlatformActions,
    request: AppRequest?,
    onRequestHandled: () -> Unit,
) {
    setSingletonImageLoaderFactory { graph.imageLoader }
    val requestNotifications = rememberNotificationPermission {
        graph.scope.launch { graph.settings.update { it.copy(notificationsAsked = true) } }
    }
    LaunchedEffect(Unit) {
        if (!graph.notifier.enabled && !graph.settings.current().notificationsAsked) requestNotifications()
    }
    LifecycleResumeEffect(Unit) {
        graph.devices.resume()
        onPauseOrDispose {}
    }
    CompositionLocalProvider(LocalGraph provides graph, LocalPlatformActions provides actions) {
        CarteroTheme {
            AppNavigation(request = request, onRequestHandled = onRequestHandled)
        }
    }
}
