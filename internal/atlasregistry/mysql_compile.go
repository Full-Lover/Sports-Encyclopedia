package atlasregistry

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

var (
	ErrInvalidCompileRequest = errors.New("invalid registry compile request")
	ErrBaselineMissing       = errors.New("registry baseline not found")
)

var _ Registry = (*MySQLRegistry)(nil)

func (registry *MySQLRegistry) CompilePublicationContent(ctx context.Context, request CompileRequest) (RegistryCompilation, error) {
	if registry == nil || registry.db == nil || validateCompileRequest(request) != nil {
		return RegistryCompilation{}, ErrInvalidCompileRequest
	}
	if err := registry.sealRun(ctx, request.RunID); err != nil {
		return RegistryCompilation{}, err
	}
	tx, err := registry.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return RegistryCompilation{}, err
	}
	defer tx.Rollback()
	var state string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM registry_runs WHERE run_id = ? FOR UPDATE`,
		request.RunID).Scan(&state); err != nil {
		return RegistryCompilation{}, err
	}
	if state == "ABANDONED" {
		return RegistryCompilation{}, ErrRunAbandoned
	}
	if state != "SEALED" {
		return RegistryCompilation{}, ErrRunSealed
	}
	var stored []byte
	var storedRequestedAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT compilation, requested_at FROM registry_compilations
		WHERE run_id = ? AND baseline_token = ? AND profile = ? AND configuration_fingerprint = ?
		FOR UPDATE`, request.RunID, request.BaselineToken, request.Profile,
		request.ConfigurationFingerprint).Scan(&stored, &storedRequestedAt)
	if err == nil {
		var existing RegistryCompilation
		if err := json.Unmarshal(stored, &existing); err != nil {
			return RegistryCompilation{}, fmt.Errorf("decode existing registry compilation: %w", err)
		}
		if !sameFailedGroups(existing.FailedGroups, request.FailedGroups) ||
			!storedRequestedAt.UTC().Equal(request.RequestedAt.UTC().Truncate(time.Microsecond)) {
			return RegistryCompilation{}, ErrRevisionConflict
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return RegistryCompilation{}, err
	}
	var baseline CompiledRegistryContent
	if request.BaselineToken != "" {
		var raw []byte
		err := tx.QueryRowContext(ctx, `SELECT content FROM registry_baselines
			WHERE baseline_token = ?`, request.BaselineToken).Scan(&raw)
		if errors.Is(err, sql.ErrNoRows) {
			return RegistryCompilation{}, ErrBaselineMissing
		}
		if err != nil {
			return RegistryCompilation{}, err
		}
		if err := json.Unmarshal(raw, &baseline); err != nil {
			return RegistryCompilation{}, fmt.Errorf("decode registry baseline: %w", err)
		}
		if err := filterBaselineFacts(ctx, tx, &baseline); err != nil {
			return RegistryCompilation{}, err
		}
	}
	facts, err := loadStagedFacts(ctx, tx, request.RunID)
	if err != nil {
		return RegistryCompilation{}, err
	}
	media, err := loadStagedMedia(ctx, tx, request.RunID)
	if err != nil {
		return RegistryCompilation{}, err
	}
	inherited, err := loadInheritedMedia(ctx, tx, baseline)
	if err != nil {
		return RegistryCompilation{}, err
	}
	media = append(media, inherited...)
	content, err := compileRegistryContent(request, baseline, facts, media)
	if err != nil {
		return RegistryCompilation{}, err
	}
	if err := reserveSlugs(ctx, tx, request.RunID, &content); err != nil {
		return RegistryCompilation{}, err
	}
	token, err := randomBaselineToken()
	if err != nil {
		return RegistryCompilation{}, err
	}
	compilation := RegistryCompilation{RunID: request.RunID, BaselineTokenUsed: request.BaselineToken,
		NextBaselineToken: token, Profile: request.Profile, SchemaVersion: 2,
		Content: content, FailedGroups: canonicalFailedGroups(request.FailedGroups)}
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return RegistryCompilation{}, err
	}
	compilationJSON, err := json.Marshal(compilation)
	if err != nil {
		return RegistryCompilation{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO registry_baselines
		(baseline_token, originating_run_id, content) VALUES (?, ?, ?)`, token,
		request.RunID, contentJSON); err != nil {
		return RegistryCompilation{}, fmt.Errorf("save registry baseline: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO registry_compilations
		(run_id, baseline_token, profile, configuration_fingerprint, next_baseline_token,
		requested_at, compilation) VALUES (?, ?, ?, ?, ?, ?, ?)`, request.RunID,
		request.BaselineToken, request.Profile, request.ConfigurationFingerprint,
		token, request.RequestedAt.UTC(), compilationJSON); err != nil {
		return RegistryCompilation{}, fmt.Errorf("save registry compilation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return RegistryCompilation{}, err
	}
	return compilation, nil
}

func (registry *MySQLRegistry) sealRun(ctx context.Context, runID string) error {
	tx, err := registry.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO registry_runs (run_id) VALUES (?)`, runID); err != nil {
		return err
	}
	var state string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM registry_runs WHERE run_id = ? FOR UPDATE`,
		runID).Scan(&state); err != nil {
		return err
	}
	if state == "ABANDONED" {
		return ErrRunAbandoned
	}
	if state == "OPEN" {
		if _, err := tx.ExecContext(ctx, `UPDATE registry_runs SET state = 'SEALED',
			sealed_at = UTC_TIMESTAMP(6) WHERE run_id = ?`, runID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func validateCompileRequest(request CompileRequest) error {
	if !validToken(request.RunID, 128) || request.RequestedAt.IsZero() ||
		(request.Profile != ProfilePreview && request.Profile != ProfileV1) ||
		!validHash(request.ConfigurationFingerprint) ||
		(request.BaselineToken != "" && !validHash(request.BaselineToken)) {
		return ErrInvalidCompileRequest
	}
	seen := make(map[[3]string]struct{}, len(request.FailedGroups))
	for _, failure := range request.FailedGroups {
		if !validToken(failure.TeamID, 64) || failure.FailedAt.IsZero() {
			return ErrInvalidCompileRequest
		}
		if _, err := PolicyForLeague(failure.League); err != nil {
			return ErrInvalidCompileRequest
		}
		switch failure.Group {
		case GroupIdentity, GroupVenue, GroupLeader, GroupRoster:
		default:
			return ErrInvalidCompileRequest
		}
		key := [3]string{string(failure.League), failure.TeamID, string(failure.Group)}
		if _, duplicate := seen[key]; duplicate {
			return ErrInvalidCompileRequest
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validHash(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func canonicalFailedGroups(input []FailedGroup) []FailedGroup {
	result := append([]FailedGroup(nil), input...)
	for i := range result {
		result[i].FailedAt = result[i].FailedAt.UTC()
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].League != result[j].League {
			return result[i].League < result[j].League
		}
		if result[i].TeamID != result[j].TeamID {
			return result[i].TeamID < result[j].TeamID
		}
		return result[i].Group < result[j].Group
	})
	return result
}

func sameFailedGroups(left, right []FailedGroup) bool {
	a, _ := json.Marshal(canonicalFailedGroups(left))
	b, _ := json.Marshal(canonicalFailedGroups(right))
	return string(a) == string(b)
}

func randomBaselineToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
