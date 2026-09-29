package app.cartero.data.db

import androidx.paging.PagingSource
import androidx.room3.Dao
import androidx.room3.Delete
import androidx.room3.Insert
import androidx.room3.OnConflictStrategy
import androidx.room3.Query
import androidx.room3.Transaction
import androidx.room3.Update
import kotlinx.coroutines.flow.Flow

private const val ROW = "id, title, summary, imageUrl, source, topic, publishedAt, readingMinutes, isRead, savedAt"

private const val FILTER = """
    (:anyTopic OR topic IN (:topics))
    AND (:anySource OR source IN (:sources))
    AND (:unreadOnly = 0 OR isRead = 0)
"""

@Dao
interface ArticleDao {
    @Query("SELECT $ROW FROM articles WHERE $FILTER ORDER BY publishedAt DESC, id DESC")
    fun feed(
        topics: List<String>,
        anyTopic: Boolean,
        sources: List<String>,
        anySource: Boolean,
        unreadOnly: Boolean,
    ): PagingSource<Int, ArticleRow>

    @Query("SELECT $ROW FROM articles WHERE savedAt IS NOT NULL ORDER BY savedAt DESC")
    fun saved(): PagingSource<Int, ArticleRow>

    @Query("SELECT * FROM articles WHERE id = :id")
    fun observe(id: Long): Flow<ArticleEntity?>

    @Query("SELECT * FROM articles WHERE id = :id")
    suspend fun get(id: Long): ArticleEntity?

    @Query("SELECT topic AS value, COUNT(*) AS count FROM articles WHERE topic IS NOT NULL GROUP BY topic ORDER BY count DESC, topic")
    fun topics(): Flow<List<Facet>>

    @Query("SELECT source AS value, COUNT(*) AS count FROM articles GROUP BY source ORDER BY count DESC, source")
    fun sources(): Flow<List<Facet>>

    @Query(
        """
        SELECT * FROM articles WHERE savedAt IS NULL AND (
            topic = :value COLLATE NOCASE OR source = :value COLLATE NOCASE
            OR title LIKE '%' || :value || '%' OR summary LIKE '%' || :value || '%'
        )
        """,
    )
    suspend fun unsavedMatching(value: String): List<ArticleEntity>

    @Query("SELECT COUNT(*) FROM articles WHERE isRead = 0")
    fun unreadCount(): Flow<Int>

    @Insert(onConflict = OnConflictStrategy.IGNORE)
    suspend fun insert(articles: List<ArticleEntity>): List<Long>

    @Query(
        """
        UPDATE articles SET
            title = :title,
            summary = :summary,
            topic = :topic,
            source = :source,
            imageUrl = COALESCE(:imageUrl, imageUrl),
            publishedAt = COALESCE(:publishedAt, publishedAt),
            content = CASE WHEN fullText THEN content ELSE :content END,
            readingMinutes = CASE WHEN fullText THEN readingMinutes ELSE :readingMinutes END
        WHERE feedId = :feedId AND guid = :guid AND (
            title IS NOT :title OR summary IS NOT :summary OR topic IS NOT :topic OR source IS NOT :source
            OR (:imageUrl IS NOT NULL AND imageUrl IS NOT :imageUrl)
            OR (:publishedAt IS NOT NULL AND publishedAt IS NOT :publishedAt)
            OR (fullText = 0 AND content IS NOT :content)
        )
        """,
    )
    suspend fun refresh(
        feedId: Long,
        guid: String,
        title: String,
        summary: String,
        content: String?,
        imageUrl: String?,
        publishedAt: Long?,
        source: String,
        topic: String?,
        readingMinutes: Int,
    )

    @Transaction
    suspend fun store(articles: List<ArticleEntity>, now: Long): List<Long> {
        val ids = insert(articles)
        articles.forEachIndexed { index, article ->
            if (ids[index] == -1L) {
                refresh(
                    feedId = article.feedId,
                    guid = article.guid,
                    title = article.title,
                    summary = article.summary,
                    content = article.content,
                    imageUrl = article.imageUrl,
                    publishedAt = article.publishedAt.takeIf { it < now },
                    source = article.source,
                    topic = article.topic,
                    readingMinutes = article.readingMinutes,
                )
            }
        }
        return ids
    }

    @Query("UPDATE articles SET isRead = :read WHERE id = :id")
    suspend fun setRead(id: Long, read: Boolean)

    @Query("UPDATE articles SET isRead = 1 WHERE isRead = 0 AND $FILTER")
    suspend fun markAllRead(
        topics: List<String>,
        anyTopic: Boolean,
        sources: List<String>,
        anySource: Boolean,
        unreadOnly: Boolean,
    )

    @Query("UPDATE articles SET savedAt = CASE WHEN savedAt IS NULL THEN :now ELSE NULL END WHERE id = :id")
    suspend fun toggleSaved(id: Long, now: Long)

    @Query("UPDATE articles SET savedAt = :now WHERE id IN (:ids) AND savedAt IS NULL")
    suspend fun save(ids: List<Long>, now: Long)

    @Query("UPDATE articles SET content = :content, fullText = 1, readingMinutes = :minutes, imageUrl = COALESCE(imageUrl, :image) WHERE id = :id")
    suspend fun setFullText(id: Long, content: String, minutes: Int, image: String?)

    @Query("DELETE FROM articles WHERE savedAt IS NULL AND publishedAt < :before")
    suspend fun prune(before: Long)
}

@Dao
interface FeedDao {
    @Query(
        """
        SELECT f.id, f.url, f.title, f.siteUrl, f.syncedAt, f.error,
            (SELECT COUNT(*) FROM articles a WHERE a.feedId = f.id AND a.isRead = 0) AS unread
        FROM feeds f ORDER BY f.title COLLATE NOCASE
        """,
    )
    fun observe(): Flow<List<FeedWithStats>>

    @Query("SELECT * FROM feeds")
    suspend fun all(): List<FeedEntity>

    @Query("SELECT * FROM feeds WHERE id = :id")
    suspend fun get(id: Long): FeedEntity?

    @Query("SELECT * FROM feeds WHERE url = :url")
    suspend fun byUrl(url: String): FeedEntity?

    @Query("SELECT COUNT(*) FROM feeds")
    fun count(): Flow<Int>

    @Query("SELECT MAX(syncedAt) FROM feeds")
    suspend fun lastSynced(): Long?

    @Insert
    suspend fun insert(feed: FeedEntity): Long

    @Insert(onConflict = OnConflictStrategy.IGNORE)
    suspend fun insertAll(feeds: List<FeedEntity>): List<Long>

    @Update
    suspend fun update(feed: FeedEntity)

    @Query("UPDATE feeds SET title = :title WHERE id = :id")
    suspend fun rename(id: Long, title: String)

    @Query("DELETE FROM feeds WHERE id = :id AND url != :keep")
    suspend fun delete(id: Long, keep: String)
}

@Dao
interface RuleDao {
    @Query("SELECT * FROM rules ORDER BY action, field, value COLLATE NOCASE")
    fun observe(): Flow<List<RuleEntity>>

    @Query("SELECT * FROM rules WHERE enabled = 1")
    suspend fun enabled(): List<RuleEntity>

    @Insert(onConflict = OnConflictStrategy.IGNORE)
    suspend fun insert(rule: RuleEntity): Long

    @Query("UPDATE rules SET enabled = :enabled WHERE id = :id")
    suspend fun setEnabled(id: Long, enabled: Boolean)

    @Delete
    suspend fun delete(rule: RuleEntity)
}
