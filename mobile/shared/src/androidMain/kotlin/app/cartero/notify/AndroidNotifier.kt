package app.cartero.notify

import android.app.Activity
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationChannelGroup
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.drawable.Icon
import app.cartero.data.db.ArticleEntity
import app.cartero.data.db.RuleEntity
import coil3.ImageLoader
import coil3.request.ImageRequest
import coil3.request.SuccessResult
import coil3.request.allowHardware
import coil3.toBitmap

data class NotificationIcons(val small: Int, val save: Int, val read: Int)

class AndroidNotifier(
    private val context: Context,
    private val imageLoader: ImageLoader,
    private val icons: NotificationIcons,
    private val launcher: Class<out Activity>,
    private val receiver: Class<out BroadcastReceiver>,
) : Notifier {
    private val manager = context.getSystemService(NotificationManager::class.java)

    override val enabled: Boolean get() = manager.areNotificationsEnabled()

    override fun ensureChannel(rule: RuleEntity) {
        val id = channelId(rule.id)
        if (manager.getNotificationChannel(id) != null) return
        manager.createNotificationChannelGroup(NotificationChannelGroup(GROUP, "Rules"))
        val channel = NotificationChannel(id, rule.channelName(), NotificationManager.IMPORTANCE_DEFAULT)
        channel.group = GROUP
        manager.createNotificationChannel(channel)
    }

    override fun deleteChannel(ruleId: Long) = manager.deleteNotificationChannel(channelId(ruleId))

    override fun cancel(articleId: Long) = manager.cancel(ARTICLE_TAG, articleId.toInt())

    override suspend fun post(rule: RuleEntity, articles: List<ArticleEntity>) {
        if (!enabled || articles.isEmpty()) return
        ensureChannel(rule)
        val channel = channelId(rule.id)
        val group = "rule-${rule.id}"

        for (article in articles.take(MAX_PER_RULE)) {
            val image = article.imageUrl?.let { loadBitmap(it) }
            val notification = Notification.Builder(context, channel)
                .setSmallIcon(icons.small)
                .setContentTitle(article.title)
                .setContentText(listOfNotNull(article.source, article.topic).joinToString(" · "))
                .setSubText(rule.label())
                .setWhen(article.publishedAt)
                .setShowWhen(true)
                .setAutoCancel(true)
                .setGroup(group)
                .setContentIntent(openIntent(article.id))
                .addAction(action(article.id, ACTION_SAVE, "Save", icons.save))
                .addAction(action(article.id, ACTION_READ, "Mark read", icons.read))
                .apply {
                    if (image != null) {
                        setLargeIcon(image)
                        setStyle(Notification.BigPictureStyle().bigPicture(image).bigLargeIcon(null as Icon?))
                    } else if (article.summary.isNotEmpty()) {
                        setStyle(Notification.BigTextStyle().bigText(article.summary))
                    }
                }
                .build()
            manager.notify(ARTICLE_TAG, article.id.toInt(), notification)
        }

        if (articles.size > 1) {
            val inbox = Notification.InboxStyle().setSummaryText(rule.label())
            articles.take(MAX_PER_RULE).forEach { inbox.addLine(it.title) }
            val summary = Notification.Builder(context, channel)
                .setSmallIcon(icons.small)
                .setContentTitle("${articles.size} new · ${rule.label()}")
                .setStyle(inbox)
                .setGroup(group)
                .setGroupSummary(true)
                .setAutoCancel(true)
                .build()
            manager.notify(SUMMARY_TAG, rule.id.toInt(), summary)
        }
    }

    private suspend fun loadBitmap(url: String): Bitmap? {
        val request = ImageRequest.Builder(context)
            .data(url)
            .size(IMAGE_WIDTH, IMAGE_HEIGHT)
            .allowHardware(false)
            .build()
        return (imageLoader.execute(request) as? SuccessResult)?.image?.toBitmap()
    }

    private fun openIntent(articleId: Long): PendingIntent {
        val intent = Intent(context, launcher)
            .setAction(ACTION_OPEN)
            .putExtra(EXTRA_ARTICLE_ID, articleId)
            .addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP)
        return PendingIntent.getActivity(context, articleId.toInt(), intent, PENDING_FLAGS)
    }

    private fun action(articleId: Long, action: String, title: String, icon: Int): Notification.Action {
        val intent = Intent(context, receiver)
            .setAction(action)
            .putExtra(EXTRA_ARTICLE_ID, articleId)
        val pending = PendingIntent.getBroadcast(context, articleId.toInt(), intent, PENDING_FLAGS)
        return Notification.Action.Builder(Icon.createWithResource(context, icon), title, pending).build()
    }

    companion object {
        const val ACTION_OPEN = "app.cartero.OPEN_ARTICLE"
        const val ACTION_SAVE = "app.cartero.SAVE_ARTICLE"
        const val ACTION_READ = "app.cartero.READ_ARTICLE"
        const val EXTRA_ARTICLE_ID = "article_id"

        private const val GROUP = "rules"
        private const val ARTICLE_TAG = "article"
        private const val SUMMARY_TAG = "summary"
        private const val MAX_PER_RULE = 6
        private const val IMAGE_WIDTH = 720
        private const val IMAGE_HEIGHT = 360
        private const val PENDING_FLAGS = PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT

        fun channelId(ruleId: Long) = "rule-$ruleId"
    }
}
