package app.cartero.data.db

import androidx.room3.Entity
import androidx.room3.ForeignKey
import androidx.room3.Index
import androidx.room3.PrimaryKey

@Entity(tableName = "feeds", indices = [Index(value = ["url"], unique = true)])
data class FeedEntity(
    @PrimaryKey(autoGenerate = true) val id: Long = 0,
    val url: String,
    val title: String,
    val siteUrl: String? = null,
    val etag: String? = null,
    val lastModified: String? = null,
    val syncedAt: Long = 0,
    val error: String? = null,
)

@Entity(
    tableName = "articles",
    foreignKeys = [
        ForeignKey(
            entity = FeedEntity::class,
            parentColumns = ["id"],
            childColumns = ["feedId"],
            onDelete = ForeignKey.CASCADE,
        ),
    ],
    indices = [
        Index(value = ["feedId", "guid"], unique = true),
        Index(value = ["publishedAt"]),
        Index(value = ["savedAt"]),
        Index(value = ["topic"]),
        Index(value = ["source"]),
    ],
)
data class ArticleEntity(
    @PrimaryKey(autoGenerate = true) val id: Long = 0,
    val feedId: Long,
    val guid: String,
    val url: String?,
    val title: String,
    val summary: String,
    val content: String?,
    val imageUrl: String?,
    val author: String?,
    val source: String,
    val topic: String?,
    val publishedAt: Long,
    val readingMinutes: Int,
    val isRead: Boolean = false,
    val savedAt: Long? = null,
    val fullText: Boolean = false,
)

enum class RuleAction { Notify, Save }

enum class RuleField { Topic, Source, Keyword }

@Entity(tableName = "rules", indices = [Index(value = ["action", "field", "value"], unique = true)])
data class RuleEntity(
    @PrimaryKey(autoGenerate = true) val id: Long = 0,
    val action: RuleAction,
    val field: RuleField,
    val value: String,
    val enabled: Boolean = true,
)

data class ArticleRow(
    val id: Long,
    val title: String,
    val summary: String,
    val imageUrl: String?,
    val source: String,
    val topic: String?,
    val publishedAt: Long,
    val readingMinutes: Int,
    val isRead: Boolean,
    val savedAt: Long?,
)

data class Facet(val value: String, val count: Int)

data class FeedWithStats(
    val id: Long,
    val url: String,
    val title: String,
    val siteUrl: String?,
    val syncedAt: Long,
    val error: String?,
    val unread: Int,
)
