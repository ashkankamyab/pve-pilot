package jobs

import (
	"sync"
	"time"
)

// Store holds jobs in memory and notifies subscribers of updates.
type Store struct {
	mu          sync.RWMutex
	jobs        map[string]*Job
	subscribers map[string][]chan JobEvent
}

func NewStore() *Store {
	return &Store{
		jobs:        make(map[string]*Job),
		subscribers: make(map[string][]chan JobEvent),
	}
}

func (s *Store) Create(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	s.jobs[job.ID] = job
}

func (s *Store) Get(id string) *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if j, ok := s.jobs[id]; ok {
		copy := *j
		return &copy
	}
	return nil
}

func (s *Store) List() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		copy := *j
		result = append(result, &copy)
	}
	return result
}

// UpdateStep advances a job's step and notifies subscribers.
func (s *Store) UpdateStep(id string, step StepName, status JobStatus) {
	s.mu.Lock()
	j, ok := s.jobs[id]
	if !ok {
		s.mu.Unlock()
		return
	}
	j.Step = step
	j.Status = status
	j.Progress = StepProgress[step]
	j.UpdatedAt = time.Now()
	event := JobEvent{
		JobID:    id,
		Status:   j.Status,
		Step:     j.Step,
		Progress: j.Progress,
		IP:       j.IPAddress,
	}
	subs := s.subscribers[id]
	s.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// subscriber not keeping up, skip
		}
	}
}

// SetError marks a job as failed and notifies subscribers.
func (s *Store) SetError(id string, errMsg string) {
	s.mu.Lock()
	j, ok := s.jobs[id]
	if !ok {
		s.mu.Unlock()
		return
	}
	j.Status = StatusFailed
	j.Error = errMsg
	j.UpdatedAt = time.Now()
	event := JobEvent{
		JobID:    id,
		Status:   StatusFailed,
		Step:     j.Step,
		Progress: j.Progress,
		Error:    errMsg,
	}
	subs := s.subscribers[id]
	s.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

// SetIP sets the IP address on a job and notifies.
func (s *Store) SetIP(id string, ip string) {
	s.mu.Lock()
	j, ok := s.jobs[id]
	if !ok {
		s.mu.Unlock()
		return
	}
	j.IPAddress = ip
	j.UpdatedAt = time.Now()
	s.mu.Unlock()
}

// Subscribe returns a channel that receives events for a job.
// Call Unsubscribe when done.
func (s *Store) Subscribe(id string) chan JobEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan JobEvent, 16)
	s.subscribers[id] = append(s.subscribers[id], ch)
	return ch
}

// Unsubscribe removes a subscriber channel.
func (s *Store) Unsubscribe(id string, ch chan JobEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	subs := s.subscribers[id]
	for i, sub := range subs {
		if sub == ch {
			s.subscribers[id] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}
