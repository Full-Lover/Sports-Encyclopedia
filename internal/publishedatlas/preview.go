package publishedatlas

const PreviewSnapshotID SnapshotID = "preview-0004"
const preview0003SnapshotID SnapshotID = "preview-0003"
const preview0002SnapshotID SnapshotID = "preview-0002"
const preview0001SnapshotID SnapshotID = "preview-0001"

func PreviewMapDocument() MapDocument {
	document := Preview0003MapDocument()
	document.SnapshotID = PreviewSnapshotID

	visual := TeamVisual{
		Kind: TeamVisualAbbreviation,
		Text: "SEA",
		Alt:  "Seattle Mariners abbreviation",
	}
	preview := TeamPreview{
		TeamID:     "mlb-seattle-mariners",
		TeamName:   "Seattle Mariners",
		TeamVisual: visual,
		VenuePhoto: Photo{
			Kind: PhotoPlaceholder,
			Alt:  "T-Mobile Park image unavailable",
		},
		VenueName:           "T-Mobile Park",
		RegularGameCapacity: 47943,
		OpenedYear:          1999,
		VenueFactsSourceURL: "https://www.mlb.com/mariners/history/ballparks",
		League:              LeagueMLB,
		Actions: TeamPreviewActions{
			OfficialWebsiteURL: "https://www.mlb.com/mariners",
			SharePath:          "/teams/seattle-mariners",
			DetailsPath:        "/teams/seattle-mariners",
		},
	}
	place := MapPlace{
		VenueID:        "t-mobile-park",
		AccessibleName: "T-Mobile Park, home of Seattle Mariners",
		// Team alignment: https://www.mlb.com/mariners
		Teams: []MapTeam{{
			TeamID:        preview.TeamID,
			Name:          preview.TeamName,
			League:        LeagueMLB,
			OfficialGroup: "American League",
			Division:      "American League West",
			VenueName:     preview.VenueName,
			Visual:        visual,
			Preview:       preview,
		}},
	}
	// Venue coordinates: OpenStreetMap-derived Mapcarta T-Mobile Park location.
	place.Coordinates.Latitude = 47.59093
	place.Coordinates.Longitude = -122.33263
	document.Places = append(document.Places, place)
	return document
}

func Preview0003MapDocument() MapDocument {
	document := Preview0002MapDocument()
	document.SnapshotID = preview0003SnapshotID

	visual := TeamVisual{
		Kind: TeamVisualAbbreviation,
		Text: "NYG",
		Alt:  "New York Giants abbreviation",
	}
	preview := TeamPreview{
		TeamID:     "nfl-new-york-giants",
		TeamName:   "New York Giants",
		TeamVisual: visual,
		VenuePhoto: Photo{
			Kind: PhotoPlaceholder,
			Alt:  "MetLife Stadium image unavailable",
		},
		VenueName:           "MetLife Stadium",
		RegularGameCapacity: 82500,
		OpenedYear:          2010,
		VenueFactsSourceURL: "https://www.metlifestadium.com/stadium/about-metlife-stadium",
		League:              LeagueNFL,
		Actions: TeamPreviewActions{
			OfficialWebsiteURL: "https://www.giants.com/",
			SharePath:          "/teams/new-york-giants",
			DetailsPath:        "/teams/new-york-giants",
		},
	}
	place := MapPlace{
		VenueID:        "metlife-stadium",
		AccessibleName: "MetLife Stadium, home of New York Giants",
		// Team alignment and venue: https://www.nfl.com/teams/new-york-giants/
		Teams: []MapTeam{{
			TeamID:        preview.TeamID,
			Name:          preview.TeamName,
			League:        LeagueNFL,
			OfficialGroup: "NFC",
			Division:      "NFC East",
			VenueName:     preview.VenueName,
			Visual:        visual,
			Preview:       preview,
		}},
	}
	// Venue coordinates: OpenStreetMap way 24221553.
	place.Coordinates.Latitude = 40.81352
	place.Coordinates.Longitude = -74.07435
	document.Places = append(document.Places, place)
	return document
}

func Preview0002MapDocument() MapDocument {
	document := Preview0001MapDocument()
	document.SnapshotID = preview0002SnapshotID
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
