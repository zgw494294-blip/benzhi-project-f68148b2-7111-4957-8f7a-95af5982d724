package publicdefaultsigningkey_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/analysis"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/application"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/persistence"
)

func TestSourceKnownDefaultKeyMustNotAuthenticateForgery(t *testing.T) {
	repo, err := persistence.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Unix(100, 0).UTC()
	stored := domain.DatingCredential{
		CredentialID: "DENDRO-existing", CaseID: "case-existing", ManifestDigest: "manifest-digest",
		CalendarConclusion: "1900-1940", ConfidenceStatement: "原始结论", IssuedBy: "签发员",
		IssuedAt: issuedAt, Status: "valid", PayloadDigest: "stored-payload", SignatureDigest: "stored-signature",
	}
	record := domain.CaseRecord{
		Case:     domain.DatingCase{CaseID: "case-existing", Status: domain.StatusPublished},
		Manifest: &domain.EvidenceManifest{ManifestDigest: "manifest-digest"}, Credential: &stored,
	}
	if _, _, err := repo.Save(record, 0, "seed", "CredentialPublished", "签发员", "发布", issuedAt); err != nil {
		t.Fatal(err)
	}

	forged := stored
	forged.CalendarConclusion = "9999-9999"
	forged.ConfidenceStatement = "伪造结论"
	forged.PayloadDigest = credentialPayloadDigest(forged)
	mac := hmac.New(sha256.New, []byte("local-dendrochronology-evidence-key-v1"))
	_, _ = mac.Write([]byte(forged.PayloadDigest))
	forged.SignatureDigest = hex.EncodeToString(mac.Sum(nil))

	service := application.NewService(repo, analysis.NewEngine(), "")
	result, err := service.Verify(application.VerifyInput{Payload: &forged})
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid {
		t.Fatalf("TestSourceKnownDefaultKeyMustNotAuthenticateForgery: 使用源码固定默认密钥的篡改载荷被标记为 valid，category=%s", result.Category)
	}
}

func credentialPayloadDigest(c domain.DatingCredential) string {
	return domain.MustDigest(struct {
		CredentialID        string `json:"credentialID"`
		CaseID              string `json:"caseID"`
		ManifestDigest      string `json:"manifestDigest"`
		CalendarConclusion  string `json:"calendarConclusion"`
		ConfidenceStatement string `json:"confidenceStatement"`
		IssuedBy            string `json:"issuedBy"`
		IssuedAt            any    `json:"issuedAt"`
		Status              string `json:"status"`
	}{c.CredentialID, c.CaseID, c.ManifestDigest, c.CalendarConclusion, c.ConfidenceStatement, c.IssuedBy, c.IssuedAt, c.Status})
}
