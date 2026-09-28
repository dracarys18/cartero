package app.cartero.notify

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import app.cartero.CarteroApp
import kotlinx.coroutines.launch

class NotificationActionReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val articleId = intent.getLongExtra(AndroidNotifier.EXTRA_ARTICLE_ID, -1)
        if (articleId < 0) return
        val graph = (context.applicationContext as CarteroApp).graph
        val pending = goAsync()
        graph.scope.launch {
            try {
                when (intent.action) {
                    AndroidNotifier.ACTION_SAVE -> graph.articles.save(articleId)
                    AndroidNotifier.ACTION_READ -> graph.articles.setRead(articleId, true)
                }
                graph.notifier.cancel(articleId)
            } finally {
                pending.finish()
            }
        }
    }
}
