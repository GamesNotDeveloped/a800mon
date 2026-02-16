package monitor

import (
	"context"
	"time"
)

type UIErrorTimeout struct {
	BaseComponent
}

func NewUIErrorTimeout() *UIErrorTimeout {
	return &UIErrorTimeout{}
}

func (u *UIErrorTimeout) Update(_ctx context.Context) (bool, error) {
	st := State()
	if st.UIError == "" {
		return false, nil
	}
	if !st.UIErrorUntil.IsZero() && time.Now().After(st.UIErrorUntil) {
		u.App().DispatchAction(ActionClearUIError, nil)
		return true, nil
	}
	return false, nil
}
