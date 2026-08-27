package domain

import "time"

type Status string

const (
	StatusDraft           Status = "Draft"
	StatusSeriesSubmitted Status = "SeriesSubmitted"
	StatusAnalyzed        Status = "Analyzed"
	StatusNeedsCorrection Status = "NeedsCorrection"
	StatusUnderReview     Status = "UnderReview"
	StatusApproved        Status = "Approved"
	StatusFrozen          Status = "Frozen"
	StatusPublished       Status = "Published"
)

type DatingCase struct {
	CaseID            string    `json:"caseID"`
	SiteName          string    `json:"siteName"`
	StructureRef      string    `json:"structureRef"`
	SamplingPosition  string    `json:"samplingPosition"`
	SpeciesAssessment string    `json:"speciesAssessment"`
	SampledAt         string    `json:"sampledAt"`
	MinimumOverlap    int       `json:"minimumOverlap"`
	Status            Status    `json:"status"`
	Version           int64     `json:"version"`
	CreatedBy         string    `json:"createdBy"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type MetadataRevision struct {
	RevisionID string     `json:"revisionID"`
	Before     DatingCase `json:"before"`
	After      DatingCase `json:"after"`
	Reason     string     `json:"reason"`
	RevisedBy  string     `json:"revisedBy"`
	RevisedAt  time.Time  `json:"revisedAt"`
}

type RingSeriesRevision struct {
	RevisionID           string    `json:"revisionID"`
	CaseID               string    `json:"caseID"`
	ParentRevisionID     string    `json:"parentRevisionID,omitempty"`
	Unit                 string    `json:"unit"`
	MeasurementDirection string    `json:"measurementDirection"`
	Widths               []float64 `json:"widths"`
	ContentDigest        string    `json:"contentDigest"`
	ChangeReason         string    `json:"changeReason"`
	SubmittedBy          string    `json:"submittedBy"`
	SubmittedAt          time.Time `json:"submittedAt"`
}

type AnalysisParameters struct {
	MinimumOverlap int     `json:"minimumOverlap"`
	MaxCandidates  int     `json:"maxCandidates"`
	LowCorrelation float64 `json:"lowCorrelation"`
	OutlierZScore  float64 `json:"outlierZScore"`
}

type Candidate struct {
	Rank          int     `json:"rank"`
	StartYear     int     `json:"startYear"`
	EndYear       int     `json:"endYear"`
	Overlap       int     `json:"overlap"`
	Correlation   float64 `json:"correlation"`
	ReferenceFrom int     `json:"referenceFrom"`
}

type QualityFlag struct {
	Code          string `json:"code"`
	AffectedRange string `json:"affectedRange"`
	Severity      string `json:"severity"`
	Message       string `json:"message"`
	Blocking      bool   `json:"blocking"`
}

type CrossdatingRun struct {
	RunID                 string             `json:"runID"`
	CaseID                string             `json:"caseID"`
	RevisionID            string             `json:"revisionID"`
	ReferenceChronologyID string             `json:"referenceChronologyID"`
	ParameterSet          AnalysisParameters `json:"parameterSet"`
	RankedCandidates      []Candidate        `json:"rankedCandidates"`
	BestCalendarRange     string             `json:"bestCalendarRange"`
	CorrelationScore      float64            `json:"correlationScore"`
	QualityFlags          []QualityFlag      `json:"qualityFlags"`
	ResultDigest          string             `json:"resultDigest"`
	ExecutedAt            time.Time          `json:"executedAt"`
}

type DisputeResolution struct {
	ResolutionID          string    `json:"resolutionID"`
	CaseID                string    `json:"caseID"`
	RunID                 string    `json:"runID"`
	FlagCode              string    `json:"flagCode"`
	AffectedRange         string    `json:"affectedRange"`
	Disposition           string    `json:"disposition"`
	Rationale             string    `json:"rationale"`
	EvidenceRefs          []string  `json:"evidenceRefs"`
	ReplacementRevisionID string    `json:"replacementRevisionID,omitempty"`
	ResolvedBy            string    `json:"resolvedBy"`
	ResolvedAt            time.Time `json:"resolvedAt"`
}

type ReviewDecision struct {
	DecisionID  string    `json:"decisionID"`
	CaseID      string    `json:"caseID"`
	CaseVersion int64     `json:"caseVersion"`
	RevisionID  string    `json:"revisionID"`
	RunID       string    `json:"runID"`
	Outcome     string    `json:"outcome"`
	Findings    string    `json:"findings"`
	ReviewerID  string    `json:"reviewerID"`
	DecidedAt   time.Time `json:"decidedAt"`
}

type ReviewChecklistItem struct {
	ItemCode          string `json:"itemCode"`
	Label             string `json:"label"`
	Outcome           string `json:"outcome,omitempty"`
	Finding           string `json:"finding,omitempty"`
	RemediationTarget string `json:"remediationTarget,omitempty"`
}

type ReviewChecklist struct {
	ChecklistID string                `json:"checklistID"`
	CaseID      string                `json:"caseID"`
	CaseVersion int64                 `json:"caseVersion"`
	RevisionID  string                `json:"revisionID"`
	RunID       string                `json:"runID"`
	Items       []ReviewChecklistItem `json:"items"`
	Status      string                `json:"status"`
	ReviewerID  string                `json:"reviewerID,omitempty"`
	CreatedAt   time.Time             `json:"createdAt"`
	DecidedAt   time.Time             `json:"decidedAt,omitempty"`
}

type RemediationResponse struct {
	ChecklistID    string    `json:"checklistID"`
	ItemCode       string    `json:"itemCode"`
	Response       string    `json:"response"`
	EvidenceEvents []int64   `json:"evidenceEvents"`
	RespondedBy    string    `json:"respondedBy"`
	RespondedAt    time.Time `json:"respondedAt"`
}

type ManifestSection struct {
	Name    string `json:"name"`
	Count   int    `json:"count"`
	Digest  string `json:"digest"`
	Ready   bool   `json:"ready"`
	Message string `json:"message"`
}

type ManifestConfirmation struct {
	CaseVersion        int64  `json:"caseVersion"`
	ActiveRevisionID   string `json:"activeRevisionID"`
	ActiveRunID        string `json:"activeRunID"`
	ReviewDecisionID   string `json:"reviewDecisionID"`
	ManifestDigest     string `json:"manifestDigest"`
	ConfirmationDigest string `json:"confirmationDigest"`
}

type ManifestReadiness struct {
	Ready        bool                 `json:"ready"`
	Frozen       bool                 `json:"frozen"`
	Blockers     []string             `json:"blockers"`
	Sections     []ManifestSection    `json:"sections"`
	Confirmation ManifestConfirmation `json:"confirmation"`
	Manifest     *EvidenceManifest    `json:"manifest,omitempty"`
}

type EvidenceManifest struct {
	CaseSnapshot    DatingCase          `json:"caseSnapshot"`
	ActiveRevision  RingSeriesRevision  `json:"activeRevision"`
	ActiveRun       CrossdatingRun      `json:"activeRun"`
	Resolutions     []DisputeResolution `json:"resolutions"`
	Review          ReviewDecision      `json:"review"`
	ReviewChecklist ReviewChecklist     `json:"reviewChecklist"`
	FrozenBy        string              `json:"frozenBy"`
	FrozenAt        time.Time           `json:"frozenAt"`
	ManifestDigest  string              `json:"manifestDigest"`
}

type DatingCredential struct {
	CredentialID        string    `json:"credentialID"`
	CaseID              string    `json:"caseID"`
	ManifestDigest      string    `json:"manifestDigest"`
	CalendarConclusion  string    `json:"calendarConclusion"`
	ConfidenceStatement string    `json:"confidenceStatement"`
	IssuedBy            string    `json:"issuedBy"`
	IssuedAt            time.Time `json:"issuedAt"`
	PayloadDigest       string    `json:"payloadDigest"`
	SignatureDigest     string    `json:"signatureDigest"`
	Status              string    `json:"status"`
}

type TimelineEntry struct {
	Sequence int64     `json:"sequence"`
	Kind     string    `json:"kind"`
	Actor    string    `json:"actor"`
	At       time.Time `json:"at"`
	Summary  string    `json:"summary"`
}

type CaseRecord struct {
	Case                 DatingCase            `json:"case"`
	MetadataRevisions    []MetadataRevision    `json:"metadataRevisions,omitempty"`
	Revisions            []RingSeriesRevision  `json:"revisions"`
	Runs                 []CrossdatingRun      `json:"runs"`
	Resolutions          []DisputeResolution   `json:"resolutions"`
	Reviews              []ReviewDecision      `json:"reviews"`
	ReviewChecklists     []ReviewChecklist     `json:"reviewChecklists,omitempty"`
	RemediationResponses []RemediationResponse `json:"remediationResponses,omitempty"`
	Manifest             *EvidenceManifest     `json:"manifest,omitempty"`
	Credential           *DatingCredential     `json:"credential,omitempty"`
	ActiveRevisionID     string                `json:"activeRevisionID,omitempty"`
	ActiveRunID          string                `json:"activeRunID,omitempty"`
	LastSubmitter        string                `json:"lastSubmitter,omitempty"`
	Timeline             []TimelineEntry       `json:"timeline"`
}
