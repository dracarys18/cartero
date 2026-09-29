package app.cartero.ui.reader

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.cartero.AppGraph
import app.cartero.data.db.ArticleEntity
import app.cartero.data.sync.needsFullText
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

class ReaderViewModel(private val articleId: Long, private val graph: AppGraph) : ViewModel() {
    val article: StateFlow<ArticleEntity?> = graph.articles.article(articleId)
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), null)

    var scrollFraction = 0f

    private val fetching = MutableStateFlow(false)
    val loadingFullText: StateFlow<Boolean> = fetching.asStateFlow()

    init {
        viewModelScope.launch {
            graph.articles.setRead(articleId, true)
            val current = graph.articles.article(articleId).filterNotNull().first()
            if (current.needsFullText()) fetchFullText(current)
        }
    }

    fun loadFullText() {
        val current = article.value ?: return
        viewModelScope.launch { fetchFullText(current) }
    }

    fun toggleSaved() {
        viewModelScope.launch { graph.articles.toggleSaved(articleId) }
    }

    fun setScale(scale: Float) {
        viewModelScope.launch { graph.settings.update { it.copy(readerScale = scale) } }
    }

    private suspend fun fetchFullText(article: ArticleEntity) {
        if (fetching.value) return
        fetching.value = true
        try {
            graph.articles.loadFullText(article)
        } finally {
            fetching.value = false
        }
    }
}
