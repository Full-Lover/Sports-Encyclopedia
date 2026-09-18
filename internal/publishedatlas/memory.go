package publishedatlas

import "context"

type memoryReader struct {
	documents map[SnapshotID]MapDocument
}

func NewMemoryReader(documents ...MapDocument) Reader {
	stored := make(map[SnapshotID]MapDocument, len(documents))
	for _, document := range documents {
		stored[document.SnapshotID] = cloneMapDocument(document)
	}
	return &memoryReader{documents: stored}
}

func (reader *memoryReader) Read(ctx context.Context, request ReadRequest) (ReadResult, error) {
	if err := ctx.Err(); err != nil {
		return ReadResult{}, err
	}
	if request.Kind != ReadMap || request.SnapshotID == "" {
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
