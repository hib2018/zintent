package ui

type Dashboard struct {
	IntentID, Lifecycle, RevisionID string
	BlockerCount                    int
	SnapshotID                      string
}

func (d Dashboard) Routes() []Screen {
	routes := []Screen{ScreenReview, ScreenComments, ScreenCompletion, ScreenHistory, ScreenValidation}
	if d.Lifecycle == "review_complete" {
		routes = append(routes, ScreenApproval)
	}
	if d.SnapshotID != "" {
		routes = append(routes, ScreenSnapshot)
	}
	return routes
}
