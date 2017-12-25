package libusb

type usbi_event_flags uint8
const(
	/* The list of pollfds has been modified */
	USBI_EVENT_POLLFDS_MODIFIED usbi_event_flags = 1 << 0,

	/* The user has interrupted the event handler */
	USBI_EVENT_USER_INTERRUPT usbi_event_flags = 1 << 1,
)

type usbi_clock uint8
const(
	USBI_CLOCK_MONOTONIC usbi_clock = iota
	USBI_CLOCK_REALTIME usbi_clock
)

type usbi_transfer_state_flags uint8
const(
	/* Transfer successfully submitted by backend */
	USBI_TRANSFER_IN_FLIGHT usbi_transfer_state_flags = 1 << 0,

	/* Cancellation was requested via libusb_cancel_transfer() */
	USBI_TRANSFER_CANCELLING usbi_transfer_state_flags = 1 << 1,

	/* Operation on the transfer failed because the device disappeared */
	USBI_TRANSFER_DEVICE_DISAPPEARED usbi_transfer_state_flags = 1 << 2,
)

type usbi_transfer_timeout_flags uint8
const(
	/* Set by backend submit_transfer() if the OS handles timeout */
	USBI_TRANSFER_OS_HANDLES_TIMEOUT usbi_transfer_timeout_flags = 1 << 0,

	/* The transfer timeout has been handled */
	USBI_TRANSFER_TIMEOUT_HANDLED usbi_transfer_timeout_flags = 1 << 1,

	/* The transfer timeout was successfully processed */
	USBI_TRANSFER_TIMED_OUT usbi_transfer_timeout_flags = 1 << 2,
)