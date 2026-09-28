package app.cartero.platform

import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import androidx.datastore.preferences.core.Preferences
import androidx.room3.Room
import androidx.room3.RoomDatabase
import androidx.sqlite.driver.bundled.BundledSQLiteDriver
import app.cartero.data.db.CarteroDatabase
import app.cartero.notify.IosNotifier
import app.cartero.notify.Notifier
import coil3.ImageLoader
import coil3.PlatformContext
import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.darwin.Darwin
import kotlinx.cinterop.ExperimentalForeignApi
import okio.Path.Companion.toPath
import platform.Foundation.NSDocumentDirectory
import platform.Foundation.NSFileManager
import platform.Foundation.NSUserDomainMask

class IosServices : PlatformServices {
    override val context: PlatformContext = PlatformContext.INSTANCE

    override val database: RoomDatabase.Builder<CarteroDatabase> =
        Room.databaseBuilder<CarteroDatabase>(name = documentPath(CarteroDatabase.FILE_NAME))
            .setDriver(BundledSQLiteDriver())

    override val dataStore: DataStore<Preferences> = PreferenceDataStoreFactory.createWithPath(
        produceFile = { documentPath("settings.preferences_pb").toPath() },
    )

    override val httpEngine: HttpClientEngine = Darwin.create()

    override fun notifier(imageLoader: ImageLoader): Notifier = IosNotifier()
}

@OptIn(ExperimentalForeignApi::class)
private fun documentPath(name: String): String {
    val directory = NSFileManager.defaultManager.URLForDirectory(NSDocumentDirectory, NSUserDomainMask, null, true, null)
    return requireNotNull(directory?.path) + "/" + name
}
