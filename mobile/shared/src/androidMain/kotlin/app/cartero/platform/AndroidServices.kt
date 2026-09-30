package app.cartero.platform

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.os.Build
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.PreferenceDataStoreFactory
import androidx.datastore.preferences.core.Preferences
import androidx.room3.Room
import androidx.room3.RoomDatabase
import app.cartero.data.db.CarteroDatabase
import app.cartero.data.devices.PeerNetwork
import app.cartero.notify.AndroidNotifier
import app.cartero.notify.NotificationIcons
import app.cartero.notify.Notifier
import coil3.ImageLoader
import coil3.PlatformContext
import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.okhttp.OkHttp
import okio.Path.Companion.toPath

class AndroidServices(
    private val app: Context,
    private val icons: NotificationIcons,
    private val launcher: Class<out Activity>,
    private val receiver: Class<out BroadcastReceiver>,
) : PlatformServices {
    override val context: PlatformContext = app

    override val database: RoomDatabase.Builder<CarteroDatabase> =
        Room.databaseBuilder<CarteroDatabase>(app, app.getDatabasePath(CarteroDatabase.FILE_NAME).absolutePath)

    override val dataStore: DataStore<Preferences> = PreferenceDataStoreFactory.createWithPath(
        produceFile = { app.filesDir.resolve("datastore/settings.preferences_pb").absolutePath.toPath() },
    )

    override val httpEngine: HttpClientEngine = OkHttp.create()

    override val peers: PeerNetwork = IrohPeers(app)

    override val deviceName: String = Build.MODEL

    override fun notifier(imageLoader: ImageLoader): Notifier =
        AndroidNotifier(app, imageLoader, icons, launcher, receiver)
}
