package state

import (
	"errors"
	"testing"

	"github.com/Resinat/Resin/internal/model"
)

func newTestPublication(id, name, token string) model.SubscriptionPublication {
	return model.SubscriptionPublication{
		ID:            id,
		Name:          name,
		Enabled:       true,
		RegexFilters:  []string{"HK"},
		RegionFilters: []string{"hk"},
		HealthyOnly:   false,
		Token:         token,
		CreatedAtNs:   100,
		UpdatedAtNs:   100,
	}
}

func TestSubscriptionPublicationCRUD(t *testing.T) {
	repo := newTestStateRepo(t)
	p := newTestPublication("pub-1", "ClashHK", "token-1")

	if err := repo.UpsertSubscriptionPublication(p); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetSubscriptionPublication(p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != p.Name || got.Token != p.Token || !got.Enabled {
		t.Fatalf("got = %+v", got)
	}
	if len(got.RegexFilters) != 1 || got.RegexFilters[0] != "HK" {
		t.Fatalf("regex filters = %v", got.RegexFilters)
	}
	if len(got.RegionFilters) != 1 || got.RegionFilters[0] != "hk" {
		t.Fatalf("region filters = %v", got.RegionFilters)
	}

	p.Name = "ClashHKUpdated"
	p.Token = "token-2"
	p.UpdatedAtNs = 200
	if err := repo.UpsertSubscriptionPublication(p); err != nil {
		t.Fatalf("Upsert update: %v", err)
	}
	got, err = repo.GetSubscriptionPublication(p.ID)
	if err != nil {
		t.Fatalf("Get updated: %v", err)
	}
	if got.Name != p.Name || got.Token != p.Token || got.UpdatedAtNs != 200 {
		t.Fatalf("updated = %+v", got)
	}

	list, err := repo.ListSubscriptionPublications()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list length = %d", len(list))
	}

	if err := repo.DeleteSubscriptionPublication(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetSubscriptionPublication(p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete err = %v, want ErrNotFound", err)
	}
	if err := repo.DeleteSubscriptionPublication(p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete missing err = %v, want ErrNotFound", err)
	}
}

func TestSubscriptionPublicationConflicts(t *testing.T) {
	repo := newTestStateRepo(t)
	if err := repo.UpsertSubscriptionPublication(newTestPublication("pub-1", "NameA", "token-a")); err != nil {
		t.Fatalf("Upsert first: %v", err)
	}

	// 名称冲突
	if err := repo.UpsertSubscriptionPublication(newTestPublication("pub-2", "NameA", "token-b")); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate name err = %v, want ErrConflict", err)
	}
	// token 冲突
	if err := repo.UpsertSubscriptionPublication(newTestPublication("pub-3", "NameB", "token-a")); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate token err = %v, want ErrConflict", err)
	}
}

func TestSubscriptionPublicationValidation(t *testing.T) {
	repo := newTestStateRepo(t)

	cases := []model.SubscriptionPublication{
		newTestPublication("pub-1", "bad name", "token-a"),
		newTestPublication("pub-2", "NameB", ""),
		newTestPublication("pub-3", "NameC", "token-c"),
	}
	cases[2].RegexFilters = []string{"("}

	for _, p := range cases {
		if err := repo.UpsertSubscriptionPublication(p); err == nil {
			t.Fatalf("Upsert(%+v) expected error", p)
		}
	}
}
