package reg6

import (
	"image/color"

	"github.com/burgrp/bleriot-REG6/fw/spec"
)

const (
	channelCount    = len(spec.ChannelTags)
	pwmFrequencyHz  = 24_000
	pwmTimerClockHz = 24_000_000
	pwmPeriodTicks  = pwmTimerClockHz / pwmFrequencyHz
)

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

func statusColors(online, pulse bool, levels [channelCount]int32) [channelCount]color.RGBA {
	var colors [channelCount]color.RGBA
	if !online {
		if pulse {
			colors[0] = color.RGBA{R: 0xFF, G: 0x30, B: 0x00, A: 0xFF}
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
	return color.RGBA{R: red, B: 255 - red, A: 255}
}
