package publishedatlas

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
)

// Read exposes only published snapshots. Staged candidates are never eligible.
func (store *MySQLPublicationStore) Read(ctx context.Context, request ReadRequest) (ReadResult, error) {
	if store == nil || store.db == nil {
		return ReadResult{}, &ReadFault{Code: FaultStorageUnavailable}
	}
	switch request.Kind {
	case ReadHome, ReadTeam:
		var active sql.NullString
		err := store.db.QueryRowContext(ctx, `SELECT snapshot_id FROM publication_active_pointer
			WHERE singleton_id = 1`).Scan(&active)
		if err != nil {
			return ReadResult{}, &ReadFault{Code: FaultStorageUnavailable}
		}
		if !active.Valid {
			return ReadResult{}, &ReadFault{Code: FaultNotPublished}
		}
		request.SnapshotID = SnapshotID(active.String)
	case ReadMap:
		if _, err := ParseSnapshotID(string(request.SnapshotID)); err != nil {
			return ReadResult{}, &ReadFault{Code: FaultInvalidRequest}
		}
	default:
		return ReadResult{}, &ReadFault{Code: FaultInvalidRequest}
	}
	var content []byte
	err := store.db.QueryRowContext(ctx, `SELECT map_document FROM publication_snapshots
		WHERE snapshot_id = ? AND status = 'PUBLISHED' AND schema_version = 1`, request.SnapshotID).Scan(&content)
	if errors.Is(err, sql.ErrNoRows) {
		if request.Kind != ReadMap {
			return ReadResult{}, &ReadFault{Code: FaultStorageUnavailable}
		}
		return ReadResult{}, &ReadFault{Code: FaultSnapshotMissing}
	}
	if err != nil {
		return ReadResult{}, &ReadFault{Code: FaultStorageUnavailable}
	}
	document, err := decodePreviewMapDocument(bytes.NewReader(content))
	if err != nil || document.SnapshotID != request.SnapshotID {
		return ReadResult{}, &ReadFault{Code: FaultStorageUnavailable}
	}
	if request.Kind == ReadHome {
		return ReadResult{Home: &HomeDocument{SnapshotID: document.SnapshotID}}, nil
	}
	if request.Kind == ReadTeam {
		request.SnapshotID = ""
	}
	return NewMemoryReader(document).Read(ctx, request)
}
