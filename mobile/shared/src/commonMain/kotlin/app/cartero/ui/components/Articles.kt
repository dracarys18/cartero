package app.cartero.ui.components

import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animate
import androidx.compose.animation.core.spring
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.Orientation
import androidx.compose.foundation.gestures.draggable
import androidx.compose.foundation.gestures.rememberDraggableState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.paging.PagingData
import androidx.paging.compose.LazyPagingItems
import androidx.paging.compose.itemContentType
import androidx.paging.compose.itemKey
import androidx.paging.insertSeparators
import androidx.paging.map
import app.cartero.data.db.ArticleRow
import app.cartero.resources.Res
import app.cartero.resources.ic_bookmark
import app.cartero.resources.ic_bookmark_filled
import app.cartero.resources.ic_check
import app.cartero.resources.ic_mark_email_unread
import coil3.compose.AsyncImage
import kotlin.math.abs
import kotlin.math.roundToInt
import kotlin.math.sign
import org.jetbrains.compose.resources.painterResource

sealed interface ListItem {
    val key: String

    data class Day(val label: String) : ListItem {
        override val key = "day:$label"
    }

    data class Article(val row: ArticleRow) : ListItem {
        override val key = "article:${row.id}"
    }
}

fun PagingData<ArticleRow>.asItems(): PagingData<ListItem> = map { ListItem.Article(it) }

fun PagingData<ArticleRow>.withDayHeaders(): PagingData<ListItem> {
    val current = today()
    return asItems().insertSeparators { before, after ->
        val next = (after as? ListItem.Article)?.row ?: return@insertSeparators null
        val nextDay = next.publishedAt.localDate()
        val previousDay = (before as? ListItem.Article)?.row?.publishedAt?.localDate()
        if (previousDay == nextDay) null else ListItem.Day(dayLabel(nextDay, current))
    }
}

fun LazyListScope.articleItems(
    items: LazyPagingItems<ListItem>,
    onOpen: (Long) -> Unit,
    onToggleSaved: (Long) -> Unit,
    onToggleRead: (ArticleRow) -> Unit,
) {
    items(
        count = items.itemCount,
        key = items.itemKey { it.key },
        contentType = items.itemContentType { it::class },
    ) { index ->
        when (val item = items[index]) {
            is ListItem.Day -> DayHeader(item.label)
            is ListItem.Article -> SwipeableArticle(item.row, onOpen, onToggleSaved, onToggleRead)
            null -> Spacer(Modifier.height(ARTICLE_MIN_HEIGHT))
        }
    }
}

@Composable
private fun DayHeader(label: String) {
    Text(
        text = label,
        style = MaterialTheme.typography.titleSmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(start = 20.dp, end = 20.dp, top = 20.dp, bottom = 4.dp),
    )
}

@Composable
private fun SwipeableArticle(
    row: ArticleRow,
    onOpen: (Long) -> Unit,
    onToggleSaved: (Long) -> Unit,
    onToggleRead: (ArticleRow) -> Unit,
) {
    val threshold = with(LocalDensity.current) { SWIPE_THRESHOLD.toPx() }
    val haptics = LocalHapticFeedback.current
    val save by rememberUpdatedState { onToggleSaved(row.id) }
    val read by rememberUpdatedState { onToggleRead(row) }
    var offset by remember { mutableFloatStateOf(0f) }
    val direction by remember { derivedStateOf { offset.sign } }
    val armed by remember { derivedStateOf { abs(offset) > threshold } }

    LaunchedEffect(armed) {
        if (armed) haptics.performHapticFeedback(HapticFeedbackType.GestureThresholdActivate)
    }

    Box(Modifier.height(IntrinsicSize.Min)) {
        if (direction != 0f) SwipeBackground(started = direction > 0f, row = row)
        ArticleCard(
            row = row,
            onClick = { onOpen(row.id) },
            modifier = Modifier
                .offset { IntOffset(offset.roundToInt(), 0) }
                .draggable(
                    orientation = Orientation.Horizontal,
                    state = rememberDraggableState { delta -> offset += delta },
                    onDragStopped = {
                        when {
                            offset > threshold -> save()
                            offset < -threshold -> read()
                        }
                        animate(offset, 0f, animationSpec = spring(stiffness = Spring.StiffnessMediumLow)) { value, _ ->
                            offset = value
                        }
                    },
                ),
        )
    }
}

@Composable
private fun SwipeBackground(started: Boolean, row: ArticleRow) {
    val colors = MaterialTheme.colorScheme
    val icon = when {
        started && row.savedAt == null -> Res.drawable.ic_bookmark
        started -> Res.drawable.ic_bookmark_filled
        row.isRead -> Res.drawable.ic_mark_email_unread
        else -> Res.drawable.ic_check
    }
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(if (started) colors.primaryContainer else colors.secondaryContainer)
            .padding(horizontal = 28.dp),
        contentAlignment = if (started) Alignment.CenterStart else Alignment.CenterEnd,
    ) {
        Icon(
            painterResource(icon),
            contentDescription = null,
            tint = if (started) colors.onPrimaryContainer else colors.onSecondaryContainer,
        )
    }
}

@Composable
fun ArticleCard(row: ArticleRow, onClick: () -> Unit, modifier: Modifier = Modifier) {
    val colors = MaterialTheme.colorScheme
    val typography = MaterialTheme.typography
    val meta = remember(row.source, row.publishedAt, row.readingMinutes, colors.secondary) { metaLine(row, colors.secondary) }

    Row(
        modifier = modifier
            .fillMaxWidth()
            .background(colors.surface)
            .clickable(onClick = onClick)
            .padding(horizontal = 20.dp, vertical = 14.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Column(Modifier.weight(1f)) {
            if (row.topic != null) {
                Text(
                    text = row.topic.uppercase(),
                    style = typography.labelSmall.copy(letterSpacing = 1.1.sp, fontWeight = FontWeight.Bold),
                    color = colors.primary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Spacer(Modifier.height(4.dp))
            }
            Text(
                text = row.title,
                style = typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                color = if (row.isRead) colors.onSurfaceVariant else colors.onSurface,
                maxLines = 3,
                overflow = TextOverflow.Ellipsis,
            )
            Spacer(Modifier.height(6.dp))
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                if (row.savedAt != null) {
                    Icon(
                        painterResource(Res.drawable.ic_bookmark_filled),
                        contentDescription = "Saved",
                        modifier = Modifier.size(14.dp),
                        tint = colors.tertiary,
                    )
                }
                Text(
                    text = meta,
                    style = typography.bodySmall.copy(fontWeight = FontWeight.Normal),
                    color = colors.outline,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        if (row.imageUrl != null) {
            AsyncImage(
                model = row.imageUrl,
                contentDescription = null,
                contentScale = ContentScale.Crop,
                alpha = if (row.isRead) 0.7f else 1f,
                modifier = Modifier
                    .size(THUMBNAIL)
                    .clip(RoundedCornerShape(16.dp))
                    .background(colors.surfaceContainerHigh),
            )
        }
    }
}

private fun metaLine(row: ArticleRow, sourceColor: Color): AnnotatedString = buildAnnotatedString {
    withStyle(SpanStyle(color = sourceColor)) { append(row.source) }
    append(" · ")
    append(relativeTime(row.publishedAt))
    if (row.readingMinutes > 0) append(" · ${row.readingMinutes} min")
}

private val THUMBNAIL = 84.dp
private val SWIPE_THRESHOLD = 96.dp
private val ARTICLE_MIN_HEIGHT = 96.dp
