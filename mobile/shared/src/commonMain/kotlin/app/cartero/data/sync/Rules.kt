package app.cartero.data.sync

import app.cartero.data.db.ArticleEntity
import app.cartero.data.db.RuleEntity
import app.cartero.data.db.RuleField

class RuleMatcher(val rule: RuleEntity) {
    private val predicate: (ArticleEntity) -> Boolean = when (rule.field) {
        RuleField.Topic -> { article -> article.topic.equals(rule.value, ignoreCase = true) }
        RuleField.Source -> { article -> article.source.equals(rule.value, ignoreCase = true) }
        RuleField.Keyword -> {
            val regex = Regex("\\b${Regex.escape(rule.value.trim())}\\b", RegexOption.IGNORE_CASE)
            val match: (ArticleEntity) -> Boolean = { regex.containsMatchIn(it.title) || regex.containsMatchIn(it.summary) }
            match
        }
    }

    fun matches(article: ArticleEntity): Boolean = predicate(article)
}
