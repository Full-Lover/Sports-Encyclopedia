package publishedatlas

const PreviewSnapshotID SnapshotID = "preview-0001"

func PreviewMapDocument() MapDocument {
	visual := TeamVisual{
		Kind: TeamVisualAbbreviation,
		Text: "BOS",
		Alt:  "Boston Celtics abbreviation",
	}
	preview := TeamPreview{
		TeamID:     "nba-boston-celtics",
		TeamName:   "Boston Celtics",
		TeamVisual: visual,
		VenuePhoto: Photo{
			Kind: PhotoPlaceholder,
			Alt:  "TD Garden image unavailable",
		},
		VenueName: "TD Garden",
		League:    LeagueNBA,
		Actions: TeamPreviewActions{
			OfficialWebsiteURL: "https://www.nba.com/celtics/",
			SharePath:          "/teams/boston-celtics",
			DetailsPath:        "/teams/boston-celtics",
		},
	}
	place := MapPlace{
		VenueID:        "td-garden",
		AccessibleName: "TD Garden, home of Boston Celtics",
		// Team alignment: https://www.nba.com/news/faq
		Teams: []MapTeam{{
			TeamID:        preview.TeamID,
			Name:          preview.TeamName,
			League:        LeagueNBA,
			OfficialGroup: "Eastern Conference",
			Division:      "Atlantic Division",
			VenueName:     preview.VenueName,
			Visual:        visual,
			Preview:       preview,
		}},
	}
	place.Coordinates.Latitude = 42.366303
	place.Coordinates.Longitude = -71.062228

	return MapDocument{
		SnapshotID: PreviewSnapshotID,
		Leagues: []LeagueSummary{
			{Code: LeagueNBA, Name: "National Basketball Association", Path: "/leagues/nba"},
			{Code: LeagueNFL, Name: "National Football League", Path: "/leagues/nfl"},
			{Code: LeagueMLB, Name: "Major League Baseball", Path: "/leagues/mlb"},
			{Code: LeagueNHL, Name: "National Hockey League", Path: "/leagues/nhl"},
		},
		Places: []MapPlace{place},
	}
}
