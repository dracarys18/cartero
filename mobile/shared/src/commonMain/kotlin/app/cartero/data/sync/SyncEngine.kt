package app.cartero.data.sync

import app.cartero.data.db.ArticleEntity
import app.cartero.data.db.CarteroDatabase
import app.cartero.data.db.FeedEntity
import app.cartero.data.db.RuleAction
import app.cartero.data.feed.FeedFetcher
import app.cartero.data.feed.FetchResult
import app.cartero.data.feed.Html
import app.cartero.data.feed.ParsedItem
import app.cartero.data.settings.SettingsRepository
import app.cartero.notify.Notifier
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.sync.withPermit
import kotlinx.coroutines.withContext
import app.cartero.nowMillis
import kotlinx.io.IOException
import kotlin.time.Duration.Companion.days
import kotlin.time.Duration.Companion.minutes

class FeedException(message: String) : Exception(message)

class SyncEngine(
    private val db: CarteroDatabase,
    private val fetcher: FeedFetcher,
    private val settings: SettingsRepository,
    private val notifier: Notifier,
    private val content: ContentLoader,
) {
    private val mutex = Mutex()
    private val running = MutableStateFlow(false)
    val syncing: StateFlow<Boolean> = running.asStateFlow()

    suspend fun syncIfStale() {
        val last = db.feeds().lastSynced() ?: return
        if (nowMillis() - last > STALE_AFTER) syncAll()
    }

    suspend fun syncAll() {
        if (mutex.isLocked) return
        mutex.withLock {
            running.value = true
            try {
                val cutoff = cutoff()
                val permits = Semaphore(PARALLEL_FEEDS)
                val fresh = coroutineScope {
                    db.feeds().all()
                        .map { feed -> async { permits.withPermit { sync(feed, cutoff) } } }
                        .awaitAll()
                        .flatten()
                }
                db.articles().prune(cutoff)
                applyRules(fresh)
            } finally {
                running.value = false
            }
        }
    }

    suspend fun subscribe(input: String): Long {
        val url = normalize(input)
        db.feeds().byUrl(url)?.let { return it.id }

        var result = fetch(url)
        if (result is FetchResult.Page) {
            val discovered = result.feedUrls.firstOrNull() ?: throw FeedException("No feed found at this address")
            db.feeds().byUrl(discovered)?.let { return it.id }
            result = fetch(discovered)
        }
        val fetched = result as? FetchResult.Feed ?: throw FeedException("No feed found at this address")
        db.feeds().byUrl(fetched.url)?.let { return it.id }

        val feed = FeedEntity(
            url = fetched.url,
            title = Html.text(fetched.feed.title).ifEmpty { Html.webUrl(fetched.url)?.host ?: fetched.url },
            siteUrl = fetched.feed.siteUrl,
            etag = fetched.etag,
            lastModified = fetched.lastModified,
            syncedAt = nowMillis(),
        )
        val id = db.feeds().insert(feed)
        ingest(feed.copy(id = id), fetched, cutoff())
        return id
    }

    private suspend fun fetch(url: String): FetchResult =
        try {
            fetcher.fetch(url)
        } catch (e: IOException) {
            throw FeedException(e.message ?: "Couldn't reach $url")
        }

    private suspend fun sync(feed: FeedEntity, cutoff: Long): List<ArticleEntity> {
        val now = nowMillis()
        return try {
            when (val result = fetcher.fetch(feed.url, feed.etag, feed.lastModified)) {
                FetchResult.NotModified -> {
                    db.feeds().update(feed.copy(syncedAt = now, error = null))
                    emptyList()
                }
                is FetchResult.Page -> {
                    db.feeds().update(feed.copy(syncedAt = now, error = "Not a feed"))
                    emptyList()
                }
                is FetchResult.Feed -> {
                    db.feeds().update(feed.copy(etag = result.etag, lastModified = result.lastModified, syncedAt = now, error = null))
                    ingest(feed, result, cutoff)
                }
            }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            db.feeds().update(feed.copy(error = if (e is IOException) "Couldn't reach feed" else "Couldn't read feed"))
            emptyList()
        }
    }

    private suspend fun ingest(feed: FeedEntity, fetched: FetchResult.Feed, cutoff: Long): List<ArticleEntity> {
        val now = nowMillis()
        val articles = withContext(Dispatchers.Default) {
            fetched.feed.items
                .mapNotNull { it.toArticle(feed, fetched.url, now) }
                .filter { it.publishedAt >= cutoff }
                .distinctBy { it.guid }
        }
        if (articles.isEmpty()) return emptyList()
        val ids = db.articles().store(articles)
        return articles.zip(ids) { article, id -> article.copy(id = id) }.filter { it.id != -1L }
    }

    private suspend fun applyRules(fresh: List<ArticleEntity>) {
        if (fresh.isEmpty()) return
        val matchers = db.rules().enabled().map(::RuleMatcher)
        if (matchers.isEmpty()) return
        val (savers, notifiers) = matchers.partition { it.rule.action == RuleAction.Save }
        val now = nowMillis()

        val recent = now - NOTIFY_WINDOW
        val notified = HashSet<Long>()
        for (matcher in notifiers) {
            val matches = fresh.filter { it.publishedAt >= recent && it.id !in notified && matcher.matches(it) }
            notified += matches.map { it.id }
            notifier.post(matcher.rule, matches)
        }

        val toSave = fresh.filter { article -> savers.any { it.matches(article) } }
        if (toSave.isNotEmpty()) {
            db.articles().save(toSave.map { it.id }, now)
            content.prefetch(toSave)
        }
    }

    private suspend fun cutoff(): Long =
        nowMillis() - settings.current().retentionDays.days.inWholeMilliseconds

    private fun normalize(input: String): String {
        val trimmed = input.trim()
        val withScheme = if ("://" in trimmed) trimmed else "https://$trimmed"
        return Html.webUrl(withScheme)?.toString() ?: throw FeedException("Enter a valid address")
    }

    private companion object {
        const val PARALLEL_FEEDS = 4
        val STALE_AFTER = 15.minutes.inWholeMilliseconds
        val NOTIFY_WINDOW = 2.days.inWholeMilliseconds
    }
}

private fun ParsedItem.toArticle(feed: FeedEntity, baseUrl: String, now: Long): ArticleEntity? {
    val link = Html.resolve(baseUrl, url)
    val body = content ?: summary
    val title = Html.text(title).ifEmpty { Html.text(body, TITLE_FALLBACK_CHARS) }.ifEmpty { return null }
    return ArticleEntity(
        feedId = feed.id,
        guid = guid ?: link ?: "$title|$publishedAt",
        url = link,
        title = title,
        summary = Html.text(summary ?: content, SUMMARY_CHARS),
        content = body,
        imageUrl = Html.resolve(link ?: baseUrl, imageUrl) ?: Html.firstImage(body, link ?: baseUrl),
        author = author?.let(Html::text)?.ifEmpty { null },
        source = source?.trim()?.ifEmpty { null } ?: feed.title,
        topic = topic?.trim()?.ifEmpty { null },
        publishedAt = minOf(publishedAt ?: now, now),
        readingMinutes = Html.readingMinutes(content),
    )
}

private const val SUMMARY_CHARS = 280
private const val TITLE_FALLBACK_CHARS = 90
