package tools

import (
	"encoding/json"
	"testing"
)

func boolPtr(b bool) *bool        { return &b }
func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }
func strPtr(s string) *string     { return &s }

func TestBuildCollectionPayload_OnlyExplicitFieldsSet(t *testing.T) {
	payload := buildCollectionPayload(266192, collectionStatusUpdate{Own: boolPtr(true)})

	if payload.ObjectID != 266192 {
		t.Errorf("ObjectID = %d, want 266192", payload.ObjectID)
	}
	if payload.ObjectType != "thing" || payload.Action != "additem" || payload.AJAX != "1" || payload.QuickAdd != "1" {
		t.Errorf("base fields not set as expected: %+v", payload)
	}
	if payload.AddOwned != "true" {
		t.Errorf("AddOwned = %q, want %q", payload.AddOwned, "true")
	}
	if payload.AddWant != "" {
		t.Errorf("AddWant should be empty (unset) when not requested, got %q", payload.AddWant)
	}
}

func TestBuildCollectionPayload_OmitsUnsetFieldsFromJSON(t *testing.T) {
	payload := buildCollectionPayload(1, collectionStatusUpdate{Own: boolPtr(false)})

	out, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if _, present := raw["addwant"]; present {
		t.Errorf("addwant should be omitted from JSON when not requested, got raw=%v", raw)
	}
	if v, ok := raw["addowned"]; !ok || v != "false" {
		t.Errorf("addowned = %v, want %q", v, "false")
	}
}

func TestBuildCollectionPayload_RatingAndComment(t *testing.T) {
	payload := buildCollectionPayload(1, collectionStatusUpdate{
		Rating:  floatPtr(7.5),
		Comment: strPtr("great game"),
	})

	if payload.Rating != "7.5" {
		t.Errorf("Rating = %q, want %q", payload.Rating, "7.5")
	}
	if payload.Comment != "great game" {
		t.Errorf("Comment = %q, want %q", payload.Comment, "great game")
	}
}

func TestBuildCollectionPayload_WishlistPriority(t *testing.T) {
	payload := buildCollectionPayload(1, collectionStatusUpdate{WishlistPriority: intPtr(3)})
	if payload.WishlistPriority != 3 {
		t.Errorf("WishlistPriority = %d, want 3", payload.WishlistPriority)
	}
}

func TestBoolToBGGFlag(t *testing.T) {
	if boolToBGGFlag(true) != "true" {
		t.Error("boolToBGGFlag(true) should be \"true\"")
	}
	if boolToBGGFlag(false) != "false" {
		t.Error("boolToBGGFlag(false) should be \"false\"")
	}
}

func TestDeleteCollectionItemPayload(t *testing.T) {
	payload := deleteCollectionItemPayload(42)
	if payload.Action != "deleteitem" || payload.ObjectID != 42 {
		t.Errorf("unexpected delete payload: %+v", payload)
	}
}

func TestStatusToCollectionUpdate(t *testing.T) {
	t.Run("own", func(t *testing.T) {
		u, err := statusToCollectionUpdate("own", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Own == nil || !*u.Own {
			t.Errorf("Own not set to true: %+v", u)
		}
		if u.Want != nil {
			t.Errorf("Want should be nil for status=own, got %v", u.Want)
		}
	})

	t.Run("remove clears all flags", func(t *testing.T) {
		u, err := statusToCollectionUpdate("remove", true) // value is ignored for remove
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		flags := []*bool{u.Own, u.PrevOwned, u.ForTrade, u.Want, u.WantToPlay, u.WantToBuy, u.Wishlist, u.Preordered}
		for i, f := range flags {
			if f == nil || *f {
				t.Errorf("flag %d not cleared to false: %+v", i, u)
			}
		}
	})

	t.Run("unknown status errors", func(t *testing.T) {
		if _, err := statusToCollectionUpdate("bogus", true); err == nil {
			t.Error("expected error for unknown status, got nil")
		}
	})
}
