package publishedatlas

const PreviewSnapshotID SnapshotID = "preview-0002"
const preview0001SnapshotID SnapshotID = "preview-0001"

func PreviewMapDocument() MapDocument {
	document := Preview0001MapDocument()
	document.SnapshotID = PreviewSnapshotID
	place := &document.Places[0]
	place.AccessibleName = "TD Garden, home of Boston Celtics and Boston Bruins"

	visual := TeamVisual{
		Kind: TeamVisualAbbreviation,
		Text: "BOS",
		Alt:  "Boston Bruins abbreviation",
	}
	preview := TeamPreview{
		TeamID:     "nhl-boston-bruins",
		TeamName:   "Boston Bruins",
		TeamVisual: visual,
		VenuePhoto: Photo{
			Kind: PhotoPlaceholder,
			Alt:  "TD Garden image unavailable",
		},
		VenueName:           "TD Garden",
		RegularGameCapacity: 17850,
		OpenedYear:          1995,
		VenueFactsSourceURL: "https://www.tdgarden.com/about-td-garden",
		League:              LeagueNHL,
		Actions: TeamPreviewActions{
			OfficialWebsiteURL: "https://www.nhl.com/bruins/",
			SharePath:          "/teams/boston-bruins",
			DetailsPath:        "/teams/boston-bruins",
		},
	}
	// Team alignment: https://www.nhl.com/info/teams/
	place.Teams = append(place.Teams, MapTeam{
		TeamID:        preview.TeamID,
		Name:          preview.TeamName,
		League:        LeagueNHL,
		OfficialGroup: "Eastern Conference",
		Division:      "Atlantic Division",
		VenueName:     preview.VenueName,
		Visual:        visual,
		Preview:       preview,
	})
	return document
}

func Preview0001MapDocument() MapDocument {
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
		VenueName:           "TD Garden",
		RegularGameCapacity: 19156,
		OpenedYear:          1995,
		VenueFactsSourceURL: "https://www.tdgarden.com/about-td-garden",
		League:              LeagueNBA,
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
		SnapshotID: preview0001SnapshotID,
		Leagues: []LeagueSummary{
			{Code: LeagueNBA, Name: "National Basketball Association", Path: "/leagues/nba"},
			{Code: LeagueNFL, Name: "National Football League", Path: "/leagues/nfl"},
			{Code: LeagueMLB, Name: "Major League Baseball", Path: "/leagues/mlb"},
			{Code: LeagueNHL, Name: "National Hockey League", Path: "/leagues/nhl"},
		},
		Places: []MapPlace{place},
	}
}
