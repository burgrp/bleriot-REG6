package reg6

import (
	"image/color"
	"testing"

	"github.com/burgrp/bleriot-REG6/fw/spec"
)

func TestChannelStateReadWriteAndClamp(t *testing.T) {
	var state channelState
	if changed := state.write(spec.RegChannel1, 25, false); !changed {
		t.Fatal("first write did not report a change")
	}
	if got, null := state.read(spec.RegChannel1); got != 25 || null {
		t.Fatalf("channel 1 read = (%d, %v), want (25, false)", got, null)
	}

	state.write(spec.RegChannel2, -1, false)
	state.write(spec.RegChannel3, spec.WireFullScale+1, false)
	if got, _ := state.read(spec.RegChannel2); got != 0 {
		t.Errorf("negative write stored %d, want 0", got)
	}
	if got, _ := state.read(spec.RegChannel3); got != spec.WireFullScale {
		t.Errorf("high write stored %d, want %d", got, spec.WireFullScale)
	}

	state.write(spec.RegChannel1, 123, true)
	if got, _ := state.read(spec.RegChannel1); got != 0 {
		t.Errorf("NULL write stored %d, want 0", got)
	}
	if got, null := state.read(999); got != 0 || !null {
		t.Errorf("unknown read = (%d, %v), want (0, true)", got, null)
	}
	if changed := state.write(999, 100, false); changed {
		t.Error("unknown write reported a change")
	}
}

func TestChannelStateClearRequiresFreshWrites(t *testing.T) {
	var state channelState
	for _, tag := range spec.ChannelTags {
		state.write(tag, spec.WireFullScale/2, false)
	}
	if changed := state.clear(); !changed {
		t.Fatal("clear did not report a change")
	}
	for _, tag := range spec.ChannelTags {
		if got, _ := state.read(tag); got != 0 {
			t.Errorf("tag %d retained %d after clear", tag, got)
		}
	}
	if changed := state.clear(); changed {
		t.Error("clearing an already clear state reported a change")
	}
}

func TestWatchdogLeaseClampsAndRoundsRemainingUp(t *testing.T) {
	const now = int64(12_345)
	var lease watchdogLease
	if armed := lease.set(spec.WatchdogMaxSeconds+1, false, now); !armed {
		t.Fatal("positive watchdog write did not arm lease")
	}

	tests := []struct {
		name string
		now  int64
		want int32
	}{
		{name: "at set", now: now, want: 60},
		{name: "fractional first second", now: now + 1, want: 60},
		{name: "one second elapsed", now: now + nanosecondsPerSecond, want: 59},
		{name: "fractional final second", now: now + 59*nanosecondsPerSecond + 1, want: 1},
		{name: "at deadline", now: now + 60*nanosecondsPerSecond, want: 0},
		{name: "after deadline", now: now + 61*nanosecondsPerSecond, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := lease.remaining(test.now); got != test.want {
				t.Errorf("remaining(%d) = %d, want %d", test.now, got, test.want)
			}
		})
	}
}

func TestControllerWatchdogSafetyLifecycle(t *testing.T) {
	const now = int64(1_000)
	var state controllerState

	if changed, forceOff := state.write(spec.RegChannel1, 50, false, now); changed || forceOff {
		t.Fatalf("channel write while expired = (%v, %v), want (false, false)", changed, forceOff)
	}
	if got, null, forceOff := state.read(spec.RegWatchdog, now); got != 0 || null || forceOff {
		t.Fatalf("initial watchdog read = (%d, %v, %v), want (0, false, false)", got, null, forceOff)
	}

	if changed, forceOff := state.write(spec.RegWatchdog, 2, false, now); changed || forceOff {
		t.Fatalf("watchdog arm = (%v, %v), want (false, false)", changed, forceOff)
	}
	for _, tag := range spec.ChannelTags {
		if changed, forceOff := state.write(tag, 50, false, now); !changed || forceOff {
			t.Fatalf("armed channel %d write = (%v, %v), want (true, false)", tag, changed, forceOff)
		}
	}
	if got, null, forceOff := state.read(spec.RegWatchdog, now+nanosecondsPerSecond+1); got != 1 || null || forceOff {
		t.Fatalf("counted-down watchdog read = (%d, %v, %v), want (1, false, false)", got, null, forceOff)
	}

	deadline := now + 2*nanosecondsPerSecond
	if got, null, forceOff := state.read(spec.RegChannel1, deadline); got != 0 || null || !forceOff {
		t.Fatalf("channel read at expiry = (%d, %v, %v), want (0, false, true)", got, null, forceOff)
	}
	for _, tag := range spec.ChannelTags {
		if got, null, forceOff := state.read(tag, deadline); got != 0 || null || forceOff {
			t.Errorf("channel %d after expiry = (%d, %v, %v), want (0, false, false)", tag, got, null, forceOff)
		}
	}
	if got, null, forceOff := state.read(spec.RegWatchdog, deadline); got != 0 || null || forceOff {
		t.Fatalf("expired watchdog read = (%d, %v, %v), want (0, false, false)", got, null, forceOff)
	}
	if changed, forceOff := state.write(spec.RegChannel2, 75, false, deadline+1); changed || forceOff {
		t.Fatalf("channel write after expiry = (%v, %v), want (false, false)", changed, forceOff)
	}

	if changed, forceOff := state.write(spec.RegWatchdog, 5, false, deadline+1); changed || forceOff {
		t.Fatalf("watchdog re-arm = (%v, %v), want (false, false)", changed, forceOff)
	}
	if got, _, _ := state.read(spec.RegChannel1, deadline+1); got != 0 {
		t.Fatalf("re-arm restored channel 1 to %d, want 0", got)
	}
	if changed, forceOff := state.write(spec.RegChannel2, 75, false, deadline+1); !changed || forceOff {
		t.Fatalf("fresh channel write after re-arm = (%v, %v), want (true, false)", changed, forceOff)
	}
	if changed, forceOff := state.write(spec.RegWatchdog, 0, false, deadline+1); changed || !forceOff {
		t.Fatalf("zero watchdog write = (%v, %v), want (false, true)", changed, forceOff)
	}
	if got, _, _ := state.read(spec.RegChannel2, deadline+1); got != 0 {
		t.Fatalf("zero watchdog retained channel 2 at %d, want 0", got)
	}
}

func TestWatchdogNegativeAndNullWritesExpire(t *testing.T) {
	for _, test := range []struct {
		name  string
		value int32
		null  bool
	}{
		{name: "negative", value: -1},
		{name: "null", value: 30, null: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var state controllerState
			state.write(spec.RegWatchdog, 30, false, 0)
			state.write(spec.RegChannel6, 100, false, 0)
			if changed, forceOff := state.write(spec.RegWatchdog, test.value, test.null, 1); changed || !forceOff {
				t.Fatalf("expiry write = (%v, %v), want (false, true)", changed, forceOff)
			}
			if got, _, _ := state.read(spec.RegChannel6, 1); got != 0 {
				t.Errorf("channel 6 after expiry = %d, want 0", got)
			}
		})
	}
}

func TestControllerStateDoesNotAllocate(t *testing.T) {
	var state controllerState
	state.write(spec.RegWatchdog, spec.WatchdogMaxSeconds, false, 0)

	if allocs := testing.AllocsPerRun(1000, func() {
		state.write(spec.RegChannel1, 50, false, 1)
		state.read(spec.RegWatchdog, 1)
	}); allocs != 0 {
		t.Fatalf("steady-state controller operations allocated %v times, want 0", allocs)
	}
}

func TestLevelColorClampsAndIsOpaque(t *testing.T) {
	if got, want := levelColor(-1), levelColor(0); got != want {
		t.Errorf("negative level color = %#v, want zero-level color %#v", got, want)
	}
	if got, want := levelColor(spec.WireFullScale+1), levelColor(spec.WireFullScale); got != want {
		t.Errorf("high level color = %#v, want full-scale color %#v", got, want)
	}
	for _, level := range []int32{0, spec.WireFullScale / 2, spec.WireFullScale} {
		got := levelColor(level)
		if got.G != 50 {
			t.Errorf("levelColor(%d) green = %d, want 50", level, got.G)
		}
		if got.A != 0xFF {
			t.Errorf("levelColor(%d) alpha = %d, want 255", level, got.A)
		}
	}
}

func TestLevelCompare(t *testing.T) {
	tests := []struct {
		level int32
		want  uint32
	}{
		{level: -1, want: 0},
		{level: 0, want: 0},
		{level: spec.WireFullScale / 2, want: pwmPeriodTicks / 2},
		{level: spec.WireFullScale, want: pwmPeriodTicks},
		{level: spec.WireFullScale + 1, want: pwmPeriodTicks},
	}
	for _, test := range tests {
		if got := levelCompare(test.level); got != test.want {
			t.Errorf("levelCompare(%d) = %d, want %d", test.level, got, test.want)
		}
	}
}

func TestPWMConfiguration(t *testing.T) {
	if pwmFrequencyHz != 24_000 {
		t.Fatalf("PWM frequency = %d Hz, want 24000 Hz", pwmFrequencyHz)
	}
	if pwmPeriodTicks != 1_000 {
		t.Fatalf("PWM period = %d ticks, want 1000 ticks", pwmPeriodTicks)
	}
}

func TestStatusColors(t *testing.T) {
	levels := [channelCount]int32{0, 20, 40, 60, 80, spec.WireFullScale}
	online := statusColors(true, true, false, levels)
	for index, level := range levels {
		if got, want := online[index], levelColor(level); got != want {
			t.Errorf("online LED %d = %#v, want %#v", index+1, got, want)
		}
	}

	watchdogPulse := statusColors(true, true, true, levels)
	for index, got := range watchdogPulse {
		want := color.RGBA{}
		if index == 1 {
			want = color.RGBA{B: 0xFF, A: 0xFF}
		}
		if got != want {
			t.Errorf("watchdog pulse LED %d = %#v, want %#v", index+1, got, want)
		}
	}
	watchdogIdle := statusColors(true, false, true, levels)
	for index, got := range watchdogIdle {
		if got != (color.RGBA{}) {
			t.Errorf("watchdog idle LED %d = %#v, want off", index+1, got)
		}
	}

	offlinePulse := statusColors(false, true, true, levels)
	if got, want := offlinePulse[0], (color.RGBA{R: 0xFF, G: 0x30, B: 0x00, A: 0xFF}); got != want {
		t.Errorf("offline LED 1 = %#v, want %#v", got, want)
	}
	for index := 1; index < channelCount; index++ {
		if got := offlinePulse[index]; got != (color.RGBA{}) {
			t.Errorf("offline LED %d = %#v, want off", index+1, got)
		}
	}

	offlineIdle := statusColors(false, false, false, levels)
	for index, got := range offlineIdle {
		if got != (color.RGBA{}) {
			t.Errorf("offline idle LED %d = %#v, want off", index+1, got)
		}
	}
}
