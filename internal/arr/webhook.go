package arr

// WebhookPayload is a unified struct covering the import event payloads from
// Radarr, Sonarr, Lidarr, and Readarr. Fields unused by a given app are nil/empty.
type WebhookPayload struct {
	EventType   string              `json:"eventType"`
	MovieFile   *MovieFilePayload   `json:"movieFile,omitempty"`
	EpisodeFile *EpisodeFilePayload `json:"episodeFile,omitempty"`
	TrackFiles  []TrackFilePayload  `json:"trackFiles,omitempty"`
	BookFiles   []BookFilePayload   `json:"bookFiles,omitempty"`
}

type MovieFilePayload struct {
	Path string `json:"path"`
}

type EpisodeFilePayload struct {
	Path string `json:"path"`
}

type TrackFilePayload struct {
	Path string `json:"path"`
}

type BookFilePayload struct {
	Path string `json:"path"`
}

// ExtractFilePaths returns the final library file paths from a webhook payload.
// Returns nil when EventType is not "Download" — arr fires this event only after
// the file has been renamed and moved to its final library destination.
func ExtractFilePaths(payload WebhookPayload) []string {
	if payload.EventType != "Download" {
		return nil
	}

	var paths []string

	if payload.MovieFile != nil && payload.MovieFile.Path != "" {
		paths = append(paths, payload.MovieFile.Path)
	}
	if payload.EpisodeFile != nil && payload.EpisodeFile.Path != "" {
		paths = append(paths, payload.EpisodeFile.Path)
	}
	for _, tf := range payload.TrackFiles {
		if tf.Path != "" {
			paths = append(paths, tf.Path)
		}
	}
	for _, bf := range payload.BookFiles {
		if bf.Path != "" {
			paths = append(paths, bf.Path)
		}
	}

	return paths
}
