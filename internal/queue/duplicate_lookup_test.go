package queue

import (
	"context"
	"maragu.dev/goqite"
	"testing"
	"time"
)

func TestCompleteReusedFileRecordsStatusOnInsert(t *testing.T) {
	q := newTestQueue(t)
	ctx := context.Background()
	job := &FileJob{Path: "/source/movie.mkv", Size: 3, CreatedAt: time.Now()}
	if err := q.CompleteReusedFile(ctx, goqite.ID("reused-item"), "/output/movie.nzb", job); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := q.DB().QueryRowContext(ctx, "SELECT verification_status FROM completed_items WHERE id = ?", "reused-item").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "reused" {
		t.Fatal(status)
	}
}
