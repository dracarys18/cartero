package app.cartero.ui.navigation

import androidx.navigation3.runtime.NavKey
import app.cartero.resources.Res
import app.cartero.resources.ic_bookmarks
import app.cartero.resources.ic_bookmarks_filled
import app.cartero.resources.ic_newspaper
import app.cartero.resources.ic_newspaper_filled
import app.cartero.resources.ic_rss_feed
import androidx.savedstate.serialization.SavedStateConfiguration
import kotlinx.serialization.Serializable
import kotlinx.serialization.modules.SerializersModule
import kotlinx.serialization.modules.polymorphic
import kotlinx.serialization.modules.subclass
import org.jetbrains.compose.resources.DrawableResource

@Serializable
sealed interface TabRoute : NavKey

@Serializable
data object FeedRoute : TabRoute

@Serializable
data object SavedRoute : TabRoute

@Serializable
data object RulesRoute : NavKey

@Serializable
data object FeedsRoute : TabRoute

@Serializable
data class ReaderRoute(val articleId: Long) : NavKey

@Serializable
data object SettingsRoute : NavKey

@Serializable
data object DevicesRoute : NavKey

enum class Tab(
    val route: TabRoute,
    val label: String,
    val icon: DrawableResource,
    val selectedIcon: DrawableResource,
) {
    Feed(FeedRoute, "Feed", Res.drawable.ic_newspaper, Res.drawable.ic_newspaper_filled),
    Saved(SavedRoute, "Saved", Res.drawable.ic_bookmarks, Res.drawable.ic_bookmarks_filled),
    Feeds(FeedsRoute, "Feeds", Res.drawable.ic_rss_feed, Res.drawable.ic_rss_feed),
}

val NavigationState = SavedStateConfiguration {
    serializersModule = SerializersModule {
        polymorphic(NavKey::class) {
            subclass(FeedRoute::class)
            subclass(SavedRoute::class)
            subclass(FeedsRoute::class)
            subclass(RulesRoute::class)
            subclass(ReaderRoute::class)
            subclass(SettingsRoute::class)
            subclass(DevicesRoute::class)
        }
    }
}
