package ui

import (
	"fmt"
	"strings"
)

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
func (s CommentsScreen) View() string {
	var b strings.Builder
	b.WriteString("COMMENTS\n")
	if len(s.Records) == 0 {
		b.WriteString("  No comments.\n")
		return b.String()
	}
	for _, record := range s.Records {
		line := fmt.Sprintf("  [%s] %s", record.Status, shortRef(record.ID))
		if record.ID == s.SelectedID {
			line = highlightTopLine("→ " + strings.TrimPrefix(line, "  "))
		}
		fmt.Fprintf(&b, "%s\n  Item: %s\n  %s\n\n", line, shortRef(record.TargetItemID), record.Body)
	}
	b.WriteString("j/k select  r resolve  w withdraw  Esc back")
	return b.String()
}

func ValidateClosureReason(reason string) bool { return len([]rune(strings.TrimSpace(reason))) > 0 }
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
