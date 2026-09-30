package app.cartero.data

import app.cartero.nowMillis
import app.cartero.data.db.ArticleDao
import app.cartero.data.db.RuleAction
import app.cartero.data.db.RuleDao
import app.cartero.data.db.RuleEntity
import app.cartero.data.db.RuleField
import app.cartero.data.devices.DeviceSync
import app.cartero.data.sync.ContentLoader
import app.cartero.data.sync.RuleMatcher
import app.cartero.notify.Notifier
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.launch

class RuleRepository(
    private val dao: RuleDao,
    private val articles: ArticleDao,
    private val notifier: Notifier,
    private val content: ContentLoader,
    private val devices: DeviceSync,
    private val scope: CoroutineScope,
) {
    val rules: Flow<List<RuleEntity>> = dao.observe()

    suspend fun add(action: RuleAction, field: RuleField, value: String): Int {
        val rule = RuleEntity(action = action, field = field, value = value.trim())
        val id = dao.insert(rule)
        if (id == -1L) return 0
        devices.ruleChanged(rule)
        return when (action) {
            RuleAction.Notify -> {
                notifier.ensureChannel(rule.copy(id = id))
                0
            }
            RuleAction.Save -> saveExisting(rule)
        }
    }

    suspend fun setEnabled(rule: RuleEntity, enabled: Boolean): Int {
        dao.setEnabled(rule.id, enabled)
        devices.ruleChanged(rule)
        return if (enabled && rule.action == RuleAction.Save) saveExisting(rule) else 0
    }

    suspend fun delete(rule: RuleEntity) {
        dao.delete(rule)
        devices.ruleChanged(rule)
        if (rule.action == RuleAction.Notify) notifier.deleteChannel(rule.id)
    }

    private suspend fun saveExisting(rule: RuleEntity): Int {
        val matcher = RuleMatcher(rule)
        val matches = articles.unsavedMatching(rule.value).filter(matcher::matches)
        if (matches.isEmpty()) return 0
        articles.save(matches.map { it.id }, nowMillis())
        scope.launch { content.prefetch(matches) }
        return matches.size
    }
}
