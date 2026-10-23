package app.cartero.data.db

import androidx.room3.RoomRawQuery

/**
 * The front page search, run against the local database instead of the web's Postgres search.
 * Words must all match; "quoted phrases" stay together; a leading minus excludes a word; OR
 * separates alternatives. Title hits rank above summary hits.
 *
 * The web runs this on Postgres, which also stems words and folds case outside ASCII. SQLite's
 * LIKE does neither.
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

        fun parse(text: String): SearchQuery? {
            val terms = parseTerms(text) ?: return null
            val query = SearchSqlBuilder(terms).build()
            return SearchQuery(query.sql, query.args)
        }
    }
}

/** Groups of words joined by AND, with OR between groups, plus words excluded by a leading minus. */
private data class SearchTerms(
    val groups: List<List<String>>,
    val excluded: List<String>,
)

private data class SearchWord(
    val text: String,
    val quoted: Boolean,
    val negated: Boolean,
) {
    val isOr: Boolean get() = !quoted && !negated && text.equals("or", ignoreCase = true)
}

/** Matches a word, "a quoted phrase", -word or -"a quoted phrase". */
private val WORD = Regex("""(?<negated>-)?(?:"(?<phrase>[^"]*)"?|(?<word>\S+))""")

private fun parseTerms(text: String): SearchTerms? {
    val words = WORD.findAll(text).map { match ->
        val phrase = match.groups["phrase"]
        SearchWord(
            text = phrase?.value ?: match.groups["word"]!!.value,
            quoted = phrase != null,
            negated = match.groups["negated"] != null,
        )
    }.filter { it.text.isNotBlank() }.toList()

    val groups = mutableListOf<MutableList<String>>()
    val excluded = mutableListOf<String>()
    var current = mutableListOf<String>()

    for (word in words) {
        when {
            word.isOr -> if (current.isNotEmpty()) {
                groups += current
                current = mutableListOf()
            }
            word.negated -> excluded += word.text
            else -> current += word.text
        }
    }
    if (current.isNotEmpty()) groups += current

    val limitedGroups = groups
        .map { group -> group.distinctBy { it.lowercase() }.take(MAX_TERMS_PER_GROUP) }
        .filter { it.isNotEmpty() }
        .take(MAX_GROUPS)
    if (limitedGroups.isEmpty()) return null

    return SearchTerms(
        groups = limitedGroups,
        excluded = excluded.distinctBy { it.lowercase() }.take(MAX_EXCLUDED_WORDS),
    )
}

private class SearchSqlBuilder(private val terms: SearchTerms) {
    private val sql = StringBuilder()
    private val args = mutableListOf<String>()

    fun build(): QueryParts {
        sql.append("SELECT $ARTICLE_ROW FROM articles WHERE ")
        appendMatches()
        appendExcluded()
        sql.append(" ORDER BY ")
        appendRanking()
        sql.append(" DESC, publishedAt DESC, id DESC LIMIT ${SearchQuery.LIMIT}")
        return QueryParts(sql.toString(), args)
    }

    private fun appendMatches() {
        sql.append("(")
        terms.groups.forEachIndexed { groupIndex, group ->
            if (groupIndex > 0) sql.append(" OR ")
            sql.append("(")
            group.forEachIndexed { index, term ->
                if (index > 0) sql.append(" AND ")
                appendTextMatch(term)
            }
            sql.append(")")
        }
        sql.append(")")
    }

    private fun appendExcluded() {
        terms.excluded.forEach { term ->
            sql.append(" AND NOT ")
            appendTextMatch(term)
        }
    }

    private fun appendRanking() {
        terms.groups.flatten().forEachIndexed { index, term ->
            if (index > 0) sql.append(" + ")
            appendTitleScore(term)
        }
    }

    private fun appendTextMatch(term: String) {
        sql.append(TEXT_MATCH)
        args += likePattern(term) // title
        args += likePattern(term) // summary
    }

    private fun appendTitleScore(term: String) {
        sql.append(TITLE_SCORE)
        args += likePattern(term) // title
        args += likePattern(term) // summary
    }
}

private data class QueryParts(val sql: String, val args: List<String>)

private const val MAX_TERMS_PER_GROUP = 12
private const val MAX_GROUPS = 4
private const val MAX_EXCLUDED_WORDS = 12

private const val TEXT_MATCH = """(title LIKE ? ESCAPE '\' OR summary LIKE ? ESCAPE '\')"""

private const val TITLE_SCORE =
    """(CASE WHEN title LIKE ? ESCAPE '\' THEN 3 ELSE 0 END""" +
        """ + CASE WHEN summary LIKE ? ESCAPE '\' THEN 1 ELSE 0 END)"""

/** SQLite reads % and _ as wildcards, so escape them before wrapping the word in %…%. */
private fun likePattern(term: String): String =
    "%" + term.replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_") + "%"
