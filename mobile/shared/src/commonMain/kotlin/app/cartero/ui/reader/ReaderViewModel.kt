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

    private val fetching = MutableStateFlow(true)
    val loadingFullText: StateFlow<Boolean> = fetching.asStateFlow()

    init {
        viewModelScope.launch {
            graph.articles.setRead(articleId, true)
            val current = graph.articles.article(articleId).filterNotNull().first()
            if (current.needsFullText()) fetchFullText(current) else fetching.value = false
        }
    }

    fun loadFullText() {
        val current = article.value ?: return
        if (fetching.value) return
        viewModelScope.launch { fetchFullText(current) }
    }

    fun toggleSaved() {
        viewModelScope.launch { graph.articles.toggleSaved(articleId) }
    }

    fun setScale(scale: Float) {
        viewModelScope.launch { graph.settings.update { it.copy(readerScale = scale) } }
    }

    private suspend fun fetchFullText(current: ArticleEntity) {
        fetching.value = true
        try {
            if (graph.articles.loadFullText(current)) article.first { it?.fullText == true }
        } finally {
            fetching.value = false
        }
    }
}
