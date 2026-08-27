package persistence

import (
	"fmt"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

type recoveryState struct {
	cases      map[string]domain.CaseRecord
	idempotent map[string]domain.CaseRecord
	sequence   int64
	digest     string
}

func replay(events []Event) (recoveryState, error) {
	state := recoveryState{cases: map[string]domain.CaseRecord{}, idempotent: map[string]domain.CaseRecord{}}
	for index, event := range events {
		expectedSequence := int64(index + 1)
		if event.Sequence != expectedSequence {
			return state, fmt.Errorf("事件序号乱序：期待 %d，得到 %d", expectedSequence, event.Sequence)
		}
		if event.PreviousDigest != state.digest {
			return state, fmt.Errorf("事件 %d 的前序摘要不匹配", event.Sequence)
		}
		computed, err := eventDigest(event)
		if err != nil {
			return state, err
		}
		if computed != event.Digest {
			return state, fmt.Errorf("事件 %d 摘要损坏", event.Sequence)
		}
		current, exists := state.cases[event.CaseID]
		expectedVersion := int64(1)
		if exists {
			expectedVersion = current.Case.Version + 1
		}
		if event.CaseVersion != expectedVersion || event.Record.Case.Version != event.CaseVersion {
			return state, fmt.Errorf("档案 %s 版本不连续", event.CaseID)
		}
		state.cases[event.CaseID] = event.Record
		state.idempotent[idempotencyIndex(event.CaseID, event.IdempotencyKey)] = event.Record
		state.sequence, state.digest = event.Sequence, event.Digest
	}
	return state, nil
}
