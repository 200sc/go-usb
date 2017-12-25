package os

type windows_version int
const (
	WINDOWS_CE windows_version = -2,
	WINDOWS_UNDEFINED windows_version = -1,
	WINDOWS_UNSUPPORTED windows_version = 0,
	WINDOWS_XP windows_version = 0x51,
	WINDOWS_2003 windows_version = 0x52,	// Also XP x64
	WINDOWS_VISTA windows_version = 0x60,
	WINDOWS_7 windows_version = 0x61,
	WINDOWS_8 windows_version = 0x62,
	WINDOWS_8_1_OR_LATER windows_version = 0x63,
	WINDOWS_MAX windows_version = 0x64
)

// access modes
type rw_type uint8 
const (
	RW_NONE rw_type = iota
	RW_READ rw_type
	RW_WRITE rw_type
)