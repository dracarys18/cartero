package app.cartero.data.db

import androidx.room3.ConstructedBy
import androidx.room3.DaoReturnTypeConverters
import androidx.room3.Database
import androidx.room3.RoomDatabase
import androidx.room3.RoomDatabaseConstructor
import androidx.room3.paging.PagingSourceDaoReturnTypeConverter
import androidx.sqlite.SQLiteConnection
import androidx.sqlite.execSQL
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.IO

@Database(
    entities = [FeedEntity::class, ArticleEntity::class, RuleEntity::class],
    version = 1,
    exportSchema = false,
)
@DaoReturnTypeConverters(PagingSourceDaoReturnTypeConverter::class)
@ConstructedBy(CarteroDatabaseConstructor::class)
abstract class CarteroDatabase : RoomDatabase() {
    abstract fun articles(): ArticleDao
    abstract fun feeds(): FeedDao
    abstract fun rules(): RuleDao

    companion object {
        const val FILE_NAME = "cartero.db"
        const val DEFAULT_FEED_URL = "https://news.karthihegde.dev/feed.rss"
        private const val DEFAULT_SITE_URL = "https://news.karthihegde.dev"

        fun build(builder: Builder<CarteroDatabase>): CarteroDatabase =
            builder
                .addCallback(DefaultFeed)
                .setQueryCoroutineContext(Dispatchers.IO)
                .build()
    }

    private object DefaultFeed : Callback() {
        override suspend fun onOpen(connection: SQLiteConnection) {
            connection.execSQL(
                "INSERT OR IGNORE INTO feeds (url, title, siteUrl, syncedAt) " +
                    "VALUES ('$DEFAULT_FEED_URL', 'Cartero', '$DEFAULT_SITE_URL', 0)",
            )
        }
    }
}

@Suppress("KotlinNoActualForExpect")
expect object CarteroDatabaseConstructor : RoomDatabaseConstructor<CarteroDatabase> {
    override fun initialize(): CarteroDatabase
}
