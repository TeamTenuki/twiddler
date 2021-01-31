package clock

import (
	"testing"
	"time"
)

func TestFixedClockOverride(t *testing.T) {
	now := time.Now()
	fixedTime := OverrideByFixed(now)

	if NowUTC() != now.UTC() {
		t.Errorf("Override failed: %q != %q", NowUTC(), now.UTC())
	}

	fixedTime.Add(time.Hour)

	if NowUTC() != now.UTC().Add(time.Hour) {
		t.Errorf("Time adjustment failed: %q != %q", NowUTC(), now.UTC().Add(time.Hour))
	}
}
