package application

import (
	"strings"
	"time"

	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/analysis"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"
	"benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/persistence"
)

type Service struct {
	repo       *persistence.Repository
	engine     *analysis.Engine
	now        func() time.Time
	signingKey string
}

func NewService(repo *persistence.Repository, engine *analysis.Engine, signingKey string) *Service {
	if signingKey == "" {
		signingKey = "local-dendrochronology-evidence-key-v1"
	}
	return &Service{repo: repo, engine: engine, now: time.Now, signingKey: signingKey}
}

func (s *Service) CreateCase(input CreateCaseInput) (domain.CaseRecord, error) {
	if strings.TrimSpace(input.CaseID) == "" {
		input.CaseID = domain.NewID("case")
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return domain.CaseRecord{}, domain.Invalid("idempotencyKey", "不能为空")
	}
	if input.CreatedBy == "" {
		return domain.CaseRecord{}, domain.Invalid("createdBy", "不能为空")
	}
	now := s.now().UTC()
	c := domain.DatingCase{CaseID: input.CaseID, SiteName: input.SiteName, StructureRef: input.StructureRef, SamplingPosition: input.SamplingPosition, SpeciesAssessment: input.SpeciesAssessment, SampledAt: input.SampledAt, MinimumOverlap: input.MinimumOverlap, Status: domain.StatusDraft, CreatedBy: input.CreatedBy, CreatedAt: now, UpdatedAt: now}
	if err := domain.ValidateNewCase(c); err != nil {
		return domain.CaseRecord{}, err
	}
	record := domain.CaseRecord{Case: c, Timeline: []domain.TimelineEntry{}}
	result, _, err := s.repo.Save(record, 0, input.IdempotencyKey, "CaseCreated", input.CreatedBy, "建立木芯鉴定档案", now)
	return result, err
}

func (s *Service) GetCase(caseID string) (domain.CaseRecord, error) { return s.repo.Get(caseID) }
func (s *Service) ListCases() ([]domain.CaseRecord, error)          { return s.repo.List() }
func (s *Service) Health() persistence.Health                       { return s.repo.Health() }

func validateMeta(meta CommandMeta) error {
	if meta.ExpectedVersion < 1 {
		return domain.Invalid("expectedVersion", "必须为正整数")
	}
	if strings.TrimSpace(meta.IdempotencyKey) == "" {
		return domain.Invalid("idempotencyKey", "不能为空")
	}
	if strings.TrimSpace(meta.Actor) == "" {
		return domain.Invalid("actor", "不能为空")
	}
	return nil
}
