package app.cartero.ui.search

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.cartero.AppGraph
import app.cartero.data.db.ArticleRow
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.flatMapLatest
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

data class SearchResults(
    val query: String = "",
    val rows: List<ArticleRow> = emptyList(),
)

class SearchViewModel(private val graph: AppGraph) : ViewModel() {
    private val submitted = MutableStateFlow("")

    @OptIn(ExperimentalCoroutinesApi::class)
    val results: StateFlow<SearchResults> = submitted
        .flatMapLatest { query ->
            if (query.isEmpty()) {
                flowOf(SearchResults())
            } else {
                graph.articles.search(query).map { SearchResults(query, it) }
            }
        }
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), SearchResults())

    fun search(text: String) {
        submitted.value = text.trim()
    }

    fun clear() {
        submitted.value = ""
    }

    fun toggleSaved(id: Long) {
        viewModelScope.launch { graph.articles.toggleSaved(id) }
    }

    fun toggleRead(row: ArticleRow) {
        viewModelScope.launch { graph.articles.setRead(row.id, !row.isRead) }
    }
}
