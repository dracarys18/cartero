package app.cartero

import app.cartero.data.ArticleRepository
import app.cartero.data.FeedRepository
import app.cartero.data.RuleRepository
import app.cartero.data.db.CarteroDatabase
import app.cartero.data.devices.DeviceStore
import app.cartero.data.devices.DeviceSync
import app.cartero.data.feed.FeedFetcher
import app.cartero.data.reader.ArticleExtractor
import app.cartero.data.settings.SettingsRepository
import app.cartero.data.sync.ContentLoader
import app.cartero.data.sync.SyncEngine
import app.cartero.notify.Notifier
import app.cartero.platform.PlatformServices
import coil3.ImageLoader
import coil3.network.ktor3.KtorNetworkFetcherFactory
import coil3.request.crossfade
import io.ktor.client.HttpClient
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.UserAgent
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob

class AppGraph(private val platform: PlatformServices) {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    val http: HttpClient by lazy {
        HttpClient(platform.httpEngine) {
            expectSuccess = false
            install(UserAgent) { agent = USER_AGENT }
            install(HttpTimeout) {
                connectTimeoutMillis = 15_000
                socketTimeoutMillis = 20_000
                requestTimeoutMillis = 60_000
            }
        }
    }

    val imageLoader: ImageLoader by lazy {
        ImageLoader.Builder(platform.context)
            .components { add(KtorNetworkFetcherFactory(httpClient = { http })) }
            .crossfade(CROSSFADE_MILLIS)
            .build()
    }

    val settings by lazy { SettingsRepository(platform.dataStore) }
    val notifier: Notifier by lazy { platform.notifier(imageLoader) }

    private val database by lazy { CarteroDatabase.build(platform.database) }
    private val content by lazy { ContentLoader(platform.context, database.articles(), ArticleExtractor(http), imageLoader) }

    val sync by lazy { SyncEngine(database, FeedFetcher(http), settings, notifier, content) }
    val devices by lazy {
        DeviceSync(
            network = platform.peers,
            store = DeviceStore(platform.dataStore),
            articles = database.articles(),
            feeds = database.feeds(),
            rules = database.rules(),
            notifier = notifier,
            content = content,
            deviceName = platform.deviceName,
            scope = scope,
        )
    }
    val articles by lazy { ArticleRepository(database.articles(), content, devices, scope) }
    val feeds by lazy { FeedRepository(database.feeds(), sync, scope) }
    val rules by lazy { RuleRepository(database.rules(), database.articles(), notifier, content, devices, scope) }

    private companion object {
        const val CROSSFADE_MILLIS = 150
        const val USER_AGENT = "Mozilla/5.0 (Linux; Android 17) AppleWebKit/537.36 (KHTML, like Gecko) " +
            "Chrome/140.0 Mobile Safari/537.36 Cartero/1.0"
    }
}
