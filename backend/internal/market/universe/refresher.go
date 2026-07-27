package universe

import (
	"context"
	"log"
	"time"
)

func (s *Service) StartRefresher(ctx context.Context) {
	// Initial refresh
	if err := s.Refresh(ctx); err != nil {
		log.Printf("universe: initial refresh failed: %v", err)
	}

	ticker := time.NewTicker(s.cfg.PairUniverseRefreshMin)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Refresh(ctx); err != nil {
				log.Printf("universe: refresh failed: %v", err)
			}
		}
	}
}
