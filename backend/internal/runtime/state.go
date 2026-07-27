package runtime

import (
	"sort"
	"strings"
	"sync"

	"github.com/example/crypto-spot-signal/internal/domain"
)

type State struct {
	mu       sync.RWMutex
	features []domain.FeatureSnapshot
}

func NewState() *State {
	return &State{}
}

func (s *State) SetFeatures(features []domain.FeatureSnapshot) {
	copyOf := append([]domain.FeatureSnapshot(nil), features...)
	sort.Slice(copyOf, func(i, j int) bool {
		return copyOf[i].RuleScore > copyOf[j].RuleScore
	})
	s.mu.Lock()
	s.features = copyOf
	s.mu.Unlock()
}

func (s *State) Features() []domain.FeatureSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.FeatureSnapshot(nil), s.features...)
}

func (s *State) Feature(symbol string) (domain.FeatureSnapshot, bool) {
	symbol = strings.ToUpper(symbol)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, feature := range s.features {
		if feature.Symbol == symbol {
			return feature, true
		}
	}
	return domain.FeatureSnapshot{}, false
}
