package midi

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// MIDI 2.0 driver service registry paths (Windows 11 24H2+).
// Disabling these does NOT affect the classic Windows MIDI 1.0 service (midisrv / Windows MIDI).
var midi2Services = []string{
	`SYSTEM\CurrentControlSet\Services\Midi2.MidiSrv`,
	`SYSTEM\CurrentControlSet\Services\Midi2.BS2UMP`,
	`SYSTEM\CurrentControlSet\Services\Midi2.UMP2BS`,
	`SYSTEM\CurrentControlSet\Services\Midi2.KSTransport`,
	`SYSTEM\CurrentControlSet\Services\Midi2.VirtualMidi`,
	`SYSTEM\CurrentControlSet\Services\Midi2.DiagTransport`,
	`SYSTEM\CurrentControlSet\Services\Midi2.KSAbstraction`,
	`SYSTEM\CurrentControlSet\Services\Midi2.NetworkMidi`,
}

const (
	// startDisabled marks a driver as disabled (won't load).
	startDisabled uint32 = 4
	// startDemand marks a driver as manual/demand-start (default for most of these).
	startDemand uint32 = 3
)

// IsMIDI2Disabled checks whether MIDI 2.0 services are currently disabled.
// Returns true if the main Midi2.MidiSrv service has Start == 4 (disabled).
func IsMIDI2Disabled() (bool, error) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		midi2Services[0],
		registry.READ,
	)
	if err != nil {
		// Key doesn't exist = MIDI 2.0 not installed on this system
		return false, nil
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue("Start")
	if err != nil {
		return false, fmt.Errorf("failed to read Start value: %w", err)
	}

	return val == uint64(startDisabled), nil
}

// DisableMIDI2 sets Start=4 (disabled) on all MIDI 2.0 service entries.
// The classic MIDI 1.0 service (midisrv / Windows.MIDI.Services) is NOT touched.
func DisableMIDI2() (int, error) {
	count := 0
	for _, path := range midi2Services {
		err := setServiceStart(path, startDisabled)
		if err != nil {
			// Skip services that don't exist on this build
			continue
		}
		count++
	}
	if count == 0 {
		return 0, fmt.Errorf("no MIDI 2.0 services found on this system")
	}
	return count, nil
}

// EnableMIDI2 restores Start=3 (demand/manual) on all MIDI 2.0 service entries.
func EnableMIDI2() (int, error) {
	count := 0
	for _, path := range midi2Services {
		err := setServiceStart(path, startDemand)
		if err != nil {
			continue
		}
		count++
	}
	if count == 0 {
		return 0, fmt.Errorf("no MIDI 2.0 services found on this system")
	}
	return count, nil
}

func setServiceStart(keyPath string, startValue uint32) error {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		keyPath,
		registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer k.Close()

	return k.SetDWordValue("Start", startValue)
}
