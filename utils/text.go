package utils

import (
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// CenterAlignText takes an array of strings and centers them based on the longest string.
func CenterAlignText(text ...string) string {
	if len(text) == 0 {
		return ""
	}

	// Find the length of the longest string
	maxLength := 0
	for _, line := range text {
		if len(line) > maxLength {
			maxLength = len(line)
		}
	}

	var centeredLines []string
	for _, line := range text {
		// Calculate the number of spaces needed for centering
		totalSpaces := maxLength - len(line)
		leftPadding := totalSpaces / 2
		centeredLine := strings.Repeat(" ", leftPadding) + line + strings.Repeat(" ", totalSpaces-leftPadding)
		centeredLines = append(centeredLines, centeredLine)
	}

	// Join the centered lines with newlines
	return strings.Join(centeredLines, "\n")
}

// Device returns the device name from the DeviceOS.
func Device(os protocol.DeviceOS) string {
	switch os {
	case protocol.DeviceAndroid:
		return "Android"
	case protocol.DeviceIOS:
		return "iOS"
	case protocol.DeviceOSX:
		return "MacOS"
	case protocol.DeviceFireOS:
		return "FireOS"
	case protocol.DeviceGearVR:
		return "Gear VR"
	case protocol.DeviceHololens:
		return "Hololens"
	case protocol.DeviceWin10:
		return "Windows 10"
	case protocol.DeviceWin32:
		return "Win32"
	case protocol.DeviceDedicated:
		return "Dedicated"
	case protocol.DeviceTVOS:
		return "TV"
	case protocol.DeviceOrbis:
		return "PlayStation"
	case protocol.DeviceNX:
		return "Nintendo"
	case protocol.DeviceXBOX:
		return "Xbox"
	case protocol.DeviceWP:
		return "Windows Phone"
	}
	return "Unknown"
}

// InputMode returns the input mode name from the InputMode.
func InputMode(mode int) string {
	switch mode {
	case packet.InputModeMouse:
		return "Keyboard/Mouse"
	case packet.InputModeTouch:
		return "Touch"
	case packet.InputModeGamePad:
		return "Gamepad"
	case packet.InputModeMotionController:
		return "Motion Controller"
	}

	return "Unknown"
}
