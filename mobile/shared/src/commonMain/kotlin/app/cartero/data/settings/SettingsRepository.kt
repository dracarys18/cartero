package app.cartero.data.settings

import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.booleanPreferencesKey
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.floatPreferencesKey
import androidx.datastore.preferences.core.intPreferencesKey
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map

data class Settings(
    val syncMinutes: Int = 60,
    val wifiOnly: Boolean = false,
    val retentionDays: Int = 30,
    val readerScale: Float = 1f,
    val unreadOnly: Boolean = false,
    val inAppBrowser: Boolean = true,
    val notificationsAsked: Boolean = false,
)

class SettingsRepository(private val store: DataStore<Preferences>) {
    val settings: Flow<Settings> = store.data.map { it.toSettings() }.distinctUntilChanged()

    suspend fun current(): Settings = settings.first()

    suspend fun update(transform: (Settings) -> Settings) {
        store.edit { prefs ->
            val next = transform(prefs.toSettings())
            prefs[Keys.syncMinutes] = next.syncMinutes
            prefs[Keys.wifiOnly] = next.wifiOnly
            prefs[Keys.retentionDays] = next.retentionDays
            prefs[Keys.readerScale] = next.readerScale
            prefs[Keys.unreadOnly] = next.unreadOnly
            prefs[Keys.inAppBrowser] = next.inAppBrowser
            prefs[Keys.notificationsAsked] = next.notificationsAsked
        }
    }

    private fun Preferences.toSettings(): Settings {
        val defaults = Settings()
        return Settings(
            syncMinutes = this[Keys.syncMinutes] ?: defaults.syncMinutes,
            wifiOnly = this[Keys.wifiOnly] ?: defaults.wifiOnly,
            retentionDays = this[Keys.retentionDays] ?: defaults.retentionDays,
            readerScale = this[Keys.readerScale] ?: defaults.readerScale,
            unreadOnly = this[Keys.unreadOnly] ?: defaults.unreadOnly,
            inAppBrowser = this[Keys.inAppBrowser] ?: defaults.inAppBrowser,
            notificationsAsked = this[Keys.notificationsAsked] ?: defaults.notificationsAsked,
        )
    }

    private object Keys {
        val syncMinutes = intPreferencesKey("sync_minutes")
        val wifiOnly = booleanPreferencesKey("wifi_only")
        val retentionDays = intPreferencesKey("retention_days")
        val readerScale = floatPreferencesKey("reader_scale")
        val unreadOnly = booleanPreferencesKey("unread_only")
        val inAppBrowser = booleanPreferencesKey("in_app_browser")
        val notificationsAsked = booleanPreferencesKey("notifications_asked")
    }
}
