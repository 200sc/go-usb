package os

type usbfs_urb_type uint8

const (
	USBFS_URB_TYPE_ISO usbfs_urb_type = iota
	USBFS_URB_TYPE_INTERRUPT usbfs_urb_type
	USBFS_URB_TYPE_CONTROL usbfs_urb_type
	USBFS_URB_TYPE_BULK usbfs_urb_type
)

type reap_action uint8
const (
	NORMAL reap_action = iota
	/* submission failed after the first URB, so await cancellation/completion
	 * of all the others */
	SUBMIT_FAILED reap_action

	/* cancelled by user or timeout */
	CANCELLED reap_action

	/* completed multi-URB transfer in non-final URB */
	COMPLETED_EARLY reap_action

	/* one or more urbs encountered a low-level error */
	ERROR reap_action
)