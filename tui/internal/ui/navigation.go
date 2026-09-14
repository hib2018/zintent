package ui

type Screen uint8

const (
	ScreenIntentList Screen = iota
	ScreenDashboard
	ScreenReview
	ScreenComments
	ScreenCompletion
	ScreenApproval
	ScreenHistory
	ScreenDiff
	ScreenValidation
	ScreenSnapshot
	ScreenRecovery
)

func (s Screen) String() string {
	return [...]string{"intents", "dashboard", "review", "comments", "completion", "approval", "history", "diff", "validation", "snapshot", "recovery"}[s]
}

type navigation struct{ stack []Screen }

func (n *navigation) current() Screen {
	if len(n.stack) == 0 {
		return ScreenIntentList
	}
	return n.stack[len(n.stack)-1]
}
func (n *navigation) push(screen Screen) {
	if n.current() != screen {
		n.stack = append(n.stack, screen)
	}
}
func (n *navigation) back() {
	if len(n.stack) > 1 {
		n.stack = n.stack[:len(n.stack)-1]
	}
}
