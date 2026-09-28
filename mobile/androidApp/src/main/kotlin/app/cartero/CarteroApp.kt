package app.cartero

import android.app.Application
import app.cartero.notify.NotificationActionReceiver
import app.cartero.notify.NotificationIcons
import app.cartero.platform.AndroidServices
import app.cartero.sync.SyncJobService
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch

class CarteroApp : Application() {
    lateinit var graph: AppGraph
        private set

    override fun onCreate() {
        super.onCreate()
        val icons = NotificationIcons(
            small = R.drawable.ic_newspaper_filled,
            save = R.drawable.ic_bookmark,
            read = R.drawable.ic_check,
        )
        graph = AppGraph(AndroidServices(this, icons, MainActivity::class.java, NotificationActionReceiver::class.java))
        graph.scope.launch {
            graph.settings.settings
                .map { it.syncMinutes to it.wifiOnly }
                .distinctUntilChanged()
                .collect { (minutes, wifiOnly) -> SyncJobService.schedule(this@CarteroApp, minutes, wifiOnly) }
        }
    }
}
