package app.cartero.data.reader

import com.fleeksoft.ksoup.nodes.Document
import com.fleeksoft.ksoup.nodes.Element

data class Extracted(val html: String, val image: String?, val textLength: Int)

object Readability {
    private val unlikely = Regex(
        "banner|breadcrumb|combx|comment|community|cookie|disqus|extra|footer|gdpr|header|legends|menu|" +
            "newsletter|pager|pagination|popup|promo|related|remark|replies|rss|share|shoutbox|sidebar|" +
            "skyscraper|social|sponsor|subscribe|supplemental",
        RegexOption.IGNORE_CASE,
    )
    private val likely = Regex("and|article|body|column|content|main|shadow|post|entry|story", RegexOption.IGNORE_CASE)
    private const val JUNK = "script, style, noscript, iframe, form, nav, aside, footer, button, input, select, " +
        "textarea, svg, canvas, template, dialog, [hidden], [aria-hidden=true], [role=navigation], " +
        "[role=complementary], [role=dialog]"
    private val keptAttributes = setOf("href", "src", "alt", "title", "width", "height")
    private val lazySources = listOf("data-src", "data-lazy-src", "data-original", "data-url")

    fun parse(doc: Document): Extracted? {
        val image = doc.selectFirst("meta[property=og:image], meta[name=twitter:image]")
            ?.absUrl("content")?.ifEmpty { null }
        val title = doc.selectFirst("meta[property=og:title]")?.attr("content") ?: doc.title()

        val body = doc.body()
        body.select(JUNK).remove()
        dropUnlikely(body)
        fixLazyImages(body)

        val root = candidate(body) ?: return null
        clean(root, title)
        val length = root.text().length
        if (length < 200) return null
        return Extracted(root.html(), image, length)
    }

    private fun dropUnlikely(body: Element) {
        for (el in body.select("div, section, span, ul, table, header")) {
            val hint = el.className() + " " + el.id()
            if (unlikely.containsMatchIn(hint) && !likely.containsMatchIn(hint)) el.remove()
        }
    }

    private fun candidate(body: Element): Element? {
        body.select("article").singleOrNull()?.takeIf { it.text().length > 500 }?.let { return it }

        val scores = HashMap<Element, Double>()
        for (p in body.select("p, pre, td, blockquote")) {
            val text = p.text()
            if (text.length < 25) continue
            val score = 1.0 + text.count { it == ',' } + minOf(text.length / 100, 3)
            val parent = p.parent() ?: continue
            scores[parent] = (scores[parent] ?: 0.0) + score
            parent.parent()?.let { scores[it] = (scores[it] ?: 0.0) + score / 2 }
        }
        return scores.maxByOrNull { (el, score) -> (score + weight(el)) * (1 - linkDensity(el)) }?.key
    }

    private fun weight(el: Element): Double {
        val hint = el.className() + " " + el.id()
        var weight = 0.0
        if (unlikely.containsMatchIn(hint)) weight -= 25
        if (likely.containsMatchIn(hint)) weight += 25
        return weight
    }

    private fun linkDensity(el: Element): Double {
        val length = el.text().length
        if (length == 0) return 1.0
        val links = el.select("a").sumOf { it.text().length }
        return links.toDouble() / length
    }

    private fun fixLazyImages(body: Element) {
        for (img in body.select("img")) {
            val lazy = lazySources.firstNotNullOfOrNull { img.attr(it).ifEmpty { null } }
            val src = img.attr("src")
            when {
                lazy != null -> img.attr("src", lazy)
                src.isEmpty() || src.startsWith("data:") -> bestFromSrcset(img.attr("srcset"))?.let { img.attr("src", it) }
            }
        }
    }

    private fun bestFromSrcset(srcset: String): String? =
        srcset.split(',').map { it.trim().substringBefore(' ') }.lastOrNull { it.isNotEmpty() }

    private fun clean(root: Element, title: String) {
        val normalizedTitle = title.lowercase().trim()
        for (h1 in root.select("h1")) {
            val text = h1.text().lowercase().trim()
            if (text.isNotEmpty() && normalizedTitle.contains(text)) h1.remove() else h1.tagName("h2")
        }
        for (el in root.select("ul, ol, div, section, table")) {
            if (el.select("img, pre").isEmpty() && el.text().length < 200 && linkDensity(el) > 0.5) el.remove()
        }
        for (el in root.select("p, div, span, section")) {
            if (el.text().isBlank() && el.select("img, pre, video").isEmpty()) el.remove()
        }
        for (el in root.select("*")) {
            for (attr in el.attributes().asList()) {
                if (attr.key !in keptAttributes) el.removeAttr(attr.key)
            }
        }
        for (a in root.select("a[href]")) a.attr("href", a.absUrl("href"))
        for (img in root.select("img[src]")) img.attr("src", img.absUrl("src"))
    }
}
