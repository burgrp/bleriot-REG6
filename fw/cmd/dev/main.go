// Command dev is the local REG6 inventory and BleRiot CLI.
package main

import (
	"github.com/burgrp/bleriot-REG6/fw/spec"
	"github.com/burgrp/bleriot/lib/shared/config"
	"github.com/burgrp/bleriot/lib/shared/inventory"
	"github.com/burgrp/bleriot/lib/site/cli"
)

var rfChannel = inventory.Channel{
	Name:         "reg6",
	Number:       37,
	SpreadFactor: config.SpreadFactorS8,
}

func main() {
	cli.Start(inventory.Inventory{
		{
			Name:    "ssr",
			Address: [4]byte{0x5B, 0x2C, 0x0F, 0xBF},
			Key:     [16]byte{0x1F, 0xA8, 0xE0, 0xCB, 0x75, 0x16, 0x8D, 0x04, 0x02, 0xBB, 0x56, 0x98, 0xD8, 0xF1, 0x19, 0x41},
			Channel: rfChannel,
			Type:    spec.Type(),
			Config:  spec.Config{},
		},
	})
}
