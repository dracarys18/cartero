package app.cartero.sync

import android.app.job.JobInfo
import android.app.job.JobParameters
import android.app.job.JobScheduler
import android.app.job.JobService
import android.content.ComponentName
import android.content.Context
import android.os.PersistableBundle
import app.cartero.CarteroApp
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import java.util.concurrent.TimeUnit

class SyncJobService : JobService() {
    private var job: Job? = null

    override fun onStartJob(params: JobParameters): Boolean {
        val graph = (application as CarteroApp).graph
        job = graph.scope.launch {
            try {
                graph.sync.syncAll()
            } finally {
                jobFinished(params, false)
            }
        }
        return true
    }

    override fun onStopJob(params: JobParameters): Boolean {
        job?.cancel()
        return true
    }

    companion object {
        private const val JOB_ID = 1
        private const val WIFI_ONLY = "wifi_only"

        fun schedule(context: Context, minutes: Int, wifiOnly: Boolean) {
            val scheduler = context.getSystemService(JobScheduler::class.java)
            val interval = TimeUnit.MINUTES.toMillis(minutes.toLong())
            val pending = scheduler.getPendingJob(JOB_ID)
            if (pending != null && pending.intervalMillis == interval && pending.extras.getBoolean(WIFI_ONLY) == wifiOnly) return

            val job = JobInfo.Builder(JOB_ID, ComponentName(context, SyncJobService::class.java))
                .setPeriodic(interval)
                .setRequiredNetworkType(if (wifiOnly) JobInfo.NETWORK_TYPE_UNMETERED else JobInfo.NETWORK_TYPE_ANY)
                .setRequiresBatteryNotLow(true)
                .setPersisted(true)
                .setExtras(PersistableBundle().apply { putBoolean(WIFI_ONLY, wifiOnly) })
                .build()
            scheduler.schedule(job)
        }
    }
}
