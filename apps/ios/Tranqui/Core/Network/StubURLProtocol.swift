import Foundation

/// The canned answer a stubbed request produces. An offline simulation is a
/// handler that throws (URLProtocol reports failures through its client).
struct StubResponse: Sendable {
    let status: Int
    let headers: [String: String]
    let body: Data

    init(status: Int, headers: [String: String] = [:], body: Data = Data()) {
        self.status = status
        self.headers = headers
        self.body = body
    }
}

/// URLSession stubbing for the tests (and the launch-argument smoke mode
/// that lands with the onboarding UI): installs one handler that answers
/// every request, records exactly what went out, and can fail a request
/// with a URLError to simulate offline. This is the idiomatic replacement
/// for MockWebServer: the tests assert method, path and exact body on the
/// recorded requests, with the contract's own examples as fixtures.
///
/// State is process-global by nature of URLProtocol; the lock keeps the
/// recordings consistent across transport threads.
final class StubURLProtocol: URLProtocol, @unchecked Sendable {
    struct Recorded: Sendable {
        let method: String
        let path: String
        let headers: [String: String]
        let body: Data
    }

    private nonisolated(unsafe) static let state = State()

    /// Installs the handler (nil = every request fails) and clears previous
    /// recordings. Call at the start of each test.
    static func reset(handler: (@Sendable (URLRequest) throws -> StubResponse)? = nil) {
        state.reset(handler: handler)
    }

    static func recordedRequests() -> [Recorded] {
        state.recordedRequests()
    }

    /// The last request that hit the wire (tests drive one call at a time).
    static var lastRequest: Recorded? {
        state.recordedRequests().last
    }

    override static func canInit(with _: URLRequest) -> Bool {
        true
    }

    override static func canonicalRequest(for request: URLRequest) -> URLRequest {
        request
    }

    override func startLoading() {
        let recorded = Recorded(
            method: request.httpMethod ?? "GET",
            path: request.url?.path ?? "",
            headers: request.allHTTPHeaderFields ?? [:],
            body: Self.bodyData(of: request)
        )
        Self.state.record(recorded)
        do {
            let stub = try Self.state.answer(for: request)
            guard let url = request.url else {
                client?.urlProtocol(self, didFailWithError: URLError(.badURL))
                return
            }
            guard let response = HTTPURLResponse(
                url: url,
                statusCode: stub.status,
                httpVersion: nil,
                headerFields: stub.headers
            ) else {
                client?.urlProtocol(self, didFailWithError: URLError(.badServerResponse))
                return
            }
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            if !stub.body.isEmpty {
                client?.urlProtocol(self, didLoad: stub.body)
            }
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    override func stopLoading() {
        // Nothing to tear down: the client was already signalled above.
    }

    /// URLSession hands the body over as a stream; drain it back to Data so
    /// the tests can assert the exact bytes on the wire.
    private static func bodyData(of request: URLRequest) -> Data {
        if let body = request.httpBody {
            return body
        }
        guard let stream = request.httpBodyStream else {
            return Data()
        }
        stream.open()
        defer {
            stream.close()
        }
        var data = Data()
        let bufferSize = 4096
        let buffer = UnsafeMutablePointer<UInt8>.allocate(capacity: bufferSize)
        defer {
            buffer.deallocate()
        }
        while stream.hasBytesAvailable {
            let read = stream.read(buffer, maxLength: bufferSize)
            guard read > 0 else {
                break
            }
            data.append(buffer, count: read)
        }
        return data
    }

    private final class State: @unchecked Sendable {
        private let lock = NSLock()
        private var handler: (@Sendable (URLRequest) throws -> StubResponse)?
        private var recorded: [Recorded] = []

        func reset(handler: (@Sendable (URLRequest) throws -> StubResponse)?) {
            lock.withLock {
                self.handler = handler
                recorded = []
            }
        }

        func record(_ request: Recorded) {
            lock.withLock {
                recorded.append(request)
            }
        }

        func recordedRequests() -> [Recorded] {
            lock.withLock { recorded }
        }

        func answer(for request: URLRequest) throws -> StubResponse {
            let current = lock.withLock { handler }
            guard let current else {
                throw URLError(.unsupportedURL)
            }
            return try current(request)
        }
    }
}
