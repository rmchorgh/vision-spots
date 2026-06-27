import Foundation
import Security

// MARK: - Session token storage
//
// The app holds only the short-lived vision-spots *session JWT* (never raw Spotify tokens —
// those stay on the backend, per api-contract.md). Stored in the Keychain so it survives
// relaunch. Owned by the `ui` agent; consumed by LiveSpotifyService + AuthController.

actor KeychainStore {
    static let shared = KeychainStore()

    private let service = "org.richardmch.visionspots"
    private let account = "spotify-session-jwt"

    /// Cached for synchronous-ish reads; the Keychain is the source of truth.
    private(set) lazy var sessionToken: String? = readToken()

    func setSessionToken(_ token: String?) {
        sessionToken = token
        if let token { writeToken(token) } else { deleteToken() }
    }

    var isSignedIn: Bool { sessionToken != nil }

    // MARK: Keychain primitives

    private func baseQuery() -> [String: Any] {
        [kSecClass as String: kSecClassGenericPassword,
         kSecAttrService as String: service,
         kSecAttrAccount as String: account]
    }

    private func readToken() -> String? {
        var query = baseQuery()
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var item: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &item) == errSecSuccess,
              let data = item as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    private func writeToken(_ token: String) {
        let data = Data(token.utf8)
        deleteToken()
        var query = baseQuery()
        query[kSecValueData as String] = data
        SecItemAdd(query as CFDictionary, nil)
    }

    private func deleteToken() {
        SecItemDelete(baseQuery() as CFDictionary)
    }
}
