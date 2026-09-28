package app.cartero.ui.reader

import androidx.compose.material3.ColorScheme
import androidx.compose.runtime.Immutable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import app.cartero.data.db.ArticleEntity
import app.cartero.data.feed.Html
import app.cartero.ui.components.fullDate
import app.cartero.ui.components.localDate
import com.fleeksoft.ksoup.Ksoup
import com.fleeksoft.ksoup.safety.Safelist

const val FULL_TEXT_ACTION = "cartero-action:full-text"
expect val FONT_ORIGIN: String

@Immutable
data class ReaderPalette(
    val paper: String,
    val ink: String,
    val inkSoft: String,
    val inkFaint: String,
    val rule: String,
    val accent: String,
    val codeBackground: String,
    val fieldBackground: String,
)

fun ColorScheme.readerPalette() = ReaderPalette(
    paper = surface.css(),
    ink = onSurface.css(),
    inkSoft = onSurfaceVariant.css(),
    inkFaint = outline.css(),
    rule = outlineVariant.css(),
    accent = primary.css(),
    codeBackground = surfaceContainerHighest.css(),
    fieldBackground = surfaceContainer.css(),
)

object ArticleHtml {
    private val markup = Regex(
        "<(p|div|br|h[1-6]|ul|ol|li|img|a|figure|blockquote|pre|table|span|strong|em|article|section)\\b",
        RegexOption.IGNORE_CASE,
    )
    private val paragraphBreak = Regex("\\n\\s*\\n")
    private val safelist = Safelist.relaxed()
        .addTags("figure", "figcaption", "picture", "source", "hr", "del", "ins", "mark", "s", "abbr", "time")
        .addAttributes("img", "srcset", "sizes")
        .addAttributes("source", "srcset", "sizes", "type", "media")
        .addAttributes("code", "class")
        .addAttributes("pre", "class")

    fun build(article: ArticleEntity, palette: ReaderPalette, needsFullText: Boolean): String {
        val base = article.url.orEmpty()
        val clean = Ksoup.clean(bodyHtml = asHtml(article.content.orEmpty()), safelist = safelist, baseUri = base)
        val body = Ksoup.parseBodyFragment(clean, base).body()
        val images = body.select("img")
        images.forEach { img ->
            val width = img.attr("width").toIntOrNull()
            if (width != null && width < MIN_IMAGE_WIDTH) img.remove() else img.attr("loading", "lazy")
        }
        val heroInBody = images.any { it.attr("src") == article.imageUrl }

        return buildString {
            append("<!DOCTYPE html><html><head><meta charset=\"utf-8\">")
            append("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
            if (base.isNotEmpty()) append("<base href=\"").append(base.escape()).append("\">")
            append("<style>").append(palette.variables()).append(CSS).append("</style></head>")
            append("<body><main class=\"reader\">")

            article.topic?.let { append("<span class=\"kicker\">").append(it.escape()).append("</span>") }
            append("<h1 class=\"title\">").append(article.title.escape()).append("</h1>")
            append("<div class=\"byline\">")
            append("<span class=\"source\">").append(article.source.escape()).append("</span>")
            article.author?.takeIf { it != article.source }?.let { byline(it, "author") }
            byline(article.publishedAt.localDate().fullDate())
            if (article.readingMinutes > 0) byline("${article.readingMinutes} min read")
            append("</div>")

            if (article.imageUrl != null && !heroInBody) {
                append("<figure class=\"hero\"><img src=\"").append(article.imageUrl.escape()).append("\" alt=\"\"></figure>")
            }

            append("<article class=\"article-body\">").append(body.html()).append("</article>")

            append("<footer class=\"footer\">")
            if (needsFullText) append("<a href=\"$FULL_TEXT_ACTION\">Load full article</a>")
            article.url?.let { append("<a href=\"").append(it.escape()).append("\">Read on ").append(article.source.escape()).append(" ↗</a>") }
            append("</footer></main></body></html>")
        }
    }

    private fun StringBuilder.byline(text: String, cls: String? = null) {
        append("<span class=\"dot\">·</span><span")
        if (cls != null) append(" class=\"").append(cls).append('"')
        append('>').append(text.escape()).append("</span>")
    }

    private fun asHtml(content: String): String {
        if (markup.containsMatchIn(content)) return content
        return content.trim().split(paragraphBreak).joinToString("") { paragraph ->
            "<p>" + Html.escape(paragraph.trim()).replace('\n', ' ') + "</p>"
        }
    }

    private fun String.escape(): String = Html.escape(this)

    private fun ReaderPalette.variables() =
        ":root{--paper:$paper;--ink:$ink;--ink-soft:$inkSoft;--ink-faint:$inkFaint;--rule:$rule;" +
            "--accent:$accent;--code-bg:$codeBackground;--field-bg:$fieldBackground}"

    private const val MIN_IMAGE_WIDTH = 48

    private val CSS = """
        @font-face{font-family:Manrope;src:url(${FONT_ORIGIN}manrope_regular.ttf);font-weight:400}
        @font-face{font-family:Manrope;src:url(${FONT_ORIGIN}manrope_medium.ttf);font-weight:500}
        @font-face{font-family:Manrope;src:url(${FONT_ORIGIN}manrope_semibold.ttf);font-weight:600}
        @font-face{font-family:Manrope;src:url(${FONT_ORIGIN}manrope_bold.ttf);font-weight:700}
        @font-face{font-family:Manrope;src:url(${FONT_ORIGIN}manrope_extrabold.ttf);font-weight:800}
        @font-face{font-family:"Source Serif 4";src:url(${FONT_ORIGIN}source_serif_regular.ttf);font-weight:400;font-style:normal}
        @font-face{font-family:"Source Serif 4";src:url(${FONT_ORIGIN}source_serif_bold.ttf);font-weight:700;font-style:normal}
        @font-face{font-family:"Source Serif 4";src:url(${FONT_ORIGIN}source_serif_italic.ttf);font-weight:400;font-style:italic}
        *{margin:0;padding:0;box-sizing:border-box}
        html{-webkit-text-size-adjust:100%}
        body{background:var(--paper);color:var(--ink);font-family:Manrope,sans-serif;-webkit-font-smoothing:antialiased;text-rendering:optimizeLegibility}
        a{color:var(--accent)}
        .reader{max-width:720px;margin:0 auto;padding:8px 20px 160px}
        .kicker{display:block;text-transform:uppercase;letter-spacing:.13em;font-size:.7rem;font-weight:700;color:var(--accent);margin-bottom:12px}
        h1.title{font-weight:800;font-size:1.72rem;line-height:1.14;letter-spacing:-.02em;color:var(--ink);margin-bottom:14px;overflow-wrap:anywhere;hyphens:auto}
        .byline{display:flex;flex-wrap:wrap;gap:4px 8px;align-items:center;color:var(--ink-faint);font-size:.84rem;padding-bottom:16px;margin-bottom:22px;border-bottom:1px solid var(--rule)}
        .byline .source{color:var(--ink);font-weight:700}
        .byline .author{color:var(--ink-soft);font-weight:500}
        .hero{margin:0 0 24px}
        .hero img{display:block;width:100%;max-height:420px;object-fit:cover;border-radius:16px;background:var(--field-bg)}
        .article-body{font-family:"Source Serif 4",serif;font-size:1.08rem;line-height:1.74;color:var(--ink);overflow-wrap:anywhere;hyphens:auto}
        .article-body p{margin:0 0 1.1em}
        .article-body h1,.article-body h2,.article-body h3,.article-body h4,.article-body h5,.article-body h6{font-family:Manrope,sans-serif;font-weight:700;line-height:1.25;color:var(--ink);margin:1.6em 0 .55em;letter-spacing:-.01em}
        .article-body h1{font-size:1.42rem}.article-body h2{font-size:1.28rem}.article-body h3{font-size:1.14rem}
        .article-body h4,.article-body h5,.article-body h6{font-size:1.02rem;color:var(--ink-soft)}
        .article-body a{text-decoration:underline;text-decoration-thickness:1px;text-underline-offset:3px}
        .article-body img{display:block;max-width:100%;height:auto;border-radius:12px;margin:1.4em auto;background:var(--field-bg)}
        .article-body figure{margin:1.4em 0}
        .article-body figure img{margin:0 auto}
        .article-body figcaption{font-family:Manrope,sans-serif;font-size:.8rem;color:var(--ink-faint);text-align:center;margin-top:8px}
        .article-body blockquote{margin:1.4em 0;padding:.2em 0 .2em 1.1em;border-left:3px solid var(--accent);color:var(--ink-soft);font-style:italic}
        .article-body ul,.article-body ol{margin:0 0 1.1em;padding-left:1.4em}
        .article-body li{margin-bottom:.4em}
        .article-body li::marker{color:var(--accent)}
        .article-body code{font-family:ui-monospace,"Roboto Mono",monospace;font-size:.85em;background:var(--code-bg);border-radius:6px;padding:.12em .36em}
        .article-body pre{background:var(--code-bg);border-radius:12px;padding:14px 16px;margin:1.4em 0;overflow-x:auto;line-height:1.5}
        .article-body pre code{background:none;padding:0;font-size:.84rem}
        .article-body hr{border:none;border-top:1px solid var(--rule);margin:2em 0}
        .article-body table{display:block;overflow-x:auto;border-collapse:collapse;margin:1.4em 0;font-size:.9rem}
        .article-body th,.article-body td{border:1px solid var(--rule);padding:8px 10px;text-align:left}
        .article-body th{font-family:Manrope,sans-serif;background:var(--field-bg)}
        .footer{display:flex;flex-wrap:wrap;gap:16px;margin-top:36px;padding-top:18px;border-top:1px solid var(--rule)}
        .footer a{font-size:.74rem;font-weight:700;text-transform:uppercase;letter-spacing:.1em;text-decoration:none}
    """.trimIndent().replace("\n", "")
}

private fun Color.css(): String = "#" + (toArgb() and 0xFFFFFF).toString(16).padStart(6, '0')

val READER_FONTS = setOf(
    "manrope_regular.ttf",
    "manrope_medium.ttf",
    "manrope_semibold.ttf",
    "manrope_bold.ttf",
    "manrope_extrabold.ttf",
    "source_serif_regular.ttf",
    "source_serif_bold.ttf",
    "source_serif_italic.ttf",
)
