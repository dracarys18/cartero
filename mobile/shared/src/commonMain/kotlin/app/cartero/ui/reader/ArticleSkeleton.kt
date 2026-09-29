package app.cartero.ui.reader

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import app.cartero.ui.components.ShimmerBlock
import app.cartero.ui.components.rememberShimmerBrush

@Composable
fun ArticleSkeleton(top: Dp, modifier: Modifier = Modifier) {
    val brush = rememberShimmerBrush()
    Box(modifier.fillMaxSize(), contentAlignment = Alignment.TopCenter) {
        Column(
            modifier = Modifier
                .widthIn(max = 720.dp)
                .fillMaxWidth()
                .padding(start = 20.dp, end = 20.dp, top = top + 8.dp),
        ) {
            ShimmerBlock(brush, Modifier.fillMaxWidth(0.22f).height(10.dp))
            Spacer(Modifier.height(16.dp))
            Column(verticalArrangement = Arrangement.spacedBy(9.dp)) {
                TITLE_LINES.forEach { ShimmerBlock(brush, Modifier.fillMaxWidth(it).height(24.dp)) }
            }
            Spacer(Modifier.height(18.dp))
            ShimmerBlock(brush, Modifier.fillMaxWidth(0.5f).height(12.dp))
            Spacer(Modifier.height(36.dp))
            ShimmerBlock(
                brush,
                Modifier.fillMaxWidth().aspectRatio(16f / 9f),
                shape = RoundedCornerShape(16.dp),
            )
            Spacer(Modifier.height(28.dp))
            Column(verticalArrangement = Arrangement.spacedBy(15.dp)) {
                BODY_LINES.forEach { ShimmerBlock(brush, Modifier.fillMaxWidth(it).height(14.dp)) }
            }
        }
    }
}

private val TITLE_LINES = listOf(1f, 0.94f, 0.58f)
private val BODY_LINES = listOf(1f, 0.97f, 1f, 0.92f, 0.99f, 0.64f, 1f, 0.95f, 0.88f)
