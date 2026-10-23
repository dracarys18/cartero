package app.cartero.data.db

import androidx.room3.RoomRawQuery

/**
 * Mirrors the web front page search: terms match story titles and summaries, quoted phrases stay
 * together, a leading minus excludes a term, OR splits alternatives, and title hits rank above
 * summary hits. SQLite only ignores case for ASCII, the web's Postgres search stems words.
 */
class SearchQuery private constructor(
    val sql: String,
    val args: List<String>,
) {
    fun toRawQuery(): RoomRawQuery = RoomRawQuery(sql) { statement ->
        args.forEachIndexed { index, value -> statement.bindText(index + 1, value) }
    }

    companion object {
        const val LIMIT = 40
        private const val MATCH = """(title LIKE ? ESCAPE '\' OR summary LIKE ? ESCAPE '\')"""
        private const val SCORE =
            """(CASE WHEN title LIKE ? ESCAPE '\' THEN 3 ELSE 0 END""" +
                """ + CASE WHEN summary LIKE ? ESCAPE '\' THEN 1 ELSE 0 END)"""

        fun parse(text: String): SearchQuery? {
            val groups = mutableListOf<MutableList<String>>()
            val excluded = mutableListOf<String>()
            var current = mutableListOf<String>()
            var index = 0

            while (index < text.length) {
                while (index < text.length && text[index].isWhitespace()) index++
                if (index >= text.length) break

                var negated = false
                if (text[index] == '-') {
                    negated = true
                    index++
                }
                val quoted = index < text.length && text[index] == '"'
                if (quoted) index++
                val start = index
                while (index < text.length && if (quoted) text[index] != '"' else !text[index].isWhitespace()) index++
                val term = text.substring(start, index).trim()
                if (quoted && index < text.length) index++

                when {
                    term.isEmpty() -> Unit
                    !quoted && term.equals("or", ignoreCase = true) -> {
                        if (current.isNotEmpty()) {
                            groups += current
                            current = mutableListOf()
                        }
                    }
                    negated -> excluded += term
                    else -> current += term
                }
            }
            if (current.isNotEmpty()) groups += current

            val orGroups = groups
                .map { group -> group.distinctBy { it.lowercase() }.take(MAX_TERMS) }
                .filter { it.isNotEmpty() }
                .take(MAX_GROUPS)
            if (orGroups.isEmpty()) return null
            val excludedTerms = excluded.distinctBy { it.lowercase() }.take(MAX_TERMS)
            val scoreTerms = orGroups.flatten()

            val where = buildString {
                append('(')
                append(orGroups.joinToString(" OR ") { group -> "(" + group.joinToString(" AND ") { MATCH } + ")" })
                append(')')
                excludedTerms.forEach { append(" AND NOT ").append(MATCH) }
            }

            val args = buildList {
                for (term in orGroups.flatten() + excludedTerms + scoreTerms) {
                    add(likePattern(term))
                    add(likePattern(term))
                }
            }

            return SearchQuery(
                sql = "SELECT $ARTICLE_ROW FROM articles WHERE $where " +
                    "ORDER BY $SCORE DESC, publishedAt DESC, id DESC LIMIT $LIMIT",
                args = args,
            )
        }

        private fun likePattern(term: String): String =
            "%" + term.replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_") + "%"

        private const val MAX_TERMS = 12
        private const val MAX_GROUPS = 4
    }
}
