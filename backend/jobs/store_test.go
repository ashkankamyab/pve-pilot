package jobs

import (
	"sync"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
	if len(s.List()) != 0 {
		t.Fatal("new store should be empty")
	}
}

func TestCreateAndGet(t *testing.T) {
	s := NewStore()
	job := &Job{ID: "j1", Type: "vm", Status: StatusPending, Name: "test-vm"}
	s.Create(job)

	got := s.Get("j1")
	if got == nil {
		t.Fatal("expected job, got nil")
	}
	if got.Name != "test-vm" {
		t.Errorf("expected name test-vm, got %s", got.Name)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestGetReturnsNilForMissing(t *testing.T) {
	s := NewStore()
	if s.Get("nonexistent") != nil {
		t.Error("expected nil for missing job")
	}
}

func TestGetReturnsCopy(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1", Name: "original"})

	got := s.Get("j1")
	got.Name = "modified"

	got2 := s.Get("j1")
	if got2.Name != "original" {
		t.Error("Get should return a copy, not a reference to internal state")
	}
}

func TestList(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1"})
	s.Create(&Job{ID: "j2"})
	s.Create(&Job{ID: "j3"})

	list := s.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(list))
	}

	ids := map[string]bool{}
	for _, j := range list {
		ids[j.ID] = true
	}
	for _, id := range []string{"j1", "j2", "j3"} {
		if !ids[id] {
			t.Errorf("missing job %s in list", id)
		}
	}
}

func TestListReturnsCopies(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1", Name: "original"})

	list := s.List()
	list[0].Name = "modified"

	got := s.Get("j1")
	if got.Name != "original" {
		t.Error("List should return copies")
	}
}

func TestUpdateStep(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1", Status: StatusPending})

	s.UpdateStep("j1", StepCloning, StatusRunning)

	got := s.Get("j1")
	if got.Step != StepCloning {
		t.Errorf("expected step %s, got %s", StepCloning, got.Step)
	}
	if got.Status != StatusRunning {
		t.Errorf("expected status %s, got %s", StatusRunning, got.Status)
	}
	if got.Progress != StepProgress[StepCloning] {
		t.Errorf("expected progress %d, got %d", StepProgress[StepCloning], got.Progress)
	}
}

func TestUpdateStepNonexistentJob(t *testing.T) {
	s := NewStore()
	// Should not panic
	s.UpdateStep("nonexistent", StepCloning, StatusRunning)
}

func TestSetError(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1", Status: StatusRunning, Step: StepCloning})

	s.SetError("j1", "clone failed: disk full")

	got := s.Get("j1")
	if got.Status != StatusFailed {
		t.Errorf("expected status %s, got %s", StatusFailed, got.Status)
	}
	if got.Error != "clone failed: disk full" {
		t.Errorf("unexpected error: %s", got.Error)
	}
}

func TestSetErrorNonexistentJob(t *testing.T) {
	s := NewStore()
	// Should not panic
	s.SetError("nonexistent", "some error")
}

func TestSetIP(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1"})

	s.SetIP("j1", "192.168.1.100")

	got := s.Get("j1")
	if got.IPAddress != "192.168.1.100" {
		t.Errorf("expected IP 192.168.1.100, got %s", got.IPAddress)
	}
}

func TestSetIPNonexistentJob(t *testing.T) {
	s := NewStore()
	// Should not panic
	s.SetIP("nonexistent", "1.2.3.4")
}

func TestSubscribeReceivesUpdateStepEvents(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1", Status: StatusPending})

	ch := s.Subscribe("j1")
	defer s.Unsubscribe("j1", ch)

	s.UpdateStep("j1", StepCloning, StatusRunning)

	select {
	case event := <-ch:
		if event.JobID != "j1" {
			t.Errorf("expected job_id j1, got %s", event.JobID)
		}
		if event.Step != StepCloning {
			t.Errorf("expected step %s, got %s", StepCloning, event.Step)
		}
		if event.Status != StatusRunning {
			t.Errorf("expected status %s, got %s", StatusRunning, event.Status)
		}
		if event.Progress != StepProgress[StepCloning] {
			t.Errorf("expected progress %d, got %d", StepProgress[StepCloning], event.Progress)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestSubscribeReceivesErrorEvents(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1", Status: StatusRunning})

	ch := s.Subscribe("j1")
	defer s.Unsubscribe("j1", ch)

	s.SetError("j1", "something broke")

	select {
	case event := <-ch:
		if event.Status != StatusFailed {
			t.Errorf("expected status failed, got %s", event.Status)
		}
		if event.Error != "something broke" {
			t.Errorf("unexpected error: %s", event.Error)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error event")
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1"})

	ch := s.Subscribe("j1")
	s.Unsubscribe("j1", ch)

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1"})

	ch1 := s.Subscribe("j1")
	ch2 := s.Subscribe("j1")
	defer s.Unsubscribe("j1", ch1)
	defer s.Unsubscribe("j1", ch2)

	s.UpdateStep("j1", StepStarting, StatusRunning)

	for _, ch := range []chan JobEvent{ch1, ch2} {
		select {
		case event := <-ch:
			if event.Step != StepStarting {
				t.Errorf("expected step starting, got %s", event.Step)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event on subscriber")
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1", Status: StatusPending})

	var wg sync.WaitGroup
	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Get("j1")
			s.List()
		}()
	}
	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.UpdateStep("j1", StepCloning, StatusRunning)
			s.SetIP("j1", "10.0.0.1")
		}()
	}
	wg.Wait()
}

func TestStepProgressMapping(t *testing.T) {
	// Verify all defined steps have progress values
	steps := []StepName{
		StepCloning, StepConfiguring, StepResizing, StepAddingDisks,
		StepStarting, StepWaitingRun, StepReady,
		StepBackingUp,
		StepStopping, StepDeleting, StepRestoring,
	}
	for _, step := range steps {
		if _, ok := StepProgress[step]; !ok {
			t.Errorf("step %s has no progress mapping", step)
		}
	}

	// StepReady should be 100
	if StepProgress[StepReady] != 100 {
		t.Errorf("StepReady progress should be 100, got %d", StepProgress[StepReady])
	}
}

func TestUpdateStepIPIncludedInEvent(t *testing.T) {
	s := NewStore()
	s.Create(&Job{ID: "j1"})
	s.SetIP("j1", "10.0.0.5")

	ch := s.Subscribe("j1")
	defer s.Unsubscribe("j1", ch)

	s.UpdateStep("j1", StepReady, StatusCompleted)

	select {
	case event := <-ch:
		if event.IP != "10.0.0.5" {
			t.Errorf("expected IP 10.0.0.5 in event, got %s", event.IP)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}
