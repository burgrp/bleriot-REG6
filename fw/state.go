package reg6

import (
	"image/color"

	"github.com/burgrp/bleriot-REG6/fw/spec"
)

const (
	channelCount         = len(spec.ChannelTags)
	nanosecondsPerSecond = int64(1_000_000_000)
	pwmFrequencyHz       = 24_000
	pwmTimerClockHz      = 24_000_000
	pwmPeriodTicks       = pwmTimerClockHz / pwmFrequencyHz
)

type controllerState struct {
	channels channelState
	watchdog watchdogLease
}

func (state *controllerState) read(tag uint16, now int64) (value int32, null, forceOff bool) {
	forceOff = state.expireWatchdog(now)
	if tag == spec.RegWatchdog {
		return state.watchdog.remaining(now), false, forceOff
	}
	value, null = state.channels.read(tag)
	return value, null, forceOff
}

func (state *controllerState) write(tag uint16, value int32, null bool, now int64) (levelsChanged, forceOff bool) {
	forceOff = state.expireWatchdog(now)
	if tag == spec.RegWatchdog {
		if !state.watchdog.set(value, null, now) {
			state.channels.clear()
			return false, true
		}
		return false, forceOff
	}
	if !state.watchdog.active(now) {
		return false, forceOff
	}
	return state.channels.write(tag, value, null), forceOff
}

func (state *controllerState) expireWatchdog(now int64) bool {
	if !state.watchdog.expire(now) {
		return false
	}
	state.channels.clear()
	return true
}

type watchdogLease struct {
	deadline int64
}

func (lease *watchdogLease) set(value int32, null bool, now int64) bool {
	if null || value <= 0 {
		lease.deadline = 0
		return false
	}
	if value > spec.WatchdogMaxSeconds {
		value = spec.WatchdogMaxSeconds
	}
	lease.deadline = now + int64(value)*nanosecondsPerSecond
	return true
}

func (lease *watchdogLease) active(now int64) bool {
	return lease.deadline != 0 && lease.deadline > now
}

func (lease *watchdogLease) remaining(now int64) int32 {
	if !lease.active(now) {
		return 0
	}
	remaining := lease.deadline - now
	return int32((remaining + nanosecondsPerSecond - 1) / nanosecondsPerSecond)
}

func (lease *watchdogLease) expire(now int64) bool {
	if lease.deadline == 0 || lease.deadline > now {
		return false
	}
	lease.deadline = 0
	return true
}

type channelState struct {
	levels [channelCount]int32
}

func (state *channelState) read(tag uint16) (int32, bool) {
	index, ok := channelIndex(tag)
	if !ok {
		return 0, true
	}
	return state.levels[index], false
}

func (state *channelState) write(tag uint16, value int32, null bool) bool {
	index, ok := channelIndex(tag)
	if !ok {
		return false
	}
	if null {
		value = 0
	}
	value = clampLevel(value)
	if state.levels[index] == value {
		return false
	}
	state.levels[index] = value
	return true
}

func (state *channelState) clear() bool {
	changed := false
	for index := range state.levels {
		if state.levels[index] != 0 {
			state.levels[index] = 0
			changed = true
		}
	}
	return changed
}

func channelIndex(tag uint16) (int, bool) {
	for index, candidate := range spec.ChannelTags {
		if candidate == tag {
			return index, true
		}
	}
	return 0, false
}

func clampLevel(value int32) int32 {
	if value < 0 {
		return 0
	}
	if value > spec.WireFullScale {
		return spec.WireFullScale
	}
	return value
}

func levelCompare(level int32) uint32 {
	level = clampLevel(level)
	return uint32((int64(level)*pwmPeriodTicks + int64(spec.WireFullScale)/2) / int64(spec.WireFullScale))
}

func statusColors(online, pulse, watchdogExpired bool, levels [channelCount]int32) [channelCount]color.RGBA {
	var colors [channelCount]color.RGBA
	if !online {
		if pulse {
			colors[0] = color.RGBA{R: 0xFF, G: 0x30, B: 0x00, A: 0xFF}
		}
		return colors
	}
	if watchdogExpired {
		if pulse {
			colors[1] = color.RGBA{B: 0xFF, A: 0xFF}
		}
		return colors
	}

	for index, level := range levels {
		colors[index] = levelColor(level)
	}
	return colors
}

func levelColor(level int32) color.RGBA {
	level = clampLevel(level)
	red := uint8((int64(level)*255 + int64(spec.WireFullScale)/2) / int64(spec.WireFullScale))
	// Normal mode runs at half the status-indicator intensity.
	return color.RGBA{R: red / 2, G: 50, B: (255 - red) / 2, A: 255}
}
