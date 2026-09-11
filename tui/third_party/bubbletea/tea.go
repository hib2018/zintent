// Package tea provides the small reducer surface used by zintent.
// The module path matches Bubble Tea v2 so the application boundary can be
// developed offline; remove the go.mod replace when the pinned module is available.
package tea

type Msg any
type Cmd func() Msg

type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() View
}

type View struct{ Content string }

func NewView(content string) View { return View{Content: content} }

type KeyPressMsg struct{ Key string }

func (k KeyPressMsg) String() string { return k.Key }

type WindowSizeMsg struct{ Width, Height int }
type QuitMsg struct{}

func Quit() Msg { return QuitMsg{} }
