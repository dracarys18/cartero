package app.cartero.data

import app.cartero.nowMillis
import androidx.paging.Pager
import androidx.paging.PagingConfig
import androidx.paging.PagingData
import app.cartero.data.db.ArticleDao
import app.cartero.data.db.ArticleEntity
import app.cartero.data.db.ArticleRow
import app.cartero.data.db.Facet
import app.cartero.data.devices.DeviceSync
import app.cartero.data.sync.ContentLoader
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.launch

data class FeedFilter(
    val topics: Set<String> = emptySet(),
    val sources: Set<String> = emptySet(),
    val unreadOnly: Boolean = false,
) {
    val isEmpty: Boolean get() = topics.isEmpty() && sources.isEmpty()
}

class ArticleRepository(
    private val dao: ArticleDao,
    private val content: ContentLoader,
    private val devices: DeviceSync,
    private val scope: CoroutineScope,
) {
    val topics: Flow<List<Facet>> = dao.topics()
    val sources: Flow<List<Facet>> = dao.sources()
    val unreadCount: Flow<Int> = dao.unreadCount()

    fun feed(filter: FeedFilter): Flow<PagingData<ArticleRow>> =
        Pager(PAGING) {
            dao.feed(
                topics = filter.topics.toList(),
                anyTopic = filter.topics.isEmpty(),
                sources = filter.sources.toList(),
                anySource = filter.sources.isEmpty(),
                unreadOnly = filter.unreadOnly,
            )
        }.flow

    fun saved(): Flow<PagingData<ArticleRow>> = Pager(PAGING) { dao.saved() }.flow

    fun article(id: Long): Flow<ArticleEntity?> = dao.observe(id)

    suspend fun setRead(id: Long, read: Boolean) = dao.setRead(id, read)

    suspend fun markAllRead(filter: FeedFilter) = dao.markAllRead(
        topics = filter.topics.toList(),
        anyTopic = filter.topics.isEmpty(),
        sources = filter.sources.toList(),
        anySource = filter.sources.isEmpty(),
        unreadOnly = filter.unreadOnly,
    )

    suspend fun toggleSaved(id: Long) {
        dao.toggleSaved(id, nowMillis())
        devices.savedChanged(id)
        prefetchIfSaved(id)
    }

    suspend fun save(id: Long) {
        dao.save(listOf(id), nowMillis())
        devices.savedChanged(id)
        prefetchIfSaved(id)
    }

    suspend fun loadFullText(article: ArticleEntity): Boolean = content.fullText(article) != null

    private suspend fun prefetchIfSaved(id: Long) {
        val article = dao.get(id)?.takeIf { it.savedAt != null } ?: return
        scope.launch { content.prefetch(article) }
    }

    private companion object {
        val PAGING = PagingConfig(pageSize = 40, prefetchDistance = 30, initialLoadSize = 80, enablePlaceholders = false)
    }
}
