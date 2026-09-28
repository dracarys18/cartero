package app.cartero.data.feed

import com.fleeksoft.ksoup.Ksoup
import com.fleeksoft.ksoup.nodes.Element
import com.fleeksoft.ksoup.parser.Parser

const val TOPIC_DOMAIN = "topic"
const val SOURCE_DOMAIN = "source"

object XmlFeedParser {
    fun parse(xml: String): ParsedFeed {
        val root = Ksoup.parse(xml, parser = Parser.xmlParser()).children().firstOrNull()
            ?: return ParsedFeed(null, null, emptyList())
        val channel = root.children().firstOrNull { it.local() == "channel" } ?: root

        var title: String? = null
        var siteUrl: String? = null
        for (child in channel.children()) {
            when (child.local()) {
                "title" -> title = title ?: child.textValue()
                "link" -> siteUrl = siteUrl ?: child.channelLink()
            }
        }

        val entries = (channel.children() + if (channel === root) emptyList() else root.children())
            .filter { it.local() == "item" || it.local() == "entry" }
            .map { entry -> ItemBuilder().also { read(entry, it) }.build() }
        return ParsedFeed(title, siteUrl, entries)
    }

    private fun read(parent: Element, item: ItemBuilder) {
        for (child in parent.children()) {
            val media = child.prefix() == "media"
            when (child.local()) {
                "title" -> item.title = child.textValue()
                "link" -> link(child, item)
                "guid", "id" -> item.guid = child.textValue()
                "description" -> item.summary = child.textValue()
                "summary" -> item.summary = child.markup()
                "encoded" -> item.content = child.textValue()
                "content" -> if (media) mediaContent(child, item) else item.content = child.markup()
                "thumbnail" -> if (media) item.imageUrl = item.imageUrl ?: child.attr("url").ifEmpty { null }
                "group" -> if (media) read(child, item)
                "enclosure" -> if (child.attr("type").startsWith("image/")) item.imageUrl = item.imageUrl ?: child.attr("url").ifEmpty { null }
                "pubdate", "published", "issued" -> item.publishedAt = Dates.parse(child.textValue())
                "date", "updated", "modified" -> item.updatedAt = Dates.parse(child.textValue())
                "creator" -> item.author = child.textValue()
                "author" -> item.author = child.children().firstOrNull { it.local() == "name" }?.textValue() ?: child.textValue()
                "category" -> category(child, item)
                "source" -> if (child.children().isEmpty()) item.source = item.source ?: child.textValue()
            }
        }
    }

    private fun link(el: Element, item: ItemBuilder) {
        val href = el.attr("href")
        if (href.isEmpty()) {
            item.url = item.url ?: el.textValue()
            return
        }
        when (el.attr("rel").ifEmpty { "alternate" }) {
            "alternate" -> item.url = item.url ?: href
            "enclosure" -> if (el.attr("type").startsWith("image/")) item.imageUrl = item.imageUrl ?: href
        }
    }

    private fun mediaContent(el: Element, item: ItemBuilder) {
        val isImage = el.attr("medium") == "image" || el.attr("type").startsWith("image/")
        if (isImage) item.imageUrl = item.imageUrl ?: el.attr("url").ifEmpty { null }
        read(el, item)
    }

    private fun category(el: Element, item: ItemBuilder) {
        val value = el.attr("term").ifEmpty { el.textValue().orEmpty() }.trim()
        if (value.isEmpty()) return
        when (el.attr("domain").ifEmpty { el.attr("scheme") }) {
            TOPIC_DOMAIN -> item.topic = value
            SOURCE_DOMAIN -> item.source = value
            else -> item.category = item.category ?: value
        }
    }

    private fun Element.channelLink(): String? {
        val href = attr("href")
        if (href.isEmpty()) return textValue()
        return href.takeIf { attr("rel").let { it.isEmpty() || it == "alternate" } }
    }

    private fun Element.textValue(): String? = wholeText().trim().ifEmpty { null }

    private fun Element.markup(): String? = if (attr("type") == "xhtml") html().trim().ifEmpty { null } else textValue()

    private fun Element.local(): String = tagName().substringAfter(':').lowercase()

    private fun Element.prefix(): String = tagName().substringBefore(':', "").lowercase()
}
