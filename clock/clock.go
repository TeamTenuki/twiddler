package clock

import "time"

var clock Clock = &timeClockT{}

// Clock is a time source.
type Clock interface {
	NowUTC() time.Time
}

// NowUTC returns current time. The value is provided by time.Now
// unless the package clock is overriden with FixedClock.
func NowUTC() time.Time {
	return clock.NowUTC()
}

// Since returns the time difference between now (possibly overriden) and
// a given time t.
func Since(t time.Time) time.Duration {
	return NowUTC().Sub(t)
}

// OverrideClock allows to override the time source.
// Calling function with `nil' resets the time source to default (time.Now).
func OverrideClock(c Clock) {
	if c == nil {
		clock = &timeClockT{}
	} else {
		clock = c
	}
}

// OverrideByFixed overrides current time source with FixedClock
// of specified time. The time is converted to UTC before overriding.
// Pointer to created FixedClock is returned.
func OverrideByFixed(t time.Time) *FixedClock {
	fixedClock := &FixedClock{t: t.UTC()}
	OverrideClock(fixedClock)
	return fixedClock
}

type timeClockT struct{}

func (c *timeClockT) NowUTC() time.Time {
	return time.Now().UTC()
}

// FixedClock is an implementation of Clock that returns a fixed time
// on every NowUTC call. The time may be adjusted manually after creation.
// This should be mainly used for testing purposes.
type FixedClock struct {
	t time.Time
}

func (c *FixedClock) NowUTC() time.Time {
	return c.t
}

// Set time to a specific value (this always resets passed in time to UTC).
func (c *FixedClock) Set(t time.Time) {
	c.t = t.UTC()
}

// Add moves time forward by duration d.
func (c *FixedClock) Add(d time.Duration) {
	c.t = c.t.Add(d)
}

// Sub moves time backward by duration d.
func (c *FixedClock) Sub(d time.Duration) {
	c.t = c.t.Add(-d)
}
