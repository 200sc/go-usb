package usb

/* standard USB stuff */

/** \ingroup libusb_desc
 * Device and/or Interface Class codes */
type libusb_class_code uint8
const(
	/** In the context of a \ref libusb_device_descriptor "device descriptor",
	 * this bDeviceClass value indicates that each interface specifies its
	 * own class information and all interfaces operate independently.
	 */
	LIBUSB_CLASS_PER_INTERFACE libusb_class_code = 0

	/** Audio class */
	LIBUSB_CLASS_AUDIO libusb_class_code = 1

	/** Communications class */
	LIBUSB_CLASS_COMM libusb_class_code = 2

	/** Human Interface Device class */
	LIBUSB_CLASS_HID libusb_class_code = 3

	/** Physical */
	LIBUSB_CLASS_PHYSICAL libusb_class_code = 5

	/** Printer class */
	LIBUSB_CLASS_PRINTER libusb_class_code = 7

	/** Image class */
	LIBUSB_CLASS_IMAGE libusb_class_code = 6

	/** Mass storage class */
	LIBUSB_CLASS_MASS_STORAGE libusb_class_code = 8

	/** Hub class */
	LIBUSB_CLASS_HUB libusb_class_code = 9

	/** Data class */
	LIBUSB_CLASS_DATA libusb_class_code = 10

	/** Smart Card */
	LIBUSB_CLASS_SMART_CARD libusb_class_code = 0x0b

	/** Content Security */
	LIBUSB_CLASS_CONTENT_SECURITY libusb_class_code = 0x0d

	/** Video */
	LIBUSB_CLASS_VIDEO libusb_class_code = 0x0e

	/** Personal Healthcare */
	LIBUSB_CLASS_PERSONAL_HEALTHCARE libusb_class_code = 0x0f

	/** Diagnostic Device */
	LIBUSB_CLASS_DIAGNOSTIC_DEVICE libusb_class_code = 0xdc

	/** Wireless class */
	LIBUSB_CLASS_WIRELESS libusb_class_code = 0xe0

	/** Application class */
	LIBUSB_CLASS_APPLICATION libusb_class_code = 0xfe

	/** Class is vendor-specific */
	LIBUSB_CLASS_VENDOR_SPEC libusb_class_code = 0xff
};

/** \ingroup libusb_desc
 * Descriptor types as defined by the USB specification. */
type libusb_descriptor_type uint8
const(
	/** Device descriptor. See libusb_device_descriptor. */
	LIBUSB_DT_DEVICE libusb_descriptor_type = 0x01

	/** Configuration descriptor. See libusb_config_descriptor. */
	LIBUSB_DT_CONFIG libusb_descriptor_type = 0x02

	/** String descriptor */
	LIBUSB_DT_STRING libusb_descriptor_type = 0x03

	/** Interface descriptor. See libusb_interface_descriptor. */
	LIBUSB_DT_INTERFACE libusb_descriptor_type = 0x04

	/** Endpoint descriptor. See libusb_endpoint_descriptor. */
	LIBUSB_DT_ENDPOINT libusb_descriptor_type = 0x05

	/** BOS descriptor */
	LIBUSB_DT_BOS libusb_descriptor_type = 0x0f

	/** Device Capability descriptor */
	LIBUSB_DT_DEVICE_CAPABILITY libusb_descriptor_type = 0x10

	/** HID descriptor */
	LIBUSB_DT_HID libusb_descriptor_type = 0x21

	/** HID report descriptor */
	LIBUSB_DT_REPORT libusb_descriptor_type = 0x22

	/** Physical descriptor */
	LIBUSB_DT_PHYSICAL libusb_descriptor_type = 0x23

	/** Hub descriptor */
	LIBUSB_DT_HUB libusb_descriptor_type = 0x29

	/** SuperSpeed Hub descriptor */
	LIBUSB_DT_SUPERSPEED_HUB libusb_descriptor_type = 0x2a

	/** SuperSpeed Endpoint Companion descriptor */
	LIBUSB_DT_SS_ENDPOINT_COMPANION libusb_descriptor_type = 0x30
};

/** \ingroup libusb_desc
 * Endpoint direction. Values for bit 7 of the
 * \ref libusb_endpoint_descriptor::bEndpointAddress "endpoint address" scheme.
 */
type libusb_endpoint_direction uint8
const(
	/** In: device-to-host */
	LIBUSB_ENDPOINT_IN libusb_endpoint_direction = 0x80

	/** Out: host-to-device */
	LIBUSB_ENDPOINT_OUT libusb_endpoint_direction = 0x00
)

/** \ingroup libusb_desc
 * Endpoint transfer type. Values for bits 0:1 of the
 * \ref libusb_endpoint_descriptor::bmAttributes "endpoint attributes" field.
 */
type libusb_transfer_type uint8
const(
	/** Control endpoint */
	LIBUSB_TRANSFER_TYPE_CONTROL libusb_transfer_type = iota

	/** Isochronous endpoint */
	LIBUSB_TRANSFER_TYPE_ISOCHRONOUS libusb_transfer_type

	/** Bulk endpoint */
	LIBUSB_TRANSFER_TYPE_BULK libusb_transfer_type

	/** Interrupt endpoint */
	LIBUSB_TRANSFER_TYPE_INTERRUPT libusb_transfer_type

	/** Stream endpoint */
	LIBUSB_TRANSFER_TYPE_BULK_STREAM libusb_transfer_type
)

/** \ingroup libusb_misc
 * Standard requests, as defined in table 9-5 of the USB 3.0 specifications */
type libusb_standard_request uint8
const(
	/** Request status of the specific recipient */
	LIBUSB_REQUEST_GET_STATUS libusb_standard_request = 0x00

	/** Clear or disable a specific feature */
	LIBUSB_REQUEST_CLEAR_FEATURE libusb_standard_request = 0x01

	/* 0x02 is reserved */

	/** Set or enable a specific feature */
	LIBUSB_REQUEST_SET_FEATURE libusb_standard_request = 0x03

	/* 0x04 is reserved */

	/** Set device address for all future accesses */
	LIBUSB_REQUEST_SET_ADDRESS libusb_standard_request = 0x05

	/** Get the specified descriptor */
	LIBUSB_REQUEST_GET_DESCRIPTOR libusb_standard_request = 0x06

	/** Used to update existing descriptors or add new descriptors */
	LIBUSB_REQUEST_SET_DESCRIPTOR libusb_standard_request = 0x07

	/** Get the current device configuration value */
	LIBUSB_REQUEST_GET_CONFIGURATION libusb_standard_request = 0x08

	/** Set device configuration */
	LIBUSB_REQUEST_SET_CONFIGURATION libusb_standard_request = 0x09

	/** Return the selected alternate setting for the specified interface */
	LIBUSB_REQUEST_GET_INTERFACE libusb_standard_request = 0x0A

	/** Select an alternate interface for the specified interface */
	LIBUSB_REQUEST_SET_INTERFACE libusb_standard_request = 0x0B

	/** Set then report an endpoint's synchronization frame */
	LIBUSB_REQUEST_SYNCH_FRAME libusb_standard_request = 0x0C

	/** Sets both the U1 and U2 Exit Latency */
	LIBUSB_REQUEST_SET_SEL libusb_standard_request = 0x30

	/** Delay from the time a host transmits a packet to the time it is
	  * received by the device. */
	LIBUSB_SET_ISOCH_DELAY libusb_standard_request = 0x31
)

/** \ingroup libusb_misc
 * Request type bits of the
 * \ref libusb_control_setup::bmRequestType "bmRequestType" field in control
 * transfers. */
type libusb_request_type int
const (
	/** Standard */
	LIBUSB_REQUEST_TYPE_STANDARD libusb_request_type = (0x00 << 5)

	/** Class */
	LIBUSB_REQUEST_TYPE_CLASS libusb_request_type = (0x01 << 5)

	/** Vendor */
	LIBUSB_REQUEST_TYPE_VENDOR libusb_request_type = (0x02 << 5)

	/** Reserved */
	LIBUSB_REQUEST_TYPE_RESERVED libusb_request_type = (0x03 << 5)
)

/** \ingroup libusb_misc
 * Recipient bits of the
 * \ref libusb_control_setup::bmRequestType "bmRequestType" field in control
 * transfers. Values 4 through 31 are reserved. */
type libusb_request_recipient uint8
const(
	/** Device */
	LIBUSB_RECIPIENT_DEVICE libusb_request_recipient = iota

	/** Interface */
	LIBUSB_RECIPIENT_INTERFACE libusb_request_recipient

	/** Endpoint */
	LIBUSB_RECIPIENT_ENDPOINT libusb_request_recipient 

	/** Other */
	LIBUSB_RECIPIENT_OTHER libusb_request_recipient
)

/** \ingroup libusb_desc
 * Synchronization type for isochronous endpoints. Values for bits 2:3 of the
 * \ref libusb_endpoint_descriptor::bmAttributes "bmAttributes" field in
 * libusb_endpoint_descriptor.
 */
type libusb_iso_sync_type uint8
const (
	/** No synchronization */
	LIBUSB_ISO_SYNC_TYPE_NONE libusb_iso_sync_type = iota

	/** Asynchronous */
	LIBUSB_ISO_SYNC_TYPE_ASYNC libusb_iso_sync_type

	/** Adaptive */
	LIBUSB_ISO_SYNC_TYPE_ADAPTIVE libusb_iso_sync_type

	/** Synchronous */
	LIBUSB_ISO_SYNC_TYPE_SYNC libusb_iso_sync_type
)

/** \ingroup libusb_desc
 * Usage type for isochronous endpoints. Values for bits 4:5 of the
 * \ref libusb_endpoint_descriptor::bmAttributes "bmAttributes" field in
 * libusb_endpoint_descriptor.
 */
type libusb_iso_usage_type uint8
const(
	/** Data endpoint */
	LIBUSB_ISO_USAGE_TYPE_DATA libusb_iso_usage_type = iota

	/** Feedback endpoint */
	LIBUSB_ISO_USAGE_TYPE_FEEDBACK libusb_iso_usage_type 

	/** Implicit feedback Data endpoint */
	LIBUSB_ISO_USAGE_TYPE_IMPLICIT libusb_iso_usage_type
)

/** \ingroup libusb_dev
 * Speed codes. Indicates the speed at which the device is operating.
 */
type libusb_speed uint8
const(
	/** The OS doesn't report or know the device speed. */
	LIBUSB_SPEED_UNKNOWN libusb_speed = iota

	/** The device is operating at low speed (1.5MBit/s). */
	LIBUSB_SPEED_LOW libusb_speed 

	/** The device is operating at full speed (12MBit/s). */
	LIBUSB_SPEED_FULL libusb_speed 

	/** The device is operating at high speed (480MBit/s). */
	LIBUSB_SPEED_HIGH libusb_speed 

	/** The device is operating at super speed (5000MBit/s). */
	LIBUSB_SPEED_SUPER libusb_speed 
)

/** \ingroup libusb_dev
 * Supported speeds (wSpeedSupported) bitfield. Indicates what
 * speeds the device supports.
 */
type libusb_supported_speed uint8
const(
	/** Low speed operation supported (1.5MBit/s). */
	LIBUSB_LOW_SPEED_OPERATION   libusb_supported_speed = 1

	/** Full speed operation supported (12MBit/s). */
	LIBUSB_FULL_SPEED_OPERATION  libusb_supported_speed = 2

	/** High speed operation supported (480MBit/s). */
	LIBUSB_HIGH_SPEED_OPERATION  libusb_supported_speed = 4

	/** Superspeed operation supported (5000MBit/s). */
	LIBUSB_SUPER_SPEED_OPERATION libusb_supported_speed = 8
)

/** \ingroup libusb_dev
 * Masks for the bits of the
 * \ref libusb_usb_2_0_extension_descriptor::bmAttributes "bmAttributes" field
 * of the USB 2.0 Extension descriptor.
 */
type libusb_usb_2_0_extension_attributes uint8
/** Supports Link Power Management (LPM) */
const LIBUSB_BM_LPM_SUPPORT libusb_usb_2_0_extension_attributes = 2

/** \ingroup libusb_dev
 * Masks for the bits of the
 * \ref libusb_ss_usb_device_capability_descriptor::bmAttributes "bmAttributes" field
 * field of the SuperSpeed USB Device Capability descriptor.
 */
type libusb_ss_usb_device_capability_attributes uint8
/** Supports Latency Tolerance Messages (LTM) */
const LIBUSB_BM_LTM_SUPPORT libusb_ss_usb_device_capability_attributes = 2

/** \ingroup libusb_dev
 * USB capability types
 */
type libusb_bos_type uint8
const(
	/** Wireless USB device capability */
	LIBUSB_BT_WIRELESS_USB_DEVICE_CAPABILITY libusb_bos_type = 1

	/** USB 2.0 extensions */
	LIBUSB_BT_USB_2_0_EXTENSION	libusb_bos_type = 2

	/** SuperSpeed USB device capability */
	LIBUSB_BT_SS_USB_DEVICE_CAPABILITY libusb_bos_type = 3

	/** Container ID type */
	LIBUSB_BT_CONTAINER_ID libusb_bos_type = 4
)

/** \ingroup libusb_misc
 * Error codes. Most libusb functions return 0 on success or one of these
 * codes on failure.
 * You can call libusb_error_name() to retrieve a string representation of an
 * error code or libusb_strerror() to get an end-user suitable description of
 * an error code.
 */
type libusb_error int8
const(
	/** Success (no error) */
	LIBUSB_SUCCESS libusb_error = 0

	/** Input/output error */
	LIBUSB_ERROR_IO libusb_error = -1

	/** Invalid parameter */
	LIBUSB_ERROR_INVALID_PARAM libusb_error = -2

	/** Access denied (insufficient permissions) */
	LIBUSB_ERROR_ACCESS libusb_error = -3

	/** No such device (it may have been disconnected) */
	LIBUSB_ERROR_NO_DEVICE libusb_error = -4

	/** Entity not found */
	LIBUSB_ERROR_NOT_FOUND libusb_error = -5

	/** Resource busy */
	LIBUSB_ERROR_BUSY libusb_error = -6

	/** Operation timed out */
	LIBUSB_ERROR_TIMEOUT libusb_error = -7

	/** Overflow */
	LIBUSB_ERROR_OVERFLOW libusb_error = -8

	/** Pipe error */
	LIBUSB_ERROR_PIPE libusb_error = -9

	/** System call interrupted (perhaps due to signal) */
	LIBUSB_ERROR_INTERRUPTED libusb_error = -10

	/** Insufficient memory */
	LIBUSB_ERROR_NO_MEM libusb_error = -11

	/** Operation not supported or unimplemented on this platform */
	LIBUSB_ERROR_NOT_SUPPORTED libusb_error = -12

	/* NB: Remember to update LIBUSB_ERROR_COUNT below as well as the
	   message strings in strerror.c when adding new error codes here. */

	/** Other error */
	LIBUSB_ERROR_OTHER libusb_error = -99
)

/** \ingroup libusb_asyncio
 * Transfer status codes */
type libusb_transfer_status uint8 
const(
	/** Transfer completed without error. Note that this does not indicate
	 * that the entire amount of requested data was transferred. */
	LIBUSB_TRANSFER_COMPLETED libusb_transfer_status = iota

	/** Transfer failed */
	LIBUSB_TRANSFER_ERROR libusb_transfer_status

	/** Transfer timed out */
	LIBUSB_TRANSFER_TIMED_OUT libusb_transfer_status

	/** Transfer was cancelled */
	LIBUSB_TRANSFER_CANCELLED libusb_transfer_status

	/** For bulk/interrupt endpoints: halt condition detected (endpoint
	 * stalled). For control endpoints: control request not supported. */
	LIBUSB_TRANSFER_STALL libusb_transfer_status

	/** Device was disconnected */
	LIBUSB_TRANSFER_NO_DEVICE libusb_transfer_status

	/** Device sent more data than requested */
	LIBUSB_TRANSFER_OVERFLOW libusb_transfer_status

	/* NB! Remember to update libusb_error_name()
	   when adding new status codes here. */
)

/** \ingroup libusb_asyncio
 * libusb_transfer.flags values */
type libusb_transfer_flags uint8
const(
	/** Report short frames as errors */
	LIBUSB_TRANSFER_SHORT_NOT_OK libusb_transfer_flags = 1<<0

	/** Automatically free() transfer buffer during libusb_free_transfer().
	 * Note that buffers allocated with libusb_dev_mem_alloc() should not
	 * be attempted freed in this way, since free() is not an appropriate
	 * way to release such memory. */
	LIBUSB_TRANSFER_FREE_BUFFER libusb_transfer_flags = 1<<1

	/** Automatically call libusb_free_transfer() after callback returns.
	 * If this flag is set, it is illegal to call libusb_free_transfer()
	 * from your transfer callback, as this will result in a double-free
	 * when this flag is acted upon. */
	LIBUSB_TRANSFER_FREE_TRANSFER libusb_transfer_flags = 1<<2

	/** Terminate transfers that are a multiple of the endpoint's
	 * wMaxPacketSize with an extra zero length packet. This is useful
	 * when a device protocol mandates that each logical request is
	 * terminated by an incomplete packet (i.e. the logical requests are
	 * not separated by other means).
	 *
	 * This flag only affects host-to-device transfers to bulk and interrupt
	 * endpoints. In other situations, it is ignored.
	 *
	 * This flag only affects transfers with a length that is a multiple of
	 * the endpoint's wMaxPacketSize. On transfers of other lengths, this
	 * flag has no effect. Therefore, if you are working with a device that
	 * needs a ZLP whenever the end of the logical request falls on a packet
	 * boundary, then it is sensible to set this flag on <em>every</em>
	 * transfer (you do not have to worry about only setting it on transfers
	 * that end on the boundary).
	 *
	 * This flag is currently only supported on Linux.
	 * On other systems, libusb_submit_transfer() will return
	 * LIBUSB_ERROR_NOT_SUPPORTED for every transfer where this flag is set.
	 *
	 * Available since libusb-1.0.9.
	 */
	LIBUSB_TRANSFER_ADD_ZERO_PACKET libusb_transfer_flags = 1 << 3
)

/** \ingroup libusb_misc
 * Capabilities supported by an instance of libusb on the current running
 * platform. Test if the loaded library supports a given capability by calling
 * \ref libusb_has_capability().
 */
type libusb_capability int
const(
	/** The libusb_has_capability() API is available. */
	LIBUSB_CAP_HAS_CAPABILITY libusb_capability = 0x0000
	/** Hotplug support is available on this platform. */
	LIBUSB_CAP_HAS_HOTPLUG libusb_capability = 0x0001
	/** The library can access HID devices without requiring user intervention.
	 * Note that before being able to actually access an HID device, you may
	 * still have to call additional libusb functions such as
	 * \ref libusb_detach_kernel_driver(). */
	LIBUSB_CAP_HAS_HID_ACCESS libusb_capability = 0x0100
	/** The library supports detaching of the default USB driver, using 
	 * \ref libusb_detach_kernel_driver(), if one is set by the OS kernel */
	LIBUSB_CAP_SUPPORTS_DETACH_KERNEL_DRIVER libusb_capability = 0x0101
)

/** \ingroup libusb_lib
 *  Log message levels.
 *  - LIBUSB_LOG_LEVEL_NONE (0)    : no messages ever printed by the library (default)
 *  - LIBUSB_LOG_LEVEL_ERROR (1)   : error messages are printed to stderr
 *  - LIBUSB_LOG_LEVEL_WARNING (2) : warning and error messages are printed to stderr
 *  - LIBUSB_LOG_LEVEL_INFO (3)    : informational messages are printed to stdout, warning
 *    and error messages are printed to stderr
 *  - LIBUSB_LOG_LEVEL_DEBUG (4)   : debug and informational messages are printed to stdout,
 *    warnings and errors to stderr
 */
type libusb_log_level uint8
const(
	LIBUSB_LOG_LEVEL_NONE libusb_log_level = iota
	LIBUSB_LOG_LEVEL_ERROR libusb_log_level
	LIBUSB_LOG_LEVEL_WARNING libusb_log_level
	LIBUSB_LOG_LEVEL_INFO libusb_log_level
	LIBUSB_LOG_LEVEL_DEBUG libusb_log_level
)

/** \ingroup libusb_hotplug
 *
 * Since version 1.0.16, \ref LIBUSB_API_VERSION >= 0x01000102
 *
 * Flags for hotplug events */
type libusb_hotplug_flag uint8
const(/** Default value when not using any flags. */
	LIBUSB_HOTPLUG_NO_FLAGS libusb_hotplug_flag = 0

	/** Arm the callback and fire it for all matching currently attached devices. */
	LIBUSB_HOTPLUG_ENUMERATE libusb_hotplug_flag = 1<<0
)

/** \ingroup libusb_hotplug
 *
 * Since version 1.0.16, \ref LIBUSB_API_VERSION >= 0x01000102
 *
 * Hotplug events */
type libusb_hotplug_event uint8
const(
	/** A device has been plugged in and is ready to use */
	LIBUSB_HOTPLUG_EVENT_DEVICE_ARRIVED libusb_hotplug_event = 0x01

	/** A device has left and is no longer available.
	 * It is the user's responsibility to call libusb_close on any handle associated with a disconnected device.
	 * It is safe to call libusb_get_device_descriptor on a device that has left */
	LIBUSB_HOTPLUG_EVENT_DEVICE_LEFT    libusb_hotplug_event = 0x02
)

const (
	LIBUSB_TRANSFER_TYPE_MASK = 0x03    /* in bmAttributes */
	LIBUSB_ISO_USAGE_TYPE_MASK = 0x30
	LIBUSB_ISO_SYNC_TYPE_MASK = 0x0C
)