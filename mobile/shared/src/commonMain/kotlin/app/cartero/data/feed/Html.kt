package app.cartero.data.feed

import com.fleeksoft.ksoup.parser.Parser
import io.ktor.http.URLBuilder
import io.ktor.http.URLProtocol
import io.ktor.http.Url
import io.ktor.http.parseUrl
import io.ktor.http.takeFrom

object Html {
    private val imgSrc = Regex("""<img\b[^>]*?\bsrc\s*=\s*["']([^"']+)["']""", RegexOption.IGNORE_CASE)
    private val skipped = setOf("script", "style", "noscript", "svg", "figcaption")

    fun text(html: String?, limit: Int = Int.MAX_VALUE): String {
        if (html.isNullOrEmpty()) return ""
        val out = StringBuilder(minOf(html.length, limit))
        var i = 0
        var skipUntil: String? = null
        var space = false
        while (i < html.length && out.length < limit) {
            val c = html[i]
            if (c == '<' && isTagStart(html.getOrNull(i + 1))) {
                val end = html.indexOf('>', i)
                if (end < 0) break
                val tag = tagName(html, i + 1, end)
                if (skipUntil != null) {
                    if (tag == "/$skipUntil") skipUntil = null
                } else if (tag in skipped) {
                    skipUntil = tag
                } else {
                    space = true
                }
                i = end + 1
                continue
            }
            if (skipUntil == null) {
                if (c.isWhitespace()) {
                    space = true
                } else {
                    if (space && out.isNotEmpty()) out.append(' ')
                    space = false
                    out.append(c)
                }
            }
            i++
        }
        return Parser.unescapeEntities(out.toString(), false).trim()
    }

    fun escape(text: String): String = buildString(text.length) {
        for (c in text) {
            when (c) {
                '<' -> append("&lt;")
                '>' -> append("&gt;")
                '&' -> append("&amp;")
                '"' -> append("&quot;")
                '\'' -> append("&#39;")
                else -> append(c)
            }
        }
    }

    fun firstImage(html: String?, base: String?): String? = images(html, base).firstOrNull()

    fun images(html: String?, base: String?): Sequence<String> {
        if (html.isNullOrEmpty()) return emptySequence()
        return imgSrc.findAll(html).mapNotNull { resolve(base, Parser.unescapeEntities(it.groupValues[1], true)) }
    }

    fun readingMinutes(html: String?): Int {
        if (html.isNullOrBlank()) return 0
        val words = text(html).count { it == ' ' } + 1
        return (words + 224) / 225
    }

    fun resolve(base: String?, url: String?): String? {
        val trimmed = url?.trim()
        if (trimmed.isNullOrEmpty() || trimmed.startsWith("data:")) return null
        val baseUrl = base?.let(::parseUrl)?.takeIf { it.isWeb() }
        if (trimmed.startsWith("//")) return parseUrl("${baseUrl?.protocol?.name ?: "https"}:$trimmed")?.toString()
        parseUrl(trimmed)?.takeIf { it.isWeb() }?.let { return it.toString() }
        if (baseUrl == null) return null
        return runCatching { URLBuilder(baseUrl).takeFrom(trimmed).build() }.getOrNull()?.takeIf { it.isWeb() }?.toString()
    }

    fun webUrl(input: String): Url? = parseUrl(input)?.takeIf { it.isWeb() && it.host.isNotEmpty() }

    private fun Url.isWeb() = protocol == URLProtocol.HTTPS || protocol == URLProtocol.HTTP

    private fun isTagStart(next: Char?): Boolean =
        next != null && (next.isLetter() || next == '/' || next == '!' || next == '?')

    private fun tagName(html: String, from: Int, to: Int): String {
        var end = from
        if (end < to && html[end] == '/') end++
        while (end < to && html[end].isLetterOrDigit()) end++
        return html.substring(from, end).lowercase()
    }
}
