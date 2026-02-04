package selector

import (
	"context"
	"time"

	"github.com/go-gost/core/metadata"
	"github.com/go-gost/core/selector"
	mdutil "github.com/go-gost/x/metadata/util"
)

type failFilter[T any] struct {
	maxFails    int
	failTimeout time.Duration
}

// FailFilter filters the dead objects.
// An object is marked as dead if its failed count is greater than MaxFails.
func FailFilter[T any](maxFails int, timeout time.Duration) selector.Filter[T] {
	return &failFilter[T]{
		maxFails:    maxFails,
		failTimeout: timeout,
	}
}

// Filter filters dead objects.
// For single-node case, skip filtering to ensure availability (matches upstream).
// For multi-node case, filter out failed nodes to enable failover.
func (f *failFilter[T]) Filter(ctx context.Context, vs ...T) []T {
	if len(vs) <= 1 {
		return vs
	}
	var l []T
	for _, v := range vs {
		maxFails := f.maxFails
		failTimeout := f.failTimeout
		if mi, _ := any(v).(metadata.Metadatable); mi != nil {
			if md := mi.Metadata(); md != nil {
				if md.IsExists(labelMaxFails) {
					maxFails = mdutil.GetInt(md, labelMaxFails)
				}
				if md.IsExists(labelFailTimeout) {
					failTimeout = mdutil.GetDuration(md, labelFailTimeout)
				}
			}
		}
		if maxFails <= 0 {
			maxFails = 1
		}
		if failTimeout <= 0 {
			failTimeout = DefaultFailTimeout
		}

		if mi, _ := any(v).(selector.Markable); mi != nil {
			if marker := mi.Marker(); marker != nil {
				if marker.Count() < int64(maxFails) ||
					time.Since(marker.Time()) >= failTimeout {
					l = append(l, v)
				}
				continue
			}
		}
		l = append(l, v)
	}
	return l
}

type backupFilter[T any] struct{}

// BackupFilter filters the backup objects.
// An object is marked as backup if its metadata has backup flag.
func BackupFilter[T any]() selector.Filter[T] {
	return &backupFilter[T]{}
}

// Filter filters backup objects.
func (f *backupFilter[T]) Filter(ctx context.Context, vs ...T) []T {
	if len(vs) <= 1 {
		return vs
	}

	var l, backups []T
	for _, v := range vs {
		if mi, _ := any(v).(metadata.Metadatable); mi != nil {
			if mdutil.GetBool(mi.Metadata(), labelBackup) {
				backups = append(backups, v)
				continue
			}
		}
		l = append(l, v)
	}

	if len(l) == 0 {
		return backups
	}
	return l
}
