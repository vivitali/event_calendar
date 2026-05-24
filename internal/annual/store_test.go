package annual

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeSSM struct {
	getValue string
	getErr   error
	putValue string
	putErr   error
}

func (f *fakeSSM) GetParameter(ctx context.Context, name string) (string, error) {
	return f.getValue, f.getErr
}

func (f *fakeSSM) PutParameter(ctx context.Context, name, value string) error {
	f.putValue = value
	return f.putErr
}

func TestSSMStore_Get_ReturnsParsedEvents(t *testing.T) {
	want := []AnnualEvent{
		{Name: "X", Date: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC), URL: "https://x"},
	}
	raw, _ := json.Marshal(want)
	store := &SSMStore{api: &fakeSSM{getValue: string(raw)}, name: "/p"}
	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 1 || got[0].Name != "X" {
		t.Errorf("unexpected events: %+v", got)
	}
}

func TestSSMStore_Get_ReturnsEmptyOnNotFound(t *testing.T) {
	store := &SSMStore{api: &fakeSSM{getErr: ErrParameterNotFound}, name: "/p"}
	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %+v", got)
	}
}

func TestSSMStore_Get_PropagatesOtherErrors(t *testing.T) {
	want := errors.New("boom")
	store := &SSMStore{api: &fakeSSM{getErr: want}, name: "/p"}
	_, err := store.Get(context.Background())
	if !errors.Is(err, want) {
		t.Errorf("expected wrapped %v, got %v", want, err)
	}
}

func TestSSMStore_Put_MarshalsJSON(t *testing.T) {
	api := &fakeSSM{}
	store := &SSMStore{api: api, name: "/p"}
	events := []AnnualEvent{{Name: "Y", Date: time.Date(2026, 6, 11, 19, 0, 0, 0, time.UTC), URL: "https://y"}}
	if err := store.Put(context.Background(), events); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	var round []AnnualEvent
	if err := json.Unmarshal([]byte(api.putValue), &round); err != nil {
		t.Fatalf("put value is not JSON: %v (%q)", err, api.putValue)
	}
	if len(round) != 1 || round[0].Name != "Y" {
		t.Errorf("unexpected round-trip: %+v", round)
	}
}
