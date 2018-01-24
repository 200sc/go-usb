// I'm actually unfamiliar with this ':#' syntax
typedef USB_PROTOCOLS union {
	uint64 ul
	struct {
		uint64 Usb110:1
		uint64 Usb200:1
		uint64 Usb300:1
		uint64 ReservedMBZ:29
	}
}

typedef USB_NODE_CONNECTION_INFORMATION_EX_V2_FLAGS union {
	uint64 ul
	struct {
		uint64 DeviceIsOperatingAtSuperSpeedOrHigher:1
		uint64 DeviceIsSuperSpeedCapableOrHigher:1
		uint64 ReservedMBZ:30
	}
}