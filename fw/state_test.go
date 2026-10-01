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

func TestLevelColor(t *testing.T) {
	tests := []struct {
		name  string
		level int32
		want  color.RGBA
	}{
		{name: "zero is blue", level: 0, want: color.RGBA{B: 255, A: 255}},
		{name: "half is midpoint", level: spec.WireFullScale / 2, want: color.RGBA{R: 128, B: 127, A: 255}},
		{name: "full is red", level: spec.WireFullScale, want: color.RGBA{R: 255, A: 255}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := levelColor(test.level); got != test.want {
				t.Errorf("levelColor(%d) = %#v, want %#v", test.level, got, test.want)
			}
		})
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
	online := statusColors(true, true, levels)
	for index, level := range levels {
		if got, want := online[index], levelColor(level); got != want {
			t.Errorf("online LED %d = %#v, want %#v", index+1, got, want)
		}
	}

	offlinePulse := statusColors(false, true, levels)
	if got, want := offlinePulse[0], (color.RGBA{R: 0xFF, G: 0x30, B: 0x00, A: 0xFF}); got != want {
		t.Errorf("offline LED 1 = %#v, want %#v", got, want)
	}
	for index := 1; index < channelCount; index++ {
		if got := offlinePulse[index]; got != (color.RGBA{}) {
			t.Errorf("offline LED %d = %#v, want off", index+1, got)
		}
	}

	offlineIdle := statusColors(false, false, levels)
	for index, got := range offlineIdle {
		if got != (color.RGBA{}) {
			t.Errorf("offline idle LED %d = %#v, want off", index+1, got)
		}
	}
}
