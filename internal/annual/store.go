package annual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrParameterNotFound is returned by ssmAPI.GetParameter when the parameter
// does not exist. SSMStore.Get translates this to an empty slice.
var ErrParameterNotFound = errors.New("parameter not found")

// Store persists the curated annual-events list between Lambda invocations.
type Store interface {
	Get(ctx context.Context) ([]AnnualEvent, error)
	Put(ctx context.Context, events []AnnualEvent) error
}

// ssmAPI is the slice of the AWS SSM client we need. Lets tests substitute
// a fake without pulling in the SDK.
type ssmAPI interface {
	GetParameter(ctx context.Context, name string) (string, error)
	PutParameter(ctx context.Context, name, value string) error
}

// SSMStore is a Store backed by AWS SSM Parameter Store.
type SSMStore struct {
	api  ssmAPI
	name string
}

// NewSSMStore returns a store that reads/writes the named parameter.
func NewSSMStore(api ssmAPI, name string) *SSMStore {
	return &SSMStore{api: api, name: name}
}

// Get returns the parsed list. Missing parameter → empty slice, nil error.
func (s *SSMStore) Get(ctx context.Context) ([]AnnualEvent, error) {
	raw, err := s.api.GetParameter(ctx, s.name)
	if err != nil {
		if errors.Is(err, ErrParameterNotFound) {
			return []AnnualEvent{}, nil
		}
		return nil, fmt.Errorf("ssm get %s: %w", s.name, err)
	}
	if raw == "" {
		return []AnnualEvent{}, nil
	}
	var out []AnnualEvent
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("ssm parse %s: %w", s.name, err)
	}
	return out, nil
}

// Put serialises events to JSON and writes the parameter (overwriting).
func (s *SSMStore) Put(ctx context.Context, events []AnnualEvent) error {
	if events == nil {
		events = []AnnualEvent{}
	}
	raw, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("ssm marshal: %w", err)
	}
	if err := s.api.PutParameter(ctx, s.name, string(raw)); err != nil {
		return fmt.Errorf("ssm put %s: %w", s.name, err)
	}
	return nil
}
