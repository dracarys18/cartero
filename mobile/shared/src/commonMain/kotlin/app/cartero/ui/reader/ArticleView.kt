package app.cartero.ui.reader

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier

@Composable
expect fun ArticleView(
    html: String,
    baseUrl: String?,
    textZoom: Int,
    onLink: (String) -> Unit,
    onScroll: (progress: Float, delta: Int) -> Unit,
    modifier: Modifier = Modifier,
)
