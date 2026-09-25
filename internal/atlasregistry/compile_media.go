package atlasregistry

import "sort"

func attachMedia(content *CompiledRegistryContent, media []stagedMedia) {
	type choice struct {
		batch  MediaAssetBatch
		rights MediaRightsOption
	}
	chosen := make(map[[2]string]choice)
	for _, observation := range media {
		batch := observation.Batch
		if !observation.Policy.AllowWebsite || !policyAllowsMedia(observation.Policy, batch) ||
			observation.ApprovedRights == nil {
			continue
		}
		option := *observation.ApprovedRights
		key := [2]string{string(batch.Kind), batch.EntityID}
		old, exists := chosen[key]
		if !exists || batch.FetchedAt.After(old.batch.FetchedAt) ||
			(batch.FetchedAt.Equal(old.batch.FetchedAt) && batch.AssetID < old.batch.AssetID) {
			chosen[key] = choice{batch: batch, rights: option}
		}
	}
	for i := range content.Teams {
		team := &content.Teams[i]
		identity := displayValue(team.Identity)
		team.Visual = TeamVisual{Kind: TeamVisualAbbreviation}
		if identity != nil {
			team.Visual.Abbreviation = identity.OfficialAbbreviation
		}
		if logo, found := chosen[[2]string{string(MediaLogo), team.TeamID}]; found {
			team.Visual = TeamVisual{Kind: TeamVisualMedia, Media: selectedMedia(logo.batch, logo.rights)}
		}
		team.VenuePhoto = PhotoSelection{Kind: PhotoPlaceholder}
		if venue := displayValue(team.Venue); venue != nil {
			if photo, found := chosen[[2]string{string(MediaVenuePhoto), venue.VenueID}]; found {
				team.VenuePhoto = PhotoSelection{Kind: PhotoReusableMedia, Media: selectedMedia(photo.batch, photo.rights)}
			}
		}
		team.PlayerPhotos = nil
		if roster := displayValue(team.Roster); roster != nil {
			for _, entry := range roster.Entries {
				selection := PlayerPhotoSelection{PersonID: entry.PersonID,
					Photo: PhotoSelection{Kind: PhotoPlaceholder}}
				if photo, found := chosen[[2]string{string(MediaPlayerPhoto), entry.PersonID}]; found {
					selection.Photo = PhotoSelection{Kind: PhotoReusableMedia, Media: selectedMedia(photo.batch, photo.rights)}
				}
				team.PlayerPhotos = append(team.PlayerPhotos, selection)
			}
			sort.Slice(team.PlayerPhotos, func(a, b int) bool {
				return team.PlayerPhotos[a].PersonID < team.PlayerPhotos[b].PersonID
			})
		}
	}
}

func selectedMedia(batch MediaAssetBatch, rights MediaRightsOption) *SelectedMedia {
	return &SelectedMedia{SourceID: batch.SourceID, CapabilityKey: batch.CapabilityKey,
		AssetID: batch.AssetID, Kind: batch.Kind, EntityID: batch.EntityID,
		FileURL: batch.FileURL, SourcePageURL: batch.SourcePageURL,
		ContentHash: batch.ContentHash, FetchedAt: batch.FetchedAt, Rights: rights}
}
