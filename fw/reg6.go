//go:build tinygo

package reg6

import (
	"device/py32"
	"image/color"
	"machine"
	"runtime"
	"time"

	"github.com/burgrp/bleriot-REG6/fw/spec"
	"github.com/burgrp/bleriot/lib/node"
	"github.com/burgrp/bleriot/lib/node/pan211x"
	"github.com/burgrp/tinygo-drivers/ws2812"
)

const (
	pinPWM1 = machine.PA8  // TIM1_CH1, AF2
	pinPWM2 = machine.PA9  // TIM1_CH2, AF2
	pinPWM3 = machine.PA10 // TIM1_CH3, AF2
	pinPWM4 = machine.PB4  // TIM3_CH1, AF1
	pinPWM5 = machine.PB5  // TIM3_CH2, AF1
	pinPWM6 = machine.PF3  // TIM3_CH3, AF13

	pinRadioCS   = machine.PF1
	pinRadioSCK  = machine.PA2
	pinRadioData = machine.PA3
	pinStatusLED = machine.PF4

	ledBrightness = 128
)

type Device struct {
	state          controllerState
	pwm            pwmOutputs
	led            ws2812.Device
	lastLEDFrame   [channelCount]color.RGBA
	ledInitialized bool
}

// Run starts the REG6 firmware with its baked BleRiot identity.
func Run(provisioning node.Provisioning, _ spec.Config) {
	device, err := newDevice()
	if err != nil {
		halt(device, "failed to start status LEDs: "+err.Error())
		return
	}

	bleNode, err := pan211x.StartNode(provisioning, pinRadioSCK, pinRadioData, pinRadioCS, device)
	if err != nil {
		println("failed to start BleRiot node: " + err.Error())
		device.runOffline()
		return
	}

	for {
		online, pulse, _ := bleNode.PollWithStatus()
		if err := device.syncStatus(online, pulse); err != nil {
			halt(device, "failed to update status LEDs: "+err.Error())
			return
		}
		runtime.Gosched()
	}
}

func newDevice() (*Device, error) {
	device := &Device{}
	device.pwm.configure()

	pinStatusLED.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	pinStatusLED.Low()
	pinStatusLED.Configure(machine.PinConfig{Mode: machine.PinOutput})
	device.led = ws2812.NewWS2812(pinStatusLED)
	device.led.SetBrightness(ledBrightness)
	if err := device.writeLEDFrame(statusColors(false, false, true, device.state.channels.levels)); err != nil {
		return device, err
	}
	return device, nil
}

func (device *Device) Read(tag uint16) (value int32, null bool) {
	value, null, forceOff := device.state.read(tag, nanotime())
	if forceOff {
		device.pwm.off()
	}
	return value, null
}

func (device *Device) Write(tag uint16, value int32, null bool) {
	levelsChanged, forceOff := device.state.write(tag, value, null, nanotime())
	if forceOff {
		device.pwm.off()
	} else if levelsChanged {
		device.pwm.apply(device.state.channels.levels)
	}
}

func (device *Device) syncStatus(online, pulse bool) error {
	now := nanotime()
	if device.state.expireWatchdog(now) {
		device.pwm.off()
	}
	if !online && device.state.channels.clear() {
		device.pwm.off()
	}
	return device.writeLEDFrame(statusColors(online, pulse, !device.state.watchdog.active(now), device.state.channels.levels))
}

func (device *Device) writeLEDFrame(frame [channelCount]color.RGBA) error {
	if device.ledInitialized && sameLEDFrame(device.lastLEDFrame, frame) {
		return nil
	}
	if err := device.led.WriteColors(frame[:]); err != nil {
		return err
	}
	device.lastLEDFrame = frame
	device.ledInitialized = true
	return nil
}

func (device *Device) runOffline() {
	device.state.channels.clear()
	device.pwm.off()
	for {
		if err := device.syncStatus(false, true); err != nil {
			halt(device, "failed to update offline status: "+err.Error())
			return
		}
		time.Sleep(200 * time.Millisecond)
		if err := device.syncStatus(false, false); err != nil {
			halt(device, "failed to update offline status: "+err.Error())
			return
		}
		time.Sleep(800 * time.Millisecond)
	}
}

func sameLEDFrame(left, right [channelCount]color.RGBA) bool {
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func halt(device *Device, message string) {
	if device != nil {
		device.state.channels.clear()
		device.pwm.off()
	}
	pinStatusLED.Low()
	println(message)
	for {
		time.Sleep(time.Second)
	}
}

type pwmOutputs struct{}

func (pwmOutputs) configure() {
	for _, pin := range [...]machine.Pin{pinPWM1, pinPWM2, pinPWM3, pinPWM4, pinPWM5, pinPWM6} {
		pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
		pin.Low()
	}

	py32.RCC.SetAPBENR2_TIM1EN(1)
	py32.RCC.SetAPBENR1_TIM3EN(1)
	py32.RCC.SetAPBRSTR2_TIM1RST(1)
	py32.RCC.SetAPBRSTR2_TIM1RST(0)
	py32.RCC.SetAPBRSTR1_TIM3RST(1)
	py32.RCC.SetAPBRSTR1_TIM3RST(0)

	configureTimer(py32.TIM1)
	configureTimer(py32.TIM3)

	configurePWMPin(pinPWM1, 2)
	configurePWMPin(pinPWM2, 2)
	configurePWMPin(pinPWM3, 2)
	configurePWMPin(pinPWM4, 1)
	configurePWMPin(pinPWM5, 1)
	configurePWMPin(pinPWM6, 13)

	py32.TIM1.SetCCER_CC1E(1)
	py32.TIM1.SetCCER_CC2E(1)
	py32.TIM1.SetCCER_CC3E(1)
	py32.TIM1.SetBDTR_MOE(1)
	py32.TIM3.SetCCER_CC1E(1)
	py32.TIM3.SetCCER_CC2E(1)
	py32.TIM3.SetCCER_CC3E(1)
	py32.TIM1.SetCR1_CEN(1)
	py32.TIM3.SetCR1_CEN(1)
}

func configureTimer(timer *py32.TIM_Type) {
	timer.SetCR1_CEN(0)
	timer.SetPSC(0)
	timer.SetARR(pwmPeriodTicks - 1)
	timer.SetCCR1(0)
	timer.SetCCR2(0)
	timer.SetCCR3(0)
	timer.SetCCMR1_Output_CC1S(0)
	timer.SetCCMR1_Output_OC1M(6)
	timer.SetCCMR1_Output_OC1PE(1)
	timer.SetCCMR1_Output_CC2S(0)
	timer.SetCCMR1_Output_OC2M(6)
	timer.SetCCMR1_Output_OC2PE(1)
	timer.SetCCMR2_Output_CC3S(0)
	timer.SetCCMR2_Output_OC3M(6)
	timer.SetCCMR2_Output_OC3PE(1)
	timer.SetCR1_ARPE(1)
	timer.SetEGR_UG(1)
}

func configurePWMPin(pin machine.Pin, alternateFunction uint8) {
	pin.Configure(machine.PinConfig{Mode: machine.PinAlternate})
	pin.SetAltFunc(alternateFunction)
}

func (pwmOutputs) apply(levels [channelCount]int32) {
	py32.TIM1.SetCCR1(levelCompare(levels[0]))
	py32.TIM1.SetCCR2(levelCompare(levels[1]))
	py32.TIM1.SetCCR3(levelCompare(levels[2]))
	py32.TIM3.SetCCR1(levelCompare(levels[3]))
	py32.TIM3.SetCCR2(levelCompare(levels[4]))
	py32.TIM3.SetCCR3(levelCompare(levels[5]))
	py32.TIM1.SetEGR_UG(1)
	py32.TIM3.SetEGR_UG(1)
}

func (outputs pwmOutputs) off() {
	outputs.apply([channelCount]int32{})
}
