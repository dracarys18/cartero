import Foundation
import IrohLib
import Shared

final class IrohNetwork: NSObject, IrohBridge {
    private var endpoint: Endpoint?
    private var alpn = Data()
    private var maxMessage: UInt32 = 0

    func start(
        secretKey: String?,
        alpn: String,
        maxMessageBytes: Int32,
        handler: IrohRequestHandler,
        completion: @escaping (String?, String?, String?) -> Void
    ) {
        self.alpn = Data(alpn.utf8)
        maxMessage = UInt32(maxMessageBytes)
        Task {
            do {
                let key = try secretKey.flatMap { Data(base64Encoded: $0) }.map { try SecretKey.fromBytes(bytes: $0) }
                    ?? SecretKey.generate()
                let endpoint = try await Endpoint.bind(options: EndpointOptions(
                    preset: presetN0(),
                    secretKey: key.toBytes(),
                    alpns: [self.alpn]
                ))
                self.endpoint = endpoint
                Task { await self.accept(endpoint, handler) }
                completion(key.toBytes().base64EncodedString(), key.public().description, nil)
            } catch {
                completion(nil, nil, "\(error)")
            }
        }
    }

    func request(peer: String, message: String, completion: @escaping (String?, String?) -> Void) {
        Task {
            do {
                guard let endpoint else {
                    completion(nil, "Sync hasn't started")
                    return
                }
                let addr = EndpointAddr(id: try EndpointId.fromString(s: peer), relayUrl: nil, addresses: [])
                let connection = try await endpoint.connect(addr: addr, alpn: alpn)
                defer { try? connection.close(errorCode: 0, reason: Data()) }
                let stream = try await connection.openBi()
                let send = stream.send()
                try await send.writeAll(buf: Data(message.utf8))
                try await send.finish()
                let reply = try await stream.recv().readToEnd(sizeLimit: maxMessage)
                completion(String(decoding: reply, as: UTF8.self), nil)
            } catch {
                completion(nil, "\(error)")
            }
        }
    }

    private func accept(_ endpoint: Endpoint, _ handler: IrohRequestHandler) async {
        while let incoming = await endpoint.acceptNext() {
            Task { try? await self.serve(incoming, handler) }
        }
    }

    private func serve(_ incoming: Incoming, _ handler: IrohRequestHandler) async throws {
        let connection = try await incoming.accept().connect()
        let stream = try await connection.acceptBi()
        let request = try await stream.recv().readToEnd(sizeLimit: maxMessage)
        let reply: String? = await withCheckedContinuation { continuation in
            handler.handle(
                peer: connection.remoteId().description,
                message: String(decoding: request, as: UTF8.self)
            ) { continuation.resume(returning: $0) }
        }
        guard let reply else {
            try connection.close(errorCode: 1, reason: Data())
            return
        }
        let send = stream.send()
        try await send.writeAll(buf: Data(reply.utf8))
        try await send.finish()
        _ = await connection.closed()
    }
}
