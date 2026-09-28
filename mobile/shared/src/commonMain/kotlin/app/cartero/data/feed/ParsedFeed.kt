package app.cartero.data.feed

data class ParsedFeed(
    val title: String?,
    val siteUrl: String?,
    val items: List<ParsedItem>,
)

data class ParsedItem(
    val guid: String?,
    val url: String?,
    val title: String?,
    val summary: String?,
    val content: String?,
    val imageUrl: String?,
    val author: String?,
    val source: String?,
    val topic: String?,
    val publishedAt: Long?,
)

class ItemBuilder {
    var guid: String? = null
    var url: String? = null
    var title: String? = null
    var summary: String? = null
    var content: String? = null
    var imageUrl: String? = null
    var author: String? = null
    var source: String? = null
    var topic: String? = null
    var category: String? = null
    var publishedAt: Long? = null
    var updatedAt: Long? = null

    fun build() = ParsedItem(
        guid = guid,
        url = url,
        title = title,
        summary = summary,
        content = content,
        imageUrl = imageUrl,
        author = author,
        source = source,
        topic = topic ?: category,
        publishedAt = publishedAt ?: updatedAt,
    )
}
