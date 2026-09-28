package app.cartero.data

import app.cartero.data.db.CarteroDatabase
import app.cartero.data.db.FeedDao
import app.cartero.data.db.FeedEntity
import app.cartero.data.db.FeedWithStats
import app.cartero.data.feed.Html
import app.cartero.data.sync.SyncEngine
import com.fleeksoft.ksoup.Ksoup
import com.fleeksoft.ksoup.parser.Parser
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class FeedRepository(
    private val dao: FeedDao,
    private val sync: SyncEngine,
    private val scope: CoroutineScope,
) {
    val feeds: Flow<List<FeedWithStats>> = dao.observe()
    val count: Flow<Int> = dao.count()

    suspend fun subscribe(url: String): Long = sync.subscribe(url)

    suspend fun rename(id: Long, title: String) = dao.rename(id, title.trim())

    suspend fun delete(id: Long) = dao.delete(id, CarteroDatabase.DEFAULT_FEED_URL)

    suspend fun importOpml(opml: String): Int {
        val feeds = withContext(Dispatchers.Default) {
            Ksoup.parse(opml, parser = Parser.xmlParser()).select("outline[xmlUrl]").map { outline ->
                val url = outline.attr("xmlUrl")
                FeedEntity(
                    url = url,
                    title = outline.attr("title").ifEmpty { outline.attr("text") }.ifEmpty { url },
                    siteUrl = outline.attr("htmlUrl").ifEmpty { null },
                )
            }
        }
        val added = dao.insertAll(feeds).count { it != -1L }
        if (added > 0) scope.launch { sync.syncAll() }
        return added
    }

    suspend fun exportOpml(): String {
        val feeds = dao.all()
        return buildString {
            append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
            append("<opml version=\"2.0\">\n<head><title>Cartero subscriptions</title></head>\n<body>\n")
            for (feed in feeds) {
                val title = Html.escape(feed.title)
                append("  <outline type=\"rss\" text=\"").append(title).append("\" title=\"").append(title)
                append("\" xmlUrl=\"").append(Html.escape(feed.url)).append('"')
                feed.siteUrl?.let { append(" htmlUrl=\"").append(Html.escape(it)).append('"') }
                append("/>\n")
            }
            append("</body>\n</opml>\n")
        }
    }
}
