package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"EngineTools/internal/apperror"
	"EngineTools/internal/privilege"
)

type record struct {
	action privilege.Action
	params any
}

type fakeClient struct {
	calls []record
	resp  privilege.Response
	err   error
}

func (f *fakeClient) Execute(_ context.Context, action privilege.Action, params any) (privilege.Response, error) {
	f.calls = append(f.calls, record{action: action, params: params})
	if f.err != nil {
		return privilege.Response{}, f.err
	}
	return f.resp, nil
}

func newTestApp(f *fakeClient) *App {
	a := &App{}
	a.privilege = f
	return a
}

func TestSystemFacade(t *testing.T) {
	t.Run("FixUnicodeIssues dispatches apply", func(t *testing.T) {
		f := &fakeClient{resp: privilege.Response{OK: true}}
		res := newTestApp(f).FixUnicodeIssues()
		if res.PublicError != nil || res.Changed != 1 {
			t.Fatalf("result: %+v", res)
		}
		if len(f.calls) != 1 || f.calls[0].action != privilege.ApplyManifest {
			t.Fatalf("calls: %+v", f.calls)
		}
		p, ok := f.calls[0].params.(privilege.InstallParams)
		if !ok || p.InstallID == "" {
			t.Fatalf("params: %#v", f.calls[0].params)
		}
	})

	t.Run("RestoreUnicodeFix dispatches remove", func(t *testing.T) {
		f := &fakeClient{resp: privilege.Response{OK: true}}
		res := newTestApp(f).RestoreUnicodeFix()
		if res.PublicError != nil {
			t.Fatalf("result: %+v", res)
		}
		if f.calls[0].action != privilege.RemoveManifest {
			t.Fatalf("action: %s", f.calls[0].action)
		}
	})

	t.Run("CleanMSIOrphans counts successes and keeps first error", func(t *testing.T) {
		f := &fakeClient{resp: privilege.Response{OK: true}}
		res := newTestApp(f).CleanMSIOrphans([]string{"{AAA}", "{BBB}"})
		if res.PublicError != nil || res.Changed != 2 {
			t.Fatalf("result: %+v", res)
		}
		if len(f.calls) != 2 {
			t.Fatalf("calls: %d", len(f.calls))
		}
	})

	t.Run("USBUnlockKill passes bare drive letter", func(t *testing.T) {
		f := &fakeClient{resp: privilege.Response{OK: true}}
		res := newTestApp(f).USBUnlockKill("d:")
		if res.PublicError != nil {
			t.Fatalf("result: %+v", res)
		}
		p := f.calls[0].params.(privilege.VolumeParams)
		if p.VolumeID != "D" {
			t.Fatalf("volumeId = %q", p.VolumeID)
		}
	})

	t.Run("client errors are sanitized", func(t *testing.T) {
		f := &fakeClient{err: errors.New(`open C:\Users\Alice\Secret\m.db: denied`)}
		res := newTestApp(f).FixUnicodeIssues()
		if res.PublicError == nil {
			t.Fatal("expected public error")
		}
		if strings.Contains(res.PublicError.MessageKey, `C:\`) ||
			strings.Contains(res.PublicError.Code, `C:\`) {
			t.Fatalf("leaked path: %+v", res.PublicError)
		}
		if res.Changed != 0 {
			t.Fatalf("changed on failure: %d", res.Changed)
		}
	})

	t.Run("helper rejection surfaces its code", func(t *testing.T) {
		f := &fakeClient{resp: privilege.Response{OK: false,
			Code: "privilege_action_denied", MessageKey: "errors.privilegeActionDenied"}}
		res := newTestApp(f).FixUnicodeIssues()
		if res.PublicError == nil || res.PublicError.Code != "privilege_action_denied" {
			t.Fatalf("result: %+v", res)
		}
	})

	t.Run("nil client fails closed with internal error", func(t *testing.T) {
		a := &App{}
		res := a.FixUnicodeIssues()
		if res.PublicError == nil || res.PublicError.Code != "internal" {
			t.Fatalf("result: %+v", res)
		}
	})

	t.Run("operation id is present", func(t *testing.T) {
		f := &fakeClient{resp: privilege.Response{OK: true}}
		res := newTestApp(f).FixUnicodeIssues()
		if len(res.OperationID) != 16 {
			t.Fatalf("operationId = %q", res.OperationID)
		}
	})

	_ = apperror.PublicFrom // keep import referenced for readers
}
