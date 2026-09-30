package app.cartero.data.devices

import app.cartero.data.db.ArticleDao
import app.cartero.data.db.ArticleEntity
import app.cartero.data.db.CarteroDatabase
import app.cartero.data.db.FeedDao
import app.cartero.data.db.SavedRow
import app.cartero.data.sync.ContentLoader
import app.cartero.nowMillis
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withTimeout
import kotlin.uuid.ExperimentalUuidApi
import kotlin.uuid.Uuid

class DeviceSync(
    private val network: PeerNetwork,
    private val store: DeviceStore,
    private val articles: ArticleDao,
    private val feeds: FeedDao,
    private val content: ContentLoader,
    private val deviceName: String,
    private val scope: CoroutineScope,
) {
    val devices: Flow<List<LinkedDevice>> = store.devices

    private val invite = MutableStateFlow<Invite?>(null)
    val inviteCode: Flow<String?> = invite.map { it?.code }

    private val syncingState = MutableStateFlow(false)
    val syncing: StateFlow<Boolean> = syncingState.asStateFlow()

    private val startLock = Mutex()
    private val syncLock = Mutex()
    private val mergeLock = Mutex()
    private val pushes = Channel<Unit>(Channel.CONFLATED)
    private var identity: PeerIdentity? = null

    init {
        scope.launch {
            pushes.receiveAsFlow().collect {
                delay(PUSH_DELAY_MILLIS)
                syncAll()
            }
        }
    }

    fun resume() {
        scope.launch { syncAll() }
    }

    @OptIn(ExperimentalUuidApi::class)
    suspend fun createInvite() {
        val secret = Uuid.random().toHexString()
        val code = LINK_URL_PREFIX + start().id + CODE_SEPARATOR + secret
        invite.value = Invite(code, secret, nowMillis() + INVITE_TTL_MILLIS)
    }

    fun cancelInvite() {
        invite.value = null
    }

    suspend fun link(code: String): LinkedDevice {
        val parts = code.trim().removePrefix(LINK_URL_PREFIX).split(CODE_SEPARATOR)
        require(parts.size == 2 && parts.none(String::isBlank)) { "That isn't a link code" }
        val (peer, secret) = parts
        require(peer != start().id) { "That's this device's own code" }
        val reply = send(peer, DeviceMessage.Link(secret, deviceName)) as? DeviceMessage.Linked
            ?: error("The other device didn't accept the code")
        val device = LinkedDevice(peer, reply.name)
        store.updateDevices { devices -> devices.filterNot { it.id == peer } + device }
        syncLock.withLock { syncWith(device) }
        return device
    }

    suspend fun unlink(device: LinkedDevice) {
        store.updateDevices { devices -> devices.filterNot { it.id == device.id } }
    }

    suspend fun syncAll(): Int = syncLock.withLock {
        val devices = store.devices.first()
        if (devices.isEmpty()) return 0
        syncingState.value = true
        try {
            devices.count { device -> runCatching { syncWith(device) }.isFailure }
        } finally {
            syncingState.value = false
        }
    }

    suspend fun savedChanged(articleId: Long) {
        val article = articles.get(articleId) ?: return
        val url = article.url ?: return
        val now = nowMillis()
        store.updateRemovals(now) { removals ->
            if (article.savedAt == null) removals[url] = now else removals.remove(url)
        }
        if (store.devices.first().isNotEmpty()) pushes.trySend(Unit)
    }

    private suspend fun syncWith(device: LinkedDevice) {
        val reply = send(device.id, DeviceMessage.Sync(changes())) as? DeviceMessage.Sync
            ?: error("Unexpected reply from ${device.name}")
        merge(reply.changes)
        markSynced(device.id)
    }

    private suspend fun send(peer: String, message: DeviceMessage): DeviceMessage {
        start()
        val reply = withTimeout(REQUEST_TIMEOUT_MILLIS) {
            network.request(peer, DeviceJson.encodeToString<DeviceMessage>(message))
        }
        return DeviceJson.decodeFromString<DeviceMessage>(reply)
    }

    private suspend fun start(): PeerIdentity = startLock.withLock {
        identity ?: network.start(store.secretKey(), ::handle).also {
            store.setSecretKey(it.secretKey)
            identity = it
        }
    }

    private suspend fun handle(peer: String, message: String): String? {
        val request = runCatching { DeviceJson.decodeFromString<DeviceMessage>(message) }.getOrNull() ?: return null
        val reply = when (request) {
            is DeviceMessage.Link -> acceptLink(peer, request)
            is DeviceMessage.Sync -> store.device(peer)?.let {
                merge(request.changes)
                markSynced(peer)
                DeviceMessage.Sync(changes())
            }
            is DeviceMessage.Linked -> null
        } ?: return null
        return DeviceJson.encodeToString<DeviceMessage>(reply)
    }

    private suspend fun acceptLink(peer: String, request: DeviceMessage.Link): DeviceMessage? {
        val current = invite.value ?: return null
        if (current.secret != request.secret || current.expiresAt < nowMillis()) return null
        if (!invite.compareAndSet(current, null)) return null
        store.updateDevices { devices -> devices.filterNot { it.id == peer } + LinkedDevice(peer, request.name) }
        return DeviceMessage.Linked(deviceName)
    }

    private suspend fun markSynced(id: String) {
        val now = nowMillis()
        store.updateDevices { devices -> devices.map { if (it.id == id) it.copy(syncedAt = now) else it } }
    }

    private suspend fun changes(): List<SavedChange> {
        val saved = articles.savedRows().distinctBy { it.url }
        val savedAt = saved.associate { it.url to it.savedAt }
        val removed = store.removals().filter { (url, at) -> (savedAt[url] ?: Long.MIN_VALUE) < at }
        return saved.map { SavedChange(it.url, it.savedAt, it.story()) } + removed.map { (url, at) -> SavedChange(url, at) }
    }

    private suspend fun merge(changes: List<SavedChange>) = mergeLock.withLock<Unit> {
        val savedAt = articles.savedRows().associate { it.url to it.savedAt }
        val removals = store.removals()
        val removed = mutableMapOf<String, Long>()
        val restored = mutableSetOf<String>()
        val added = mutableListOf<ArticleEntity>()
        for (change in changes) {
            val local = maxOf(savedAt[change.url] ?: Long.MIN_VALUE, removals[change.url] ?: Long.MIN_VALUE)
            if (change.at <= local) continue
            val story = change.story
            if (story == null) {
                articles.setSavedAt(change.url, null)
                removed[change.url] = change.at
            } else {
                save(change.url, story, change.at)?.let(added::add)
                restored += change.url
            }
        }
        if (removed.isNotEmpty() || restored.isNotEmpty()) {
            store.updateRemovals(nowMillis()) { it.keys.removeAll(restored); it.putAll(removed) }
        }
        if (added.isNotEmpty()) scope.launch { content.prefetch(added) }
    }

    private suspend fun save(url: String, story: SavedStory, at: Long): ArticleEntity? {
        val existing = articles.byUrl(url)
        if (existing == null) {
            val feed = feeds.byUrl(story.feedUrl) ?: feeds.byUrl(CarteroDatabase.DEFAULT_FEED_URL) ?: return null
            articles.insert(listOf(story.toArticle(feed.id, url)))
        }
        articles.setSavedAt(url, at)
        return if (existing == null) articles.byUrl(url) else null
    }

    private class Invite(val code: String, val secret: String, val expiresAt: Long)

    companion object {
        const val LINK_URL_PREFIX = "cartero://link?code="
        private const val CODE_SEPARATOR = "."
        private const val PUSH_DELAY_MILLIS = 3_000L
        private const val REQUEST_TIMEOUT_MILLIS = 20_000L
        private const val INVITE_TTL_MILLIS = 10 * 60 * 1000L
    }
}

private fun SavedRow.story() = SavedStory(
    guid = guid,
    feedUrl = feedUrl,
    title = title,
    summary = summary,
    imageUrl = imageUrl,
    author = author,
    source = source,
    topic = topic,
    publishedAt = publishedAt,
    readingMinutes = readingMinutes,
)

private fun SavedStory.toArticle(feedId: Long, url: String) = ArticleEntity(
    feedId = feedId,
    guid = guid,
    url = url,
    title = title,
    summary = summary,
    content = null,
    imageUrl = imageUrl,
    author = author,
    source = source,
    topic = topic,
    publishedAt = publishedAt,
    readingMinutes = readingMinutes,
    isRead = true,
)
