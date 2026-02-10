package a800mon

import "context"

type ShortcutInput struct {
	shortcuts  *ShortcutManager
	dispatcher *ActionDispatcher
}

func NewShortcutInput(shortcuts *ShortcutManager, dispatcher *ActionDispatcher) *ShortcutInput {
	return &ShortcutInput{shortcuts: shortcuts, dispatcher: dispatcher}
}

func (s *ShortcutInput) Update(ctx context.Context) (bool, error) { return false, nil }
func (s *ShortcutInput) PostRender(ctx context.Context) error     { return nil }

func (s *ShortcutInput) HandleInput(ch int) bool {
	st := State()
	if st.InputFocus {
		return false
	}
	if shortcut, ok := s.shortcuts.Global(ch); ok {
		if shortcut.Callback != nil {
			shortcut.Callback()
		}
		return true
	}
	layer := s.shortcuts.Get(st.ActiveMode)
	if layer != nil {
		if shortcut, ok := layer.Get(ch); ok {
			if shortcut.Callback != nil {
				shortcut.Callback()
			}
			return true
		}
	}
	lower := ch
	if ch >= int('A') && ch <= int('Z') {
		lower = ch + 32
	}
	if st.DisplayListInspect && (lower == int('j') || lower == int('k')) {
		if lower == int('j') {
			_ = s.dispatcher.Dispatch(ActionDListNext, nil)
		} else {
			_ = s.dispatcher.Dispatch(ActionDListPrev, nil)
		}
		return true
	}
	if st.DisplayListInspect && (ch == KeyDown() || ch == KeyUp()) {
		if ch == KeyDown() {
			_ = s.dispatcher.Dispatch(ActionDListNext, nil)
		} else {
			_ = s.dispatcher.Dispatch(ActionDListPrev, nil)
		}
		return true
	}
	return false
}
