package ui

import "testing"

func TestReviewWorkspaceStableNavigationAndScrolling(t *testing.T) {
	s := ReviewScreen{Height: 1}.Reload([]Item{{ID: "i1", Status: "accepted"}, {ID: "i2", Status: "unreviewed", Statement: "full statement", Provenance: "source"}})
	if s.SelectedID != "i2" {
		t.Fatal(s.SelectedID)
	}
	s.Offset = 1
	if got := s.Visible(); len(got) != 1 || got[0].ID != "i2" {
		t.Fatal(got)
	}
	s = s.Reload([]Item{{ID: "i2"}, {ID: "i3"}})
	if s.SelectedID != "i2" {
		t.Fatal("stable selection lost")
	}
}
func TestCommentReturnAndReasonValidation(t *testing.T) {
	s := CommentsScreen{ReturnItemID: "i1"}.Reload([]CommentRecord{{ID: "c1", TargetItemID: "i1", Body: "full", Status: "open"}})
	if s.Selected() == nil || !ValidateClosureReason("resolved by edit") {
		t.Fatal()
	}
	if ValidateClosureReason("") {
		t.Fatal("empty reason accepted")
	}
}
