package app.cartero.data.sync

import app.cartero.data.db.ArticleDao
import app.cartero.data.db.ArticleEntity
import app.cartero.data.feed.Html
import app.cartero.data.reader.ArticleExtractor
import app.cartero.data.reader.Extracted
import coil3.ImageLoader
import coil3.PlatformContext
import coil3.request.CachePolicy
import coil3.request.ImageRequest

class ContentLoader(
    private val context: PlatformContext,
    private val dao: ArticleDao,
    private val extractor: ArticleExtractor,
    private val imageLoader: ImageLoader,
) {
    suspend fun fullText(article: ArticleEntity): Extracted? {
        val url = article.url ?: return null
        val extracted = extractor.extract(url) ?: return null
        dao.setFullText(article.id, extracted.html, Html.readingMinutes(extracted.html), extracted.image)
        return extracted
    }

    suspend fun prefetch(articles: List<ArticleEntity>) {
        for (article in articles) prefetch(article)
    }

    suspend fun prefetch(article: ArticleEntity) {
        val extracted = if (article.needsFullText()) fullText(article) else null
        val html = extracted?.html ?: article.content
        val images = listOfNotNull(article.imageUrl ?: extracted?.image) + Html.images(html, article.url).take(MAX_IMAGES)
        for (url in images) {
            val request = ImageRequest.Builder(context)
                .data(url)
                .memoryCachePolicy(CachePolicy.DISABLED)
                .build()
            imageLoader.execute(request)
        }
    }

    private companion object {
        const val MAX_IMAGES = 12
    }
}

private const val FULL_TEXT_MIN_CHARS = 1500

fun ArticleEntity.needsFullText(): Boolean =
    !fullText && url != null && Html.text(content, FULL_TEXT_MIN_CHARS).length < FULL_TEXT_MIN_CHARS
