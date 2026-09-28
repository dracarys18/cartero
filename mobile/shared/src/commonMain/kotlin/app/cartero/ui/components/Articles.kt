package app.cartero.ui.components

import org.jetbrains.compose.resources.DrawableResource
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SwipeToDismissBox
import androidx.compose.material3.SwipeToDismissBoxValue
import androidx.compose.material3.Text
import androidx.compose.material3.rememberSwipeToDismissBoxState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
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
import kotlinx.coroutines.launch
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
    val state = rememberSwipeToDismissBoxState()
    val scope = rememberCoroutineScope()
    SwipeToDismissBox(
        state = state,
        onDismiss = { value ->
            when (value) {
                SwipeToDismissBoxValue.StartToEnd -> onToggleSaved(row.id)
                SwipeToDismissBoxValue.EndToStart -> onToggleRead(row)
                SwipeToDismissBoxValue.Settled -> Unit
            }
            scope.launch { state.reset() }
        },
        backgroundContent = { SwipeBackground(state.dismissDirection, row) },
    ) {
        ArticleCard(row, onClick = { onOpen(row.id) })
    }
}

@Composable
private fun SwipeBackground(direction: SwipeToDismissBoxValue, row: ArticleRow) {
    val colors = MaterialTheme.colorScheme
    val (icon, container, content, alignment) = when (direction) {
        SwipeToDismissBoxValue.StartToEnd -> SwipeStyle(
            icon = if (row.savedAt == null) Res.drawable.ic_bookmark else Res.drawable.ic_bookmark_filled,
            container = colors.primaryContainer,
            content = colors.onPrimaryContainer,
            alignment = Alignment.CenterStart,
        )
        SwipeToDismissBoxValue.EndToStart -> SwipeStyle(
            icon = if (row.isRead) Res.drawable.ic_mark_email_unread else Res.drawable.ic_check,
            container = colors.secondaryContainer,
            content = colors.onSecondaryContainer,
            alignment = Alignment.CenterEnd,
        )
        SwipeToDismissBoxValue.Settled -> return
    }
    Box(
        modifier = Modifier.fillMaxSize().background(container).padding(horizontal = 28.dp),
        contentAlignment = alignment,
    ) {
        Icon(painterResource(icon), contentDescription = null, tint = content)
    }
}

private data class SwipeStyle(
    val icon: DrawableResource,
    val container: Color,
    val content: Color,
    val alignment: Alignment,
)

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
private val ARTICLE_MIN_HEIGHT = 96.dp
