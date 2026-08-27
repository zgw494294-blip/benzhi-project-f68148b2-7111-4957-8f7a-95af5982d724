package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
)

type snapshotPayload struct {
	Sequence   int64                        `json:"sequence"`
	LastDigest string                       `json:"lastDigest"`
	Cases      map[string]domain.CaseRecord `json:"cases"`
}

type snapshotEnvelope struct {
	Payload snapshotPayload `json:"payload"`
	Digest  string          `json:"digest"`
}

func writeSnapshot(path string, payload snapshotPayload) error {
	digest, err := domain.Digest(payload)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshotEnvelope{Payload: payload, Digest: digest}, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".projection-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		tmp.Close()
		if !ok {
			os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return err
	}
	ok = true
	return nil
}

func readSnapshot(path string) (snapshotPayload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return snapshotPayload{}, err
	}
	var envelope snapshotEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return snapshotPayload{}, fmt.Errorf("投影快照 JSON 损坏: %w", err)
	}
	digest, err := domain.Digest(envelope.Payload)
	if err != nil {
		return snapshotPayload{}, err
	}
	if digest != envelope.Digest {
		return snapshotPayload{}, fmt.Errorf("投影快照摘要不匹配")
	}
	return envelope.Payload, nil
}
