package app.cartero.platform

import android.content.Context
import app.cartero.data.devices.PeerIdentity
import app.cartero.data.devices.PeerNetwork
import computer.iroh.Endpoint
import computer.iroh.EndpointAddr
import computer.iroh.EndpointId
import computer.iroh.EndpointOptions
import computer.iroh.Incoming
import computer.iroh.IrohAndroid
import computer.iroh.SecretKey
import computer.iroh.presetN0
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlin.io.encoding.Base64

class IrohPeers(private val context: Context) : PeerNetwork {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val alpn = PeerNetwork.ALPN.encodeToByteArray()
    private val maxMessage = PeerNetwork.MAX_MESSAGE_BYTES.toUInt()
    @Volatile
    private var endpoint: Endpoint? = null

    override suspend fun start(
        secretKey: String?,
        handler: suspend (peer: String, message: String) -> String?,
    ): PeerIdentity {
        IrohAndroid.installAndroidContext(context)
        val key = secretKey?.let { SecretKey.fromBytes(Base64.decode(it)) } ?: SecretKey.generate()
        val bound = Endpoint.bind(EndpointOptions(preset = presetN0(), secretKey = key.toBytes(), alpns = listOf(alpn)))
        endpoint = bound
        scope.launch {
            while (true) {
                val incoming = bound.acceptNext() ?: break
                scope.launch { runCatching { serve(incoming, handler) } }
            }
        }
        return PeerIdentity(Base64.encode(key.toBytes()), key.public().toString())
    }

    override suspend fun request(peer: String, message: String): String {
        val bound = checkNotNull(endpoint) { "Sync hasn't started" }
        val connection = bound.connect(EndpointAddr(EndpointId.fromString(peer), null, emptyList()), alpn)
        try {
            val stream = connection.openBi()
            val send = stream.send()
            send.writeAll(message.encodeToByteArray())
            send.finish()
            return stream.recv().readToEnd(maxMessage).decodeToString()
        } finally {
            connection.close(0, ByteArray(0))
        }
    }

    private suspend fun serve(incoming: Incoming, handler: suspend (peer: String, message: String) -> String?) {
        val connection = incoming.accept().connect()
        val stream = connection.acceptBi()
        val request = stream.recv().readToEnd(maxMessage).decodeToString()
        val reply = handler(connection.remoteId().toString(), request)
        if (reply == null) {
            connection.close(1, ByteArray(0))
            return
        }
        val send = stream.send()
        send.writeAll(reply.encodeToByteArray())
        send.finish()
        connection.closed()
    }
}
