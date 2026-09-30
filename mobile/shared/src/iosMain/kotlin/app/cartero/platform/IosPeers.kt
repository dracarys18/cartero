package app.cartero.platform

import app.cartero.data.devices.PeerIdentity
import app.cartero.data.devices.PeerNetwork
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

interface IrohBridge {
    fun start(
        secretKey: String?,
        alpn: String,
        maxMessageBytes: Int,
        handler: IrohRequestHandler,
        completion: (secretKey: String?, id: String?, error: String?) -> Unit,
    )

    fun request(peer: String, message: String, completion: (reply: String?, error: String?) -> Unit)
}

interface IrohRequestHandler {
    fun handle(peer: String, message: String, reply: (String?) -> Unit)
}

class IosPeers(private val bridge: IrohBridge) : PeerNetwork {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    override suspend fun start(
        secretKey: String?,
        handler: suspend (peer: String, message: String) -> String?,
    ): PeerIdentity = suspendCancellableCoroutine { continuation ->
        val requests = object : IrohRequestHandler {
            override fun handle(peer: String, message: String, reply: (String?) -> Unit) {
                scope.launch { reply(runCatching { handler(peer, message) }.getOrNull()) }
            }
        }
        bridge.start(secretKey, PeerNetwork.ALPN, PeerNetwork.MAX_MESSAGE_BYTES, requests) { key, id, error ->
            if (key != null && id != null) {
                continuation.resume(PeerIdentity(key, id))
            } else {
                continuation.resumeWithException(IllegalStateException(error))
            }
        }
    }

    override suspend fun request(peer: String, message: String): String = suspendCancellableCoroutine { continuation ->
        bridge.request(peer, message) { reply, error ->
            if (reply != null) continuation.resume(reply) else continuation.resumeWithException(IllegalStateException(error))
        }
    }
}
