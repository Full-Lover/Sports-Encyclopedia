package atlasregistry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type MySQLRegistry struct {
	db *sql.DB
}

func NewMySQLRegistry(db *sql.DB) *MySQLRegistry {
	return &MySQLRegistry{db: db}
}

var ErrPolicyNotAllowed = errors.New("registry source capability is not allowed")

// InstallPolicy is a trusted curation/deployment operation. It is not part of
// the AtlasRefresh-facing Registry interface and is never called from Web.
func (registry *MySQLRegistry) InstallPolicy(ctx context.Context, policy UsagePolicy) error {
	if registry == nil || registry.db == nil {
		return errors.New("registry database is nil")
	}
	if err := ValidateUsagePolicy(policy); err != nil {
		return err
	}
	factGroups, err := json.Marshal(policy.FactGroups)
	if err != nil {
		return err
	}
	mediaKinds, err := json.Marshal(policy.MediaKinds)
	if err != nil {
		return err
	}
	var league any
	if policy.Kind == CapabilityFacts {
		league = policy.League
	}
	_, err = registry.db.ExecContext(ctx, `INSERT INTO registry_source_capabilities
		(source_id, capability_key, capability_kind, league, fact_groups, media_kinds,
		official_authority, independence_key, allow_website, allow_repository,
		allow_public_api, policy_source_url, reviewed_by, reviewed_at, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE capability_kind = VALUES(capability_kind), league = VALUES(league),
		fact_groups = VALUES(fact_groups), media_kinds = VALUES(media_kinds),
		official_authority = VALUES(official_authority), independence_key = VALUES(independence_key),
		allow_website = VALUES(allow_website), allow_repository = VALUES(allow_repository),
		allow_public_api = VALUES(allow_public_api), policy_source_url = VALUES(policy_source_url),
		reviewed_by = VALUES(reviewed_by), reviewed_at = VALUES(reviewed_at), active = VALUES(active)`,
		policy.SourceID, policy.CapabilityKey, policy.Kind, league, factGroups, mediaKinds,
		policy.OfficialAuthority, policy.IndependenceKey, policy.AllowWebsite,
		policy.AllowRepository, policy.AllowPublicAPI, policy.EvidenceURL,
		policy.ReviewedBy, policy.ReviewedAt.UTC(), policy.Active)
	if err != nil {
		return fmt.Errorf("install registry source policy: %w", err)
	}
	return nil
}

type policyQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func loadUsagePolicy(ctx context.Context, query policyQuerier, sourceID, capabilityKey string) (UsagePolicy, error) {
	var policy UsagePolicy
	var league sql.NullString
	var groupsJSON, kindsJSON []byte
	err := query.QueryRowContext(ctx, `SELECT capability_kind, league, fact_groups, media_kinds,
		official_authority, independence_key, allow_website, allow_repository,
		allow_public_api, policy_source_url, reviewed_by, reviewed_at, active
		FROM registry_source_capabilities WHERE source_id = ? AND capability_key = ? FOR UPDATE`,
		sourceID, capabilityKey).Scan(&policy.Kind, &league, &groupsJSON, &kindsJSON,
		&policy.OfficialAuthority, &policy.IndependenceKey, &policy.AllowWebsite,
		&policy.AllowRepository, &policy.AllowPublicAPI, &policy.EvidenceURL,
		&policy.ReviewedBy, &policy.ReviewedAt, &policy.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return UsagePolicy{}, ErrPolicyNotAllowed
	}
	if err != nil {
		return UsagePolicy{}, fmt.Errorf("read registry source policy: %w", err)
	}
	policy.SourceID, policy.CapabilityKey = sourceID, capabilityKey
	policy.League = LeagueCode(league.String)
	if err := json.Unmarshal(groupsJSON, &policy.FactGroups); err != nil {
		return UsagePolicy{}, fmt.Errorf("decode fact policy groups: %w", err)
	}
	if err := json.Unmarshal(kindsJSON, &policy.MediaKinds); err != nil {
		return UsagePolicy{}, fmt.Errorf("decode media policy kinds: %w", err)
	}
	if err := ValidateUsagePolicy(policy); err != nil {
		return UsagePolicy{}, err
	}
	return policy, nil
}

func policyAllowsFact(policy UsagePolicy, batch FactBatch) bool {
	metadata := batch.metadata()
	if !policy.Active || policy.Kind != CapabilityFacts || policy.League != metadata.League ||
		policy.SourceID != metadata.SourceID || policy.CapabilityKey != metadata.CapabilityKey {
		return false
	}
	for _, group := range policy.FactGroups {
		if group == batch.factGroup() {
			return true
		}
	}
	return false
}

func policyAllowsMedia(policy UsagePolicy, batch MediaAssetBatch) bool {
	if !policy.Active || policy.Kind != CapabilityMedia ||
		policy.SourceID != batch.SourceID || policy.CapabilityKey != batch.CapabilityKey {
		return false
	}
	for _, kind := range policy.MediaKinds {
		if kind == batch.Kind {
			return true
		}
	}
	return false
}
