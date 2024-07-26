package store

import (
	"sync"

	"github.com/zopu/tracey/internal/aws"
)

type Store struct {
	mu         sync.Mutex
	summaries  []aws.TraceSummary
	summaryIDs map[string]struct{}
}

func New() Store {
	return Store{
		summaries:  []aws.TraceSummary{},
		summaryIDs: map[string]struct{}{},
	}
}

func (s *Store) GetTraceSummaries() []aws.TraceSummary {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]aws.TraceSummary{}, s.summaries...)
}

func (s *Store) AddTraceSummaries(summaries []aws.TraceSummary) {
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
