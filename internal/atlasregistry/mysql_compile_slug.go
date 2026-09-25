package atlasregistry

import (
	"context"
	"database/sql"
	"errors"
	"sort"
)

var ErrSlugConflict = errors.New("registry slug is reserved for another team")

func reserveSlugs(ctx context.Context, tx *sql.Tx, runID string, content *CompiledRegistryContent) error {
	byID := make(map[string]*CompiledTeam, len(content.Teams))
	for i := range content.Teams {
		team := &content.Teams[i]
		if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO registry_slug_history
			(slug, team_id, first_run_id) VALUES (?, ?, ?)`, team.Slug, team.TeamID, runID); err != nil {
			return err
		}
		var owner string
		if err := tx.QueryRowContext(ctx, `SELECT team_id FROM registry_slug_history
			WHERE slug = ? FOR UPDATE`, team.Slug).Scan(&owner); err != nil {
			return err
		}
		if owner != team.TeamID {
			return ErrSlugConflict
		}
		byID[team.TeamID] = team
	}
	rows, err := tx.QueryContext(ctx, `SELECT slug, team_id FROM registry_slug_history ORDER BY slug`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var slug, owner string
		if err := rows.Scan(&slug, &owner); err != nil {
			rows.Close()
			return err
		}
		if team, found := byID[owner]; found && slug != team.Slug {
			team.SlugHistory = append(team.SlugHistory, slug)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, team := range byID {
		sort.Strings(team.SlugHistory)
	}
	return nil
}
