package publishedatlas

import "context"

type memoryReader struct {
	active    SnapshotID
	documents map[SnapshotID]MapDocument
}

func NewMemoryReader(documents ...MapDocument) Reader {
	stored := make(map[SnapshotID]MapDocument, len(documents))
	var active SnapshotID
	for _, document := range documents {
		if active == "" {
			active = document.SnapshotID
		}
		stored[document.SnapshotID] = cloneMapDocument(document)
	}
	return &memoryReader{active: active, documents: stored}
}

func (reader *memoryReader) Read(ctx context.Context, request ReadRequest) (ReadResult, error) {
	if err := ctx.Err(); err != nil {
		return ReadResult{}, err
	}
	if request.Kind == ReadHome && request.SnapshotID == "" {
		if reader.active == "" {
			return ReadResult{}, &ReadFault{Code: FaultNotPublished}
		}
		return ReadResult{Home: &HomeDocument{SnapshotID: reader.active}}, nil
	}
	if _, err := ParseSnapshotID(string(request.SnapshotID)); request.Kind != ReadMap || err != nil {
		return ReadResult{}, &ReadFault{Code: FaultInvalidRequest}
	}

	document, exists := reader.documents[request.SnapshotID]
	if !exists {
		return ReadResult{}, &ReadFault{Code: FaultSnapshotMissing}
	}

	result := cloneMapDocument(document)
	return ReadResult{Map: &result}, nil
}

func cloneMapDocument(document MapDocument) MapDocument {
	cloned := document
	cloned.Leagues = append([]LeagueSummary(nil), document.Leagues...)
	cloned.Places = make([]MapPlace, len(document.Places))
	for placeIndex, place := range document.Places {
		cloned.Places[placeIndex] = place
		cloned.Places[placeIndex].Teams = append([]MapTeam(nil), place.Teams...)
	}
	return cloned
}
