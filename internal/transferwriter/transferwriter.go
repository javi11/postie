// Package transferwriter records a durable manifest and a transfer_files row
// for each file of a transfer before its articles are posted. It bridges the
// poster (which builds articles with their Message-IDs) to the manifest store
// and transfer persistence, so an interrupted upload can be resumed and
// verified using the exact Message-IDs and headers that were posted.
package transferwriter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/javi11/postie/internal/article"
	"github.com/javi11/postie/internal/manifest"
	"github.com/javi11/postie/internal/par2"
	"github.com/javi11/postie/internal/transferstore"
)

// Recorder writes manifests + transfer_files rows for one transfer. It is
// created per job (bound to a transfer_id) and shares the process-wide store.
type Recorder struct {
	transferID string
	baseDir    string
	store      *transferstore.Store
}

// New creates a Recorder for transferID that writes manifests under baseDir and
// persists rows through store.
func New(transferID, baseDir string, store *transferstore.Store) *Recorder {
	return &Recorder{transferID: transferID, baseDir: baseDir, store: store}
}

// fileID derives a stable identifier for a source path so re-recording the same
// file (e.g. after a retry or crash) maps to the same manifest and row.
func fileID(sourcePath string) string {
	sum := sha256.Sum256([]byte(sourcePath))
	return hex.EncodeToString(sum[:8])
}

// roleFor classifies a file by its on-disk path.
func roleFor(sourcePath string) manifest.FileRole {
	if par2.IsPar2File(sourcePath) {
		return manifest.RoleGeneratedPar2
	}
	return manifest.RoleOriginal
}

// RecordFile writes an immutable manifest of articles for sourcePath (via a
// temp file + atomic rename) and upserts the corresponding transfer_files row
// in the planned state. It must be called before the file's articles are
// posted so the manifest is durable first.
func (r *Recorder) RecordFile(ctx context.Context, sourcePath string, articles []*article.Article) error {
	fid := fileID(sourcePath)
	role := roleFor(sourcePath)
	manifestPath := manifest.FilePath(r.baseDir, r.transferID, fid)

	w, err := manifest.NewWriter(manifestPath)
	if err != nil {
		return err
	}
	for i, a := range articles {
		if err := w.Write(manifest.RecordFromArticle(i, sourcePath, role, a)); err != nil {
			_ = w.Abort()
			return err
		}
	}
	if err := w.Commit(); err != nil {
		return err
	}

	return r.store.UpsertFile(ctx, transferstore.TransferFile{
		TransferID:        r.transferID,
		FileID:            fid,
		ManifestPath:      manifestPath,
		ManifestVersion:   manifest.Version,
		SourcePath:        sourcePath,
		FileRole:          string(role),
		ArticleCount:      len(articles),
		UploadState:       transferstore.StatePlanned,
		VerificationState: transferstore.StatePlanned,
	})
}
