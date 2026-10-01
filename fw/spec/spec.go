package spec

import (
	"github.com/burgrp/bleriot/lib/shared/firmware"
	"github.com/burgrp/bleriot/lib/shared/inventory"
	"github.com/burgrp/bleriot/lib/shared/puya"
)

type Config struct{}

const (
	RegChannel1 uint16 = iota + 1
	RegChannel2
	RegChannel3
	RegChannel4
	RegChannel5
	RegChannel6
	RegWatchdog uint16 = 7

	WireFullScale      int32 = 100
	WatchdogMaxSeconds int32 = 60
)

var ChannelTags = [...]uint16{
	RegChannel1,
	RegChannel2,
	RegChannel3,
	RegChannel4,
	RegChannel5,
	RegChannel6,
}

var Chip = puya.PY32F030x8

func Type() inventory.DeviceType {
	return inventory.DeviceType{
		Name: "reg6",
		Chip: Chip,
		Firmware: firmware.Manifest{
			Package: "github.com/burgrp/bleriot-REG6/fw",
			TinyGo: firmware.TinyGoProfile{
				Scheduler:        firmware.SchedulerTasks,
				StackSizeBytes:   1024,
				GarbageCollector: firmware.GCLeaking,
				Serial:           firmware.SerialRTT,
				SizeReport:       firmware.SizeReportHTML,
				PrintAllocs:      true,
			},
			PyOCD: firmware.PyOCDProfile{
				RTTMode:                  firmware.ConnectUnderReset,
				Reclaim:                  true,
				ReclaimDelayMilliseconds: 1000,
			},
		},
		Registers: []inventory.Register{
			channelRegister(RegChannel1, "channel.1"),
			channelRegister(RegChannel2, "channel.2"),
			channelRegister(RegChannel3, "channel.3"),
			channelRegister(RegChannel4, "channel.4"),
			channelRegister(RegChannel5, "channel.5"),
			channelRegister(RegChannel6, "channel.6"),
			{
				Tag:  RegWatchdog,
				Name: "watchdog",
				Type: inventory.TypeInt,
				Metadata: map[string]string{
					"min":  "0",
					"max":  "60",
					"unit": "seconds",
				},
			},
		},
	}
}

func channelRegister(tag uint16, name string) inventory.Register {
	return inventory.Register{
		Tag:  tag,
		Name: name,
		Type: inventory.TypeInt,
		Metadata: map[string]string{
			"min":  "0",
			"max":  "100",
			"unit": "percent",
		},
	}
}
