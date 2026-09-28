package app.cartero.ui.platform

import android.Manifest
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.platform.LocalContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@Composable
actual fun rememberNotificationPermission(onResult: (Boolean) -> Unit): () -> Unit {
    val launcher = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission(), onResult)
    return { launcher.launch(Manifest.permission.POST_NOTIFICATIONS) }
}

@Composable
actual fun rememberOpmlImport(onOpml: (String) -> Unit): () -> Unit {
    val resolver = LocalContext.current.contentResolver
    val scope = rememberCoroutineScope()
    val launcher = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            val text = withContext(Dispatchers.IO) {
                runCatching { resolver.openInputStream(uri)?.use { it.readBytes().decodeToString() } }.getOrNull()
            }
            text?.let(onOpml)
        }
    }
    return { launcher.launch(OPML_TYPES) }
}

@Composable
actual fun rememberOpmlExport(opml: suspend () -> String, onResult: (Boolean) -> Unit): () -> Unit {
    val resolver = LocalContext.current.contentResolver
    val scope = rememberCoroutineScope()
    val launcher = rememberLauncherForActivityResult(ActivityResultContracts.CreateDocument(OPML_MIME)) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            val text = opml()
            val written = withContext(Dispatchers.IO) {
                runCatching { resolver.openOutputStream(uri, "wt")?.use { it.write(text.encodeToByteArray()) } != null }.getOrDefault(false)
            }
            onResult(written)
        }
    }
    return { launcher.launch(OPML_FILE) }
}

private const val OPML_MIME = "text/x-opml"
private const val OPML_FILE = "cartero.opml"
private val OPML_TYPES = arrayOf(OPML_MIME, "text/xml", "application/xml", "*/*")
