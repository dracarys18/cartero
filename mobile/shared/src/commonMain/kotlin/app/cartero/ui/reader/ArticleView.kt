package app.cartero.ui.reader

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.Dp

@Composable
expect fun ArticleView(
    html: String,
    baseUrl: String?,
    textZoom: Int,
    topInset: Dp,
    initialScroll: Float,
    onLink: (String) -> Unit,
    onScroll: (progress: Float, delta: Int) -> Unit,
    onTap: () -> Unit,
    onLoaded: () -> Unit,
    modifier: Modifier = Modifier,
)
