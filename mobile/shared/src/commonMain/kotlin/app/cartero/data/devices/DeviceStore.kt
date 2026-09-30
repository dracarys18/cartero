package app.cartero.data.devices

import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.serialization.Serializable

@Serializable
data class LinkedDevice(val id: String, val name: String, val syncedAt: Long? = null)

class DeviceStore(private val store: DataStore<Preferences>) {
    val devices: Flow<List<LinkedDevice>> = store.data.map { it.devices() }.distinctUntilChanged()

    suspend fun secretKey(): String? = store.data.first()[Keys.secret]

    suspend fun setSecretKey(key: String) {
        store.edit { it[Keys.secret] = key }
    }

    suspend fun device(id: String): LinkedDevice? = devices.first().firstOrNull { it.id == id }

    suspend fun updateDevices(transform: (List<LinkedDevice>) -> List<LinkedDevice>) {
        store.edit { it[Keys.devices] = DeviceJson.encodeToString(transform(it.devices())) }
    }

    suspend fun removals(): Map<String, Long> = store.data.first().stamps(Keys.removals)

    suspend fun updateRemovals(now: Long, transform: (MutableMap<String, Long>) -> Unit) =
        updateStamps(Keys.removals) { removals ->
            transform(removals)
            removals.values.removeAll { now - it > REMOVAL_TTL_MILLIS }
        }

    suspend fun ruleStamps(): Map<String, Long> = store.data.first().stamps(Keys.ruleStamps)

    suspend fun updateRuleStamps(transform: (MutableMap<String, Long>) -> Unit) =
        updateStamps(Keys.ruleStamps, transform)

    private suspend fun updateStamps(key: Preferences.Key<String>, transform: (MutableMap<String, Long>) -> Unit) {
        store.edit { prefs ->
            val stamps = prefs.stamps(key).toMutableMap()
            transform(stamps)
            prefs[key] = DeviceJson.encodeToString(stamps)
        }
    }

    private fun Preferences.devices(): List<LinkedDevice> =
        this[Keys.devices]?.let { DeviceJson.decodeFromString<List<LinkedDevice>>(it) }.orEmpty()

    private fun Preferences.stamps(key: Preferences.Key<String>): Map<String, Long> =
        this[key]?.let { DeviceJson.decodeFromString<Map<String, Long>>(it) }.orEmpty()

    private object Keys {
        val secret = stringPreferencesKey("device_secret_key")
        val devices = stringPreferencesKey("linked_devices")
        val removals = stringPreferencesKey("saved_removals")
        val ruleStamps = stringPreferencesKey("rule_stamps")
    }

    private companion object {
        const val REMOVAL_TTL_MILLIS = 180L * 24 * 60 * 60 * 1000
    }
}
