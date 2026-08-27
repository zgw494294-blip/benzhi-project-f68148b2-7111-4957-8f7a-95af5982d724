package application

import "benzhi-project-f68148b2-7111-4957-8f7a-95af5982d724/internal/domain"

type CommandMeta struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
	Actor           string `json:"actor"`
}

type CreateCaseInput struct {
	CaseID            string `json:"caseID"`
	SiteName          string `json:"siteName"`
	StructureRef      string `json:"structureRef"`
	SamplingPosition  string `json:"samplingPosition"`
	SpeciesAssessment string `json:"speciesAssessment"`
	SampledAt         string `json:"sampledAt"`
	MinimumOverlap    int    `json:"minimumOverlap"`
	CreatedBy         string `json:"createdBy"`
	IdempotencyKey    string `json:"idempotencyKey"`
}

type ReviseMetadataInput struct {
	CommandMeta
	SiteName                    string `json:"siteName"`
	StructureRef                string `json:"structureRef"`
	SamplingPosition            string `json:"samplingPosition"`
	SpeciesAssessment           string `json:"speciesAssessment"`
	SampledAt                   string `json:"sampledAt"`
	MinimumOverlap              int    `json:"minimumOverlap"`
	Reason                      string `json:"reason"`
	DuplicateConfirmationReason string `json:"duplicateConfirmationReason,omitempty"`
}

type DuplicateCandidate struct {
	CaseID         string   `json:"caseID"`
	MatchingFields []string `json:"matchingFields"`
}

type SubmitSeriesInput struct {
	CommandMeta
	Unit                 string    `json:"unit"`
	MeasurementDirection string    `json:"measurementDirection"`
	Widths               []float64 `json:"widths"`
	ChangeReason         string    `json:"changeReason"`
	PreflightDigest      string    `json:"preflightDigest,omitempty"`
}

type SeriesPreflightInput struct {
	CaseVersion          int64  `json:"caseVersion"`
	RawText              string `json:"rawText"`
	Unit                 string `json:"unit"`
	MeasurementDirection string `json:"measurementDirection"`
}

type SeriesFieldError struct {
	Position int    `json:"position"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}
type SeriesDifference struct {
	Position int      `json:"position"`
	Kind     string   `json:"kind"`
	Before   *float64 `json:"before,omitempty"`
	After    *float64 `json:"after,omitempty"`
}
type SeriesStatistics struct {
	Count             int     `json:"count"`
	Minimum           float64 `json:"minimum"`
	Maximum           float64 `json:"maximum"`
	AbnormalPositions []int   `json:"abnormalPositions"`
}
type SeriesPreflightResult struct {
	CaseVersion      int64              `json:"caseVersion"`
	ParentRevisionID string             `json:"parentRevisionID,omitempty"`
	FirstImport      bool               `json:"firstImport"`
	Widths           []float64          `json:"widths"`
	Statistics       SeriesStatistics   `json:"statistics"`
	Errors           []SeriesFieldError `json:"errors"`
	Warnings         []string           `json:"warnings"`
	Differences      []SeriesDifference `json:"differences"`
	ChangeRatio      float64            `json:"changeRatio"`
	ContentDigest    string             `json:"contentDigest"`
	PreflightDigest  string             `json:"preflightDigest"`
}

type AnalyzeInput struct {
	CommandMeta
	Parameters *domain.AnalysisParameters `json:"parameters,omitempty"`
}

type ReplacementSeries struct {
	Unit                 string    `json:"unit"`
	MeasurementDirection string    `json:"measurementDirection"`
	Widths               []float64 `json:"widths"`
	ChangeReason         string    `json:"changeReason"`
}

type ResolveDisputeInput struct {
	CommandMeta
	FlagCode      string             `json:"flagCode"`
	AffectedRange string             `json:"affectedRange"`
	Disposition   string             `json:"disposition"`
	Rationale     string             `json:"rationale"`
	EvidenceRefs  []string           `json:"evidenceRefs"`
	Replacement   *ReplacementSeries `json:"replacement,omitempty"`
}

type BatchDisputeItem struct {
	FlagCode      string `json:"flagCode"`
	AffectedRange string `json:"affectedRange"`
	Rationale     string `json:"rationale"`
}
type BatchResolveDisputesInput struct {
	CommandMeta
	RunID        string             `json:"runID"`
	Items        []BatchDisputeItem `json:"items"`
	EvidenceRefs []string           `json:"evidenceRefs"`
}

type StartReviewInput struct{ CommandMeta }

type ReviewInput struct {
	CommandMeta
	Outcome  string `json:"outcome"`
	Findings string `json:"findings"`
}

type ReviewChecklistInput struct {
	CommandMeta
	ChecklistID string                       `json:"checklistID"`
	Outcome     string                       `json:"outcome"`
	Items       []domain.ReviewChecklistItem `json:"items"`
}
type RemediationResponseInput struct {
	CommandMeta
	ChecklistID    string  `json:"checklistID"`
	ItemCode       string  `json:"itemCode"`
	Response       string  `json:"response"`
	EvidenceEvents []int64 `json:"evidenceEvents"`
}

type FreezeInput struct{ CommandMeta }
type ConfirmedFreezeInput struct {
	CommandMeta
	Confirmation *domain.ManifestConfirmation `json:"confirmation"`
}

type PublishInput struct {
	CommandMeta
	ConfidenceStatement string `json:"confidenceStatement"`
}

type VerifyInput struct {
	CredentialID string                   `json:"credentialID,omitempty"`
	Payload      *domain.DatingCredential `json:"payload,omitempty"`
}

type VerificationResult struct {
	Valid        bool   `json:"valid"`
	Current      bool   `json:"current"`
	CredentialID string `json:"credentialID,omitempty"`
	CaseID       string `json:"caseID,omitempty"`
	Message      string `json:"message"`
	Category     string `json:"category,omitempty"`
}

type BatchVerifyInput struct {
	Entries []string `json:"entries"`
}
type BatchVerificationItem struct {
	Index int    `json:"index"`
	Input string `json:"input"`
	VerificationResult
}
type BatchVerificationResult struct {
	Items  []BatchVerificationItem `json:"items"`
	Counts map[string]int          `json:"counts"`
}

type RunHistoryItem struct {
	Run     domain.CrossdatingRun `json:"run"`
	Current bool                  `json:"current"`
	Stale   bool                  `json:"stale"`
}
type CandidateRankChange struct {
	StartYear  int `json:"startYear"`
	BeforeRank int `json:"beforeRank,omitempty"`
	AfterRank  int `json:"afterRank,omitempty"`
	Change     int `json:"change"`
}
type RunComparison struct {
	FirstRunID            string                `json:"firstRunID"`
	SecondRunID           string                `json:"secondRunID"`
	BestYearChange        string                `json:"bestYearChange"`
	CorrelationDifference float64               `json:"correlationDifference"`
	CandidateChanges      []CandidateRankChange `json:"candidateChanges"`
	AddedFlags            []domain.QualityFlag  `json:"addedFlags"`
	RemovedFlags          []domain.QualityFlag  `json:"removedFlags"`
	PersistentFlags       []domain.QualityFlag  `json:"persistentFlags"`
	FirstStale            bool                  `json:"firstStale"`
	SecondStale           bool                  `json:"secondStale"`
}

type QueueQuery struct {
	Statuses              []domain.Status
	Site                  string
	StructureRef          string
	UpdatedFrom           string
	UpdatedTo             string
	HasUnresolvedBlocking *bool
	Cursor                string
	PageSize              int
}
type QueueRow struct {
	Record             domain.CaseRecord `json:"record"`
	LastEventSummary   string            `json:"lastEventSummary"`
	NextAction         string            `json:"nextAction"`
	UnresolvedBlocking int               `json:"unresolvedBlocking"`
}
type QueueStatistics struct {
	StatusCounts     map[domain.Status]int `json:"statusCounts"`
	BlockingTotal    int                   `json:"blockingTotal"`
	StagnationGroups map[string]int        `json:"stagnationGroups"`
}
type QueueResult struct {
	Rows       []QueueRow      `json:"rows"`
	Statistics QueueStatistics `json:"statistics"`
	NextCursor string          `json:"nextCursor,omitempty"`
}
