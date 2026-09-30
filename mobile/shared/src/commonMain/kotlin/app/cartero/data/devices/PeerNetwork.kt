package app.cartero.data.devices

interface PeerNetwork {
    suspend fun start(secretKey: String?, handler: suspend (peer: String, message: String) -> String?): PeerIdentity

    suspend fun request(peer: String, message: String): String

    companion object {
        const val ALPN = "cartero/devices/1"
        const val MAX_MESSAGE_BYTES = 16 * 1024 * 1024
    }
}

data class PeerIdentity(val secretKey: String, val id: String)
