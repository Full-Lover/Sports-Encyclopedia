package atlasregistry

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMySQLRegistryUsagePolicy(t *testing.T) {
	db := openRegistryTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ApplyRegistryMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	registry := NewMySQLRegistry(db)
	policy := validFactPolicy()
	policy.SourceID = fmt.Sprintf("policy-test-%d", time.Now().UnixNano())
	if err := registry.InstallPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	batch := validIdentityBatch()
	batch.SourceID = policy.SourceID
	stored, err := loadUsagePolicy(ctx, db, policy.SourceID, policy.CapabilityKey)
	if err != nil || !stored.AllowWebsite || !policyAllowsFact(stored, batch) {
		t.Fatalf("installed policy not usable: %#v, %v", stored, err)
	}
	if policyAllowsFact(stored, validIdentityBatch()) {
		t.Fatal("policy was borrowed by another source")
	}
	policy.Active = false
	if err := registry.InstallPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	stored, err = loadUsagePolicy(ctx, db, policy.SourceID, policy.CapabilityKey)
	if err != nil || policyAllowsFact(stored, batch) {
		t.Fatalf("revoked policy still permits facts: %#v, %v", stored, err)
	}
}
