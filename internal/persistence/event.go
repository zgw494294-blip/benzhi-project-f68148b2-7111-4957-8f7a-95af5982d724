package persistence

import (
	"encoding/json"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

type Event struct {
	Sequence       int64             `json:"sequence"`
	CaseID         string            `json:"caseID"`
	CaseVersion    int64             `json:"caseVersion"`
	Kind           string            `json:"kind"`
	Actor          string            `json:"actor"`
	Summary        string            `json:"summary"`
	IdempotencyKey string            `json:"idempotencyKey"`
	OccurredAt     time.Time         `json:"occurredAt"`
	PreviousDigest string            `json:"previousDigest"`
	Record         domain.CaseRecord `json:"record"`
	Digest         string            `json:"digest"`
}

func eventDigest(event Event) (string, error) {
	payload := struct {
		Sequence       int64           `json:"sequence"`
		CaseID         string          `json:"caseID"`
		CaseVersion    int64           `json:"caseVersion"`
		Kind           string          `json:"kind"`
		Actor          string          `json:"actor"`
		Summary        string          `json:"summary"`
		IdempotencyKey string          `json:"idempotencyKey"`
		OccurredAt     time.Time       `json:"occurredAt"`
		PreviousDigest string          `json:"previousDigest"`
		Record         json.RawMessage `json:"record"`
	}{event.Sequence, event.CaseID, event.CaseVersion, event.Kind, event.Actor, event.Summary, event.IdempotencyKey, event.OccurredAt, event.PreviousDigest, nil}
	b, err := json.Marshal(event.Record)
	if err != nil {
		return "", err
	}
	payload.Record = b
	return domain.Digest(payload)
}
