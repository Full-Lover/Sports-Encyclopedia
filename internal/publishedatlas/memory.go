package publishedatlas

import (
	"context"
	"strings"
)

type memoryReader struct {
	active    SnapshotID
	documents map[SnapshotID]MapDocument
	teamPages map[SnapshotID]map[string]TeamPageDocument
}

func NewMemoryReader(documents ...MapDocument) Reader {
	stored := make(map[SnapshotID]MapDocument, len(documents))
	teamPages := make(map[SnapshotID]map[string]TeamPageDocument, len(documents))
	var active SnapshotID
	for _, document := range documents {
		if active == "" {
			active = document.SnapshotID
		}
		stored[document.SnapshotID] = cloneMapDocument(document)
		teamPages[document.SnapshotID] = make(map[string]TeamPageDocument)
		for _, place := range document.Places {
			for _, team := range place.Teams {
				path := team.Preview.Actions.DetailsPath
				if !strings.HasPrefix(path, "/teams/") || len(path) <= len("/teams/") {
					continue
				}
				teamPages[document.SnapshotID][strings.TrimPrefix(path, "/teams/")] = TeamPageDocument{
					SnapshotID:         document.SnapshotID,
					TeamID:             team.TeamID,
					Name:               team.Name,
					League:             team.League,
					OfficialGroup:      team.OfficialGroup,
					Division:           team.Division,
					VenueName:          team.VenueName,
					OfficialWebsiteURL: team.Preview.Actions.OfficialWebsiteURL,
					Preview:            true,
				}
			}
		}
	}
	return &memoryReader{active: active, documents: stored, teamPages: teamPages}
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
	if request.Kind == ReadTeam && request.SnapshotID == "" && request.Slug != "" {
		if reader.active == "" {
			return ReadResult{}, &ReadFault{Code: FaultNotPublished}
		}
		page, exists := reader.teamPages[reader.active][request.Slug]
		if !exists {
			return ReadResult{}, &ReadFault{Code: FaultTeamMissing}
		}
		return ReadResult{Team: &page}, nil
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
