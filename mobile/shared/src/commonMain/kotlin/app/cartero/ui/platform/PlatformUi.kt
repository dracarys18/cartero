package app.cartero.ui.platform

import androidx.compose.runtime.Composable

@Composable
expect fun rememberNotificationPermission(onResult: (Boolean) -> Unit): () -> Unit

@Composable
expect fun rememberOpmlImport(onOpml: (String) -> Unit): () -> Unit

@Composable
expect fun rememberOpmlExport(opml: suspend () -> String, onResult: (Boolean) -> Unit): () -> Unit

@Composable
expect fun rememberQrScanner(onScanned: (String) -> Unit): () -> Unit

@Composable
expect fun SystemBarsHidden(hidden: Boolean)
