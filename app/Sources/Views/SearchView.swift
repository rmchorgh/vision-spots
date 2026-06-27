import SwiftUI

// MARK: - Search across tracks, albums, playlists

struct SearchView: View {
    @Environment(AppModel.self) private var appModel
    @Environment(PlayerModel.self) private var player

    @State private var query = ""
    @State private var results = SearchResults()
    @State private var isSearching = false
    @State private var searchTask: Task<Void, Never>?

    private let columns = [GridItem(.adaptive(minimum: 160, maximum: 200), spacing: 18)]

    var body: some View {
        ScrollView {
            if isSearching {
                ProgressView().controlSize(.large).frame(maxWidth: .infinity, minHeight: 240)
            } else if query.isEmpty {
                ContentUnavailableView("Search Spotify", systemImage: "magnifyingglass",
                                       description: Text("Find tracks, albums, and playlists."))
                    .frame(minHeight: 240)
            } else if results.isEmpty {
                ContentUnavailableView.search(text: query).frame(minHeight: 240)
            } else {
                resultsView
            }
        }
        .navigationTitle("Search")
        .navigationBarTitleDisplayMode(.inline)
        .searchable(text: $query, prompt: "Songs, albums, playlists")
        .onChange(of: query) { _, newValue in scheduleSearch(newValue) }
    }

    private var resultsView: some View {
        VStack(alignment: .leading, spacing: 28) {
            if !results.tracks.isEmpty {
                Text("Songs").font(.title2.weight(.bold))
                VStack(spacing: 0) {
                    ForEach(results.tracks) { track in
                        TrackRow(track: track)
                            .onTapGesture { Task { await player.play(contextURI: track.uri) } }
                        Divider().opacity(track.id == results.tracks.last?.id ? 0 : 1)
                    }
                }
                .padding(16)
                .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
            }

            if !results.playlists.isEmpty {
                Text("Playlists").font(.title2.weight(.bold))
                LazyVGrid(columns: columns, spacing: 18) {
                    ForEach(results.playlists) { MediaCard(item: .playlist($0)) }
                }
            }

            if !results.albums.isEmpty {
                Text("Albums").font(.title2.weight(.bold))
                LazyVGrid(columns: columns, spacing: 18) {
                    ForEach(results.albums) { MediaCard(item: .album($0)) }
                }
            }
        }
        .padding(28)
    }

    /// Debounce: cancel the in-flight search and wait briefly before querying.
    private func scheduleSearch(_ text: String) {
        searchTask?.cancel()
        let trimmed = text.trimmingCharacters(in: .whitespaces)
        guard !trimmed.isEmpty else { results = SearchResults(); isSearching = false; return }
        searchTask = Task {
            try? await Task.sleep(nanoseconds: 350_000_000)
            guard !Task.isCancelled else { return }
            isSearching = true
            defer { isSearching = false }
            do { results = try await appModel.service.search(query: trimmed) }
            catch { results = SearchResults() }
        }
    }
}
