package store

import (
	"testing"
)

func TestRecordFeedbackEventAppendsRow(t *testing.T) {
	st := openTestStore(t)

	if err := st.RecordFeedbackEvent(FeedbackActionOpen, "item-1", ""); err != nil {
		t.Fatalf("RecordFeedbackEvent: %v", err)
	}

	events, err := st.ListFeedbackEvents(0)
	if err != nil {
		t.Fatalf("ListFeedbackEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Action != FeedbackActionOpen {
		t.Errorf("action = %q, want %q", ev.Action, FeedbackActionOpen)
	}
	if ev.ItemID != "item-1" {
		t.Errorf("item id = %q, want item-1", ev.ItemID)
	}
	if ev.Target != "" {
		t.Errorf("target = %q, want empty", ev.Target)
	}
	if ev.CreatedAt.IsZero() {
		t.Error("created_at should be set")
	}
}

func TestListFeedbackEventsOrdersMostRecentFirst(t *testing.T) {
	st := openTestStore(t)

	events := []struct {
		action, item string
	}{
		{FeedbackActionOpen, "item-1"},
		{FeedbackActionSave, "item-2"},
		{FeedbackActionRead, "item-3"},
	}
	for _, e := range events {
		if err := st.RecordFeedbackEvent(e.action, e.item, ""); err != nil {
			t.Fatalf("record %s: %v", e.action, err)
		}
	}

	got, err := st.ListFeedbackEvents(0)
	if err != nil {
		t.Fatalf("ListFeedbackEvents: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d events, want 3", len(got))
	}
	if got[0].Action != FeedbackActionRead || got[2].Action != FeedbackActionOpen {
		t.Errorf("expected DESC by id; got %q then %q", got[0].Action, got[2].Action)
	}
}

func TestListFeedbackEventsRespectsLimit(t *testing.T) {
	st := openTestStore(t)

	for i := 0; i < 5; i++ {
		if err := st.RecordFeedbackEvent(FeedbackActionOpen, "item", ""); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	got, err := st.ListFeedbackEvents(2)
	if err != nil {
		t.Fatalf("ListFeedbackEvents: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("limit 2 returned %d rows", len(got))
	}
}

func TestListFeedbackEventsByActionFiltersByTag(t *testing.T) {
	st := openTestStore(t)

	seed := []struct{ action, item, target string }{
		{FeedbackActionOpen, "a", ""},
		{FeedbackActionSave, "a", ""},
		{FeedbackActionMuteDomain, "", "example.com"},
		{FeedbackActionSave, "b", ""},
	}
	for _, e := range seed {
		if err := st.RecordFeedbackEvent(e.action, e.item, e.target); err != nil {
			t.Fatalf("record %s: %v", e.action, err)
		}
	}

	saves, err := st.ListFeedbackEventsByAction(FeedbackActionSave, 0)
	if err != nil {
		t.Fatalf("ListFeedbackEventsByAction: %v", err)
	}
	if len(saves) != 2 {
		t.Fatalf("saves = %d, want 2", len(saves))
	}
	for _, ev := range saves {
		if ev.Action != FeedbackActionSave {
			t.Errorf("unexpected action %q", ev.Action)
		}
	}

	mutes, err := st.ListFeedbackEventsByAction(FeedbackActionMuteDomain, 0)
	if err != nil {
		t.Fatalf("ListFeedbackEventsByAction: %v", err)
	}
	if len(mutes) != 1 || mutes[0].Target != "example.com" {
		t.Fatalf("mute events = %+v", mutes)
	}
}
