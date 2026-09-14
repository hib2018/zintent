package ui

type CommentRecord struct{ ID, TargetItemID, Body, Status string }
type CommentsScreen struct {
	Records                  []CommentRecord
	SelectedID, ReturnItemID string
	Offset                   int
}

func (s CommentsScreen) Reload(records []CommentRecord) CommentsScreen {
	selected := s.SelectedID
	s.Records = append(s.Records[:0], records...)
	s.SelectedID = ""
	for _, record := range s.Records {
		if record.ID == selected {
			s.SelectedID = selected
			return s
		}
	}
	if len(s.Records) > 0 {
		s.SelectedID = s.Records[0].ID
	}
	return s
}
func (s CommentsScreen) Selected() *CommentRecord {
	for i := range s.Records {
		if s.Records[i].ID == s.SelectedID {
			return &s.Records[i]
		}
	}
	return nil
}
func ValidateClosureReason(reason string) bool { return len([]rune(reason)) > 0 }
func (s CommentsScreen) Move(delta int) CommentsScreen {
	if len(s.Records) == 0 {
		return s
	}
	index := 0
	for i, r := range s.Records {
		if r.ID == s.SelectedID {
			index = i
			break
		}
	}
	index = max(0, min(index+delta, len(s.Records)-1))
	s.SelectedID = s.Records[index].ID
	return s
}
