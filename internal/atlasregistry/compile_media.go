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
		if logo, found := chosen[[2]string{string(MediaLogo), team.TeamID}]; found {
			team.Logo = selectedMedia(logo.batch, logo.rights)
		}
		if venue := displayValue(team.Venue); venue != nil {
			if photo, found := chosen[[2]string{string(MediaVenuePhoto), venue.VenueID}]; found {
				team.VenuePhoto = selectedMedia(photo.batch, photo.rights)
			}
		}
		if roster := displayValue(team.Roster); roster != nil {
			for _, entry := range roster.Entries {
				if photo, found := chosen[[2]string{string(MediaPlayerPhoto), entry.PersonID}]; found {
					team.PlayerPhotos = append(team.PlayerPhotos, *selectedMedia(photo.batch, photo.rights))
				}
			}
			sort.Slice(team.PlayerPhotos, func(a, b int) bool {
				return team.PlayerPhotos[a].EntityID < team.PlayerPhotos[b].EntityID
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
