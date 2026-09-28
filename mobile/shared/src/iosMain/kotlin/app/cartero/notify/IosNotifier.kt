package app.cartero.notify

import app.cartero.data.db.ArticleEntity
import app.cartero.data.db.RuleEntity
import platform.UserNotifications.UNAuthorizationStatusAuthorized
import platform.UserNotifications.UNAuthorizationStatusProvisional
import platform.UserNotifications.UNMutableNotificationContent
import platform.UserNotifications.UNNotificationRequest
import platform.UserNotifications.UNUserNotificationCenter

class IosNotifier : Notifier {
    private val center = UNUserNotificationCenter.currentNotificationCenter()
    private var authorized = false

    init {
        refresh()
    }

    override val enabled: Boolean
        get() {
            refresh()
            return authorized
        }

    override fun ensureChannel(rule: RuleEntity) = Unit

    override fun deleteChannel(ruleId: Long) = Unit

    override fun cancel(articleId: Long) {
        center.removeDeliveredNotificationsWithIdentifiers(listOf(identifier(articleId)))
    }

    override suspend fun post(rule: RuleEntity, articles: List<ArticleEntity>) {
        for (article in articles.take(MAX_PER_RULE)) {
            val content = UNMutableNotificationContent().apply {
                setTitle(article.title)
                setSubtitle(rule.label())
                setBody(listOfNotNull(article.source, article.topic).joinToString(" · "))
                setThreadIdentifier("rule-${rule.id}")
                setUserInfo(mapOf(ARTICLE_ID to article.id))
            }
            center.addNotificationRequest(
                UNNotificationRequest.requestWithIdentifier(identifier(article.id), content, null),
                withCompletionHandler = null,
            )
        }
    }

    private fun refresh() {
        center.getNotificationSettingsWithCompletionHandler { settings ->
            val status = settings?.authorizationStatus
            authorized = status == UNAuthorizationStatusAuthorized || status == UNAuthorizationStatusProvisional
        }
    }

    private fun identifier(articleId: Long) = "article-$articleId"

    companion object {
        const val ARTICLE_ID = "article_id"
        private const val MAX_PER_RULE = 6
    }
}
