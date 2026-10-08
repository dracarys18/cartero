package app.cartero.data.reader

import com.fleeksoft.ksoup.nodes.Element
import com.fleeksoft.ksoup.nodes.TextNode

object TexMath {
    private val delimiter = Regex("""\$\$|\\\(|\\\[""")

    fun present(text: String): Boolean = delimiter.containsMatchIn(text)

    fun normalize(root: Element) {
        for (el in root.select(".mwe-math-element")) {
            val tex = el.selectFirst("math")?.tex() ?: el.selectFirst("img[alt]")?.attr("alt")
            el.replaceWithTex(tex, display = el.selectFirst("[class*=-display]") != null)
        }
        for (el in root.select(".katex-display")) el.replaceWithTex(el.selectFirst("math")?.tex(), display = true)
        for (el in root.select(".katex")) el.replaceWithTex(el.selectFirst("math")?.tex(), display = false)
        for (el in root.select("script[type^=\"math/tex\"]")) {
            el.replaceWithTex(el.data(), display = el.attr("type").contains("mode=display"))
        }
        for (el in root.select("math")) el.replaceWithTex(el.tex(), display = el.attr("display") == "block")
    }

    private fun Element.tex(): String? =
        selectFirst("annotation[encoding=\"application/x-tex\"]")?.wholeText()?.ifBlank { null }
            ?: attr("alttext").ifBlank { null }

    private fun Element.replaceWithTex(tex: String?, display: Boolean) {
        if (tex.isNullOrBlank() || parent() == null) return
        replaceWith(TextNode(if (display) "\\[${tex.trim()}\\]" else "\\(${tex.trim()}\\)"))
    }
}
