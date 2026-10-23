package app.cartero.data.db

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

class SearchQueryTest {
    @Test
    fun blankQueryIsNotSearched() {
        assertNull(SearchQuery.parse("   "))
        assertNull(SearchQuery.parse("OR"))
    }

    @Test
    fun termsAreCombinedWithAnd() {
        val query = requireNotNull(SearchQuery.parse("react hooks"))
        assertTrue(query.sql.contains(" AND "))
        assertEquals(8, query.args.size)
    }

    @Test
    fun phrasesStayTogether() {
        val query = requireNotNull(SearchQuery.parse("\"react hooks\""))
        assertEquals(List(4) { "%react hooks%" }, query.args)
    }

    @Test
    fun minusExcludesTerms() {
        val query = requireNotNull(SearchQuery.parse("react -kotlin"))
        assertTrue(query.sql.contains("AND NOT"))
        assertTrue(query.args.contains("%kotlin%"))
    }

    @Test
    fun wildcardsAreEscaped() {
        val query = requireNotNull(SearchQuery.parse("100%"))
        assertTrue(query.args.all { it == "%100\\%%" })
        assertTrue(query.sql.contains("ESCAPE"))
    }

    @Test
    fun orSplitsAlternatives() {
        val query = requireNotNull(SearchQuery.parse("cats OR dogs"))
        assertTrue(query.sql.contains(" OR "))
    }

    @Test
    fun queryStopsAtFortyResults() {
        val query = requireNotNull(SearchQuery.parse("react"))
        assertTrue(query.sql.endsWith("LIMIT 40"))
    }
}
