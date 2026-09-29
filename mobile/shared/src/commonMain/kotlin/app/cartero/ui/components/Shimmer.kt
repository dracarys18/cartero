package app.cartero.ui.components

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.platform.LocalWindowInfo
import androidx.compose.ui.unit.dp

@Composable
fun rememberShimmerBrush(): Brush {
    val base = MaterialTheme.colorScheme.surfaceContainer
    val highlight = MaterialTheme.colorScheme.surfaceContainerHighest
    val width = LocalWindowInfo.current.containerSize.width.toFloat()
    val transition = rememberInfiniteTransition()
    val x by transition.animateFloat(
        initialValue = -SHIMMER_BAND,
        targetValue = width + SHIMMER_BAND,
        animationSpec = infiniteRepeatable(tween(SHIMMER_MS, easing = LinearEasing)),
    )
    return Brush.linearGradient(
        colors = listOf(base, highlight, base),
        start = Offset(x - SHIMMER_BAND, 0f),
        end = Offset(x + SHIMMER_BAND, 0f),
    )
}

@Composable
fun ShimmerBlock(brush: Brush, modifier: Modifier = Modifier, shape: Shape = RoundedCornerShape(6.dp)) {
    Box(modifier.background(brush, shape))
}

private const val SHIMMER_MS = 1_200
private const val SHIMMER_BAND = 360f
