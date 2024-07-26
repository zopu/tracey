package store

import (
	"sync"

	"github.com/zopu/tracey/internal/xray"
)

type Store struct {
	mu         sync.Mutex
	summaries  []xray.TraceSummary
	summaryIDs map[string]struct{}
}

func New() Store {
	return Store{
		summaries:  []xray.TraceSummary{},
		summaryIDs: map[string]struct{}{},
	}
}

func (s *Store) GetTraceSummaries() []xray.TraceSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]xray.TraceSummary{}, s.summaries...)
}

func (s *Store) AddTraceSummaries(summaries []xray.TraceSummary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, summary := range summaries {
		if _, ok := s.summaryIDs[summary.ID()]; !ok {
			s.summaries = append(s.summaries, summary)
			s.summaryIDs[summary.ID()] = struct{}{}
		}
	}
}

func (s *Store) Size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.summaries)
}
