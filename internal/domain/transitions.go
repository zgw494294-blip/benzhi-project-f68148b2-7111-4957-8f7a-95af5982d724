package domain

func TransitionAllowed(from, to Status) bool {
	allowed := map[Status]map[Status]bool{
		StatusDraft:           {StatusSeriesSubmitted: true},
		StatusSeriesSubmitted: {StatusAnalyzed: true, StatusNeedsCorrection: true},
		StatusAnalyzed:        {StatusNeedsCorrection: true, StatusUnderReview: true, StatusSeriesSubmitted: true},
		StatusNeedsCorrection: {StatusAnalyzed: true, StatusUnderReview: true, StatusSeriesSubmitted: true},
		StatusUnderReview:     {StatusApproved: true, StatusNeedsCorrection: true},
		StatusApproved:        {StatusFrozen: true},
		StatusFrozen:          {StatusPublished: true},
	}
	return allowed[from][to]
}

func SetStatus(record *CaseRecord, to Status) error {
	if record.Case.Status == to {
		return nil
	}
	if !TransitionAllowed(record.Case.Status, to) {
		return Conflict("非法状态转换：" + string(record.Case.Status) + " -> " + string(to))
	}
	record.Case.Status = to
	return nil
}
