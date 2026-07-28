package sell

import (
	"testing"
	"time"

	"github.com/example/crypto-spot-signal/internal/outcome"
	"github.com/example/crypto-spot-signal/internal/storage"
)

func TestEvaluateDirectionalAccurateOnDecline(t *testing.T) {
	c := storage.SellCandidate{SignalID: "s1", Symbol: "BTC_USDT", EntryPrice: 100, Target1: 95, Invalidation: 105, CreatedAt: time.Now()}
	obs := outcome.PriceObservation{High: 101, Low: 92}
	result := EvaluateDirectional(c, obs, 93, time.Now())

	if !result.DirectionalAccuracy {
		t.Fatal("expected directional accuracy=true when price declined")
	}
	if !result.BreakdownFollowThrough {
		t.Fatal("expected breakdown follow-through when low crossed target1")
	}
	if result.Invalidated {
		t.Fatal("expected not invalidated when price stayed below invalidation level")
	}
}

func TestEvaluateDirectionalInvalidatedOnReclaim(t *testing.T) {
	c := storage.SellCandidate{SignalID: "s2", Symbol: "BTC_USDT", EntryPrice: 100, Target1: 95, Invalidation: 105, CreatedAt: time.Now()}
	obs := outcome.PriceObservation{High: 110, Low: 98}
	result := EvaluateDirectional(c, obs, 106, time.Now())

	if !result.Invalidated {
		t.Fatal("expected invalidated when price reclaimed the invalidation level")
	}
	if result.DirectionalAccuracy {
		t.Fatal("expected directional accuracy=false when price rose above entry")
	}
}

func TestEvaluateDirectionalSupportReclaim(t *testing.T) {
	c := storage.SellCandidate{SignalID: "s3", Symbol: "BTC_USDT", EntryPrice: 100, Target1: 95, Invalidation: 105, CreatedAt: time.Now()}
	obs := outcome.PriceObservation{High: 101, Low: 90}
	result := EvaluateDirectional(c, obs, 100.5, time.Now())

	if !result.SupportReclaim {
		t.Fatal("expected support reclaim when low dipped below entry then price recovered to/above entry")
	}
}
