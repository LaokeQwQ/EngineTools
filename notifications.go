package main

import wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

// Confirmation requests stay inside the application instead of opening Windows dialogs.
func (a *App) confirm(title, message string) bool {
	if a.ctx == nil {
		return false
	}
	a.confirmationMu.Lock()
	if a.closing {
		a.confirmationMu.Unlock()
		return false
	}
	if a.confirmations == nil {
		a.confirmations = make(map[uint32]chan bool)
	}
	a.confirmationNext++
	id := a.confirmationNext
	answer := make(chan bool, 1)
	a.confirmations[id] = answer
	a.confirmationMu.Unlock()
	defer func() {
		a.confirmationMu.Lock()
		delete(a.confirmations, id)
		a.confirmationMu.Unlock()
	}()
	wailsRuntime.EventsEmit(a.ctx, "confirmation", map[string]interface{}{
		"id": id, "title": title, "message": message,
	})
	select {
	case accepted := <-answer:
		return accepted
	case <-a.ctx.Done():
		return false
	}
}

func (a *App) RespondToConfirmation(id uint32, accepted bool) {
	a.confirmationMu.Lock()
	answer := a.confirmations[id]
	delete(a.confirmations, id)
	a.confirmationMu.Unlock()
	if answer != nil {
		answer <- accepted
	}
}

func (a *App) cancelConfirmations() {
	a.confirmationMu.Lock()
	a.closing = true
	pending := a.confirmations
	a.confirmations = nil
	a.confirmationMu.Unlock()
	for _, answer := range pending {
		answer <- false
	}
}
