package app.cartero.platform

import androidx.compose.runtime.staticCompositionLocalOf
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.room3.RoomDatabase
import app.cartero.data.db.CarteroDatabase
import app.cartero.data.db.RuleEntity
import app.cartero.notify.Notifier
import coil3.ImageLoader
import coil3.PlatformContext
import io.ktor.client.engine.HttpClientEngine

interface PlatformServices {
    val context: PlatformContext
    val database: RoomDatabase.Builder<CarteroDatabase>
    val dataStore: DataStore<Preferences>
    val httpEngine: HttpClientEngine
    fun notifier(imageLoader: ImageLoader): Notifier
}

interface PlatformActions {
    val appVersion: String
    fun openLink(url: String, inApp: Boolean)
    fun share(title: String, url: String)
    fun openNotificationSettings(rule: RuleEntity?)
}

val LocalPlatformActions = staticCompositionLocalOf<PlatformActions> { error("PlatformActions not provided") }
