package app.cartero.notify

import app.cartero.data.db.ArticleEntity
import app.cartero.data.db.RuleEntity
import app.cartero.data.db.RuleField

interface Notifier {
    val enabled: Boolean
    fun ensureChannel(rule: RuleEntity)
    fun deleteChannel(ruleId: Long)
    fun cancel(articleId: Long)
    suspend fun post(rule: RuleEntity, articles: List<ArticleEntity>)
}

fun RuleEntity.label(): String = if (field == RuleField.Keyword) "“$value”" else value

fun RuleEntity.channelName(): String = "${field.name} · ${label()}"
