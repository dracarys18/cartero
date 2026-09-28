package app.cartero.data.feed

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

object JsonFeedParser {
    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
        coerceInputValues = true
    }

    fun parse(text: String): ParsedFeed {
        val feed = json.decodeFromString<JsonFeed>(text)
        return ParsedFeed(
            title = feed.title,
            siteUrl = feed.homePageUrl,
            items = feed.items.map { it.toParsed() },
        )
    }

    private fun JsonItem.toParsed() = ParsedItem(
        guid = id,
        url = url ?: externalUrl,
        title = title,
        summary = summary,
        content = contentHtml ?: contentText?.let { "<p>${Html.escape(it)}</p>" },
        imageUrl = image ?: bannerImage,
        author = authors.firstOrNull()?.name ?: author?.name,
        source = cartero?.source,
        topic = tags.firstOrNull(),
        publishedAt = Dates.parse(datePublished) ?: Dates.parse(dateModified),
    )
}

@Serializable
private data class JsonFeed(
    val title: String? = null,
    @SerialName("home_page_url") val homePageUrl: String? = null,
    val items: List<JsonItem> = emptyList(),
)

@Serializable
private data class JsonItem(
    val id: String? = null,
    val url: String? = null,
    @SerialName("external_url") val externalUrl: String? = null,
    val title: String? = null,
    val summary: String? = null,
    @SerialName("content_html") val contentHtml: String? = null,
    @SerialName("content_text") val contentText: String? = null,
    val image: String? = null,
    @SerialName("banner_image") val bannerImage: String? = null,
    @SerialName("date_published") val datePublished: String? = null,
    @SerialName("date_modified") val dateModified: String? = null,
    val authors: List<JsonAuthor> = emptyList(),
    val author: JsonAuthor? = null,
    val tags: List<String> = emptyList(),
    @SerialName("_cartero") val cartero: CarteroExtension? = null,
)

@Serializable
private data class JsonAuthor(val name: String? = null)

@Serializable
private data class CarteroExtension(val source: String? = null)
