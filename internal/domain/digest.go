package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func Digest(value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func MustDigest(value any) string {
	d, err := Digest(value)
	if err != nil {
		panic(err)
	}
	return d
}

func CloneRecord(record CaseRecord) (CaseRecord, error) {
	b, err := json.Marshal(record)
	if err != nil {
		return CaseRecord{}, err
	}
	var clone CaseRecord
	err = json.Unmarshal(b, &clone)
	return clone, err
}
