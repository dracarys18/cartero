package app.cartero

import androidx.compose.runtime.getValue
import androidx.compose.ui.window.ComposeUIViewController
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import app.cartero.notify.IosNotifier
import app.cartero.platform.IosActions
import app.cartero.platform.IosServices
import app.cartero.ui.CarteroRoot
import app.cartero.ui.navigation.AppRequest
import app.cartero.ui.reader.ReaderWebViews
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.coroutines.Job
import kotlinx.coroutines.MainScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch
import platform.BackgroundTasks.BGAppRefreshTask
import platform.BackgroundTasks.BGAppRefreshTaskRequest
import platform.BackgroundTasks.BGTaskScheduler
import platform.Foundation.NSDate
import platform.Foundation.dateWithTimeIntervalSinceNow
import platform.UIKit.UIViewController
import platform.UserNotifications.UNNotification
import platform.UserNotifications.UNNotificationPresentationOptionBanner
import platform.UserNotifications.UNNotificationPresentationOptionList
import platform.UserNotifications.UNNotificationPresentationOptions
import platform.UserNotifications.UNNotificationResponse
import platform.UserNotifications.UNUserNotificationCenter
import platform.UserNotifications.UNUserNotificationCenterDelegateProtocol
import platform.darwin.NSObject

object CarteroIos {
    const val SYNC_TASK = "app.cartero.sync"

    val graph: AppGraph by lazy { AppGraph(IosServices()) }

    private val requests = MutableStateFlow<AppRequest?>(null)
    private val notifications = NotificationDelegate { requests.value = AppRequest.OpenArticle(it) }
    private val main = MainScope()
    private val systemBarsHidden = MutableStateFlow(false)
    private var systemBarsObserver: Job? = null

    fun start() {
        UNUserNotificationCenter.currentNotificationCenter().delegate = notifications
        BGTaskScheduler.sharedScheduler.registerForTaskWithIdentifier(SYNC_TASK, usingQueue = null) { task ->
            (task as? BGAppRefreshTask)?.let(::runSync)
        }
        graph.scope.launch {
            graph.settings.settings
                .map { it.syncMinutes }
                .distinctUntilChanged()
                .collect(::scheduleSync)
        }
        main.launch { ReaderWebViews.prepare() }
    }

    fun mainViewController(): UIViewController = ComposeUIViewController {
        val request by requests.collectAsStateWithLifecycle()
        CarteroRoot(graph, IosActions, request, onRequestHandled = { requests.value = null })
    }

    fun observeSystemBars(onChange: (Boolean) -> Unit) {
        systemBarsObserver?.cancel()
        systemBarsObserver = main.launch { systemBarsHidden.collect(onChange) }
    }

    internal fun setSystemBarsHidden(hidden: Boolean) {
        systemBarsHidden.value = hidden
    }

    @OptIn(ExperimentalForeignApi::class)
    private fun scheduleSync(minutes: Int) {
        val request = BGAppRefreshTaskRequest(SYNC_TASK)
        request.earliestBeginDate = NSDate.dateWithTimeIntervalSinceNow(minutes * 60.0)
        BGTaskScheduler.sharedScheduler.submitTaskRequest(request, error = null)
    }

    private fun runSync(task: BGAppRefreshTask) {
        val job = graph.scope.launch {
            graph.sync.syncAll()
            scheduleSync(graph.settings.current().syncMinutes)
            task.setTaskCompletedWithSuccess(true)
        }
        task.expirationHandler = {
            job.cancel()
            task.setTaskCompletedWithSuccess(false)
        }
    }
}

private class NotificationDelegate(private val onOpen: (Long) -> Unit) : NSObject(), UNUserNotificationCenterDelegateProtocol {
    override fun userNotificationCenter(
        center: UNUserNotificationCenter,
        didReceiveNotificationResponse: UNNotificationResponse,
        withCompletionHandler: () -> Unit,
    ) {
        (didReceiveNotificationResponse.notification.request.content.userInfo[IosNotifier.ARTICLE_ID] as? Long)?.let(onOpen)
        withCompletionHandler()
    }

    override fun userNotificationCenter(
        center: UNUserNotificationCenter,
        willPresentNotification: UNNotification,
        withCompletionHandler: (UNNotificationPresentationOptions) -> Unit,
    ) {
        withCompletionHandler(UNNotificationPresentationOptionBanner or UNNotificationPresentationOptionList)
    }
}
