package library

import (
	"testing"
	"time"
)

func TestManagerWriteWaitsForReader(t *testing.T) {
	m := NewManager()
	releaseRead := m.AcquireRead("library-1")
	acquired := make(chan struct{})
	go func() {
		release := m.AcquireWrite("library-1")
		close(acquired)
		release()
	}()
	select {
	case <-acquired:
		t.Fatal("writer acquired while reader held")
	case <-time.After(25 * time.Millisecond):
	}
	releaseRead()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("writer never acquired")
	}
}

func TestManagerReleaseIsIdempotent(t *testing.T) {
	m := NewManager()
	release := m.AcquireWrite("library-1")
	release()
	release() // second call must be a no-op, not a panic or double-unlock

	releaseRead := m.AcquireRead("library-2")
	releaseRead()
	releaseRead()
}

func TestManagerIndependentLibrariesDoNotBlock(t *testing.T) {
	m := NewManager()
	releaseA := m.AcquireWrite("library-a")
	defer releaseA()

	acquired := make(chan struct{})
	go func() {
		release := m.AcquireWrite("library-b")
		close(acquired)
		release()
	}()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("independent library blocked by unrelated writer")
	}
}

func TestManagerReadersShareTheLibrary(t *testing.T) {
	m := NewManager()
	releaseR1 := m.AcquireRead("library-1")
	defer releaseR1()

	acquired := make(chan struct{})
	go func() {
		release := m.AcquireRead("library-1")
		close(acquired)
		release()
	}()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second reader blocked by first reader")
	}
}

func TestManagerWriterExcludesLaterReaders(t *testing.T) {
	m := NewManager()
	releaseWrite := m.AcquireWrite("library-1")
	acquired := make(chan struct{})
	go func() {
		release := m.AcquireRead("library-1")
		close(acquired)
		release()
	}()
	select {
	case <-acquired:
		t.Fatal("reader acquired while writer held")
	case <-time.After(25 * time.Millisecond):
	}
	releaseWrite()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("reader never acquired after write release")
	}
}
