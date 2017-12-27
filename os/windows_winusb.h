/*
 * Windows backend for libusb 1.0
 * Copyright © 2009-2012 Pete Batard <pete@akeo.ie>
 * With contributions from Michael Plante, Orin Eman et al.
 * Parts of this code adapted from libusb-win32-v1 by Stephan Meyer
 * Major code testing contribution by Xiaofan Chen
 *
 * This library is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2.1 of the License, or (at your option) any later version.
 *
 * This library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA
 */

// Missing from MSVC6 setupapi.h


// Handle code for HID interface that have been claimed ("dibs")
#define INTERFACE_CLAIMED	((HANDLE)(intptr_t)0xD1B5)
// Additional return code for HID operations that completed synchronously


// http://msdn.microsoft.com/en-us/library/ff545978.aspx
// http://msdn.microsoft.com/en-us/library/ff545972.aspx
// http://msdn.microsoft.com/en-us/library/ff545982.aspx
#if !defined(GUID_DEVINTERFACE_USB_HOST_CONTROLLER)
const GUID GUID_DEVINTERFACE_USB_HOST_CONTROLLER = { 0x3ABF6F2D, 0x71C4, 0x462A, {0x8A, 0x92, 0x1E, 0x68, 0x61, 0xE6, 0xAF, 0x27} };
#endif
#if !defined(GUID_DEVINTERFACE_USB_DEVICE)
const GUID GUID_DEVINTERFACE_USB_DEVICE = { 0xA5DCBF10, 0x6530, 0x11D2, {0x90, 0x1F, 0x00, 0xC0, 0x4F, 0xB9, 0x51, 0xED} };
#endif
#if !defined(GUID_DEVINTERFACE_USB_HUB)
const GUID GUID_DEVINTERFACE_USB_HUB = { 0xF18A0E88, 0xC30C, 0x11D0, {0x88, 0x15, 0x00, 0xA0, 0xC9, 0x06, 0xBE, 0xD8} };
#endif
#if !defined(GUID_DEVINTERFACE_LIBUSB0_FILTER)
const GUID GUID_DEVINTERFACE_LIBUSB0_FILTER = { 0xF9F3FF14, 0xAE21, 0x48A0, {0x8A, 0x25, 0x80, 0x11, 0xA7, 0xA9, 0x31, 0xD9} };
#endif

#define LIBUSB_DT_HID_SIZE		9
#define HID_MAX_CONFIG_DESC_SIZE (LIBUSB_DT_CONFIG_SIZE + LIBUSB_DT_INTERFACE_SIZE \
	+ LIBUSB_DT_HID_SIZE + 2 * LIBUSB_DT_ENDPOINT_SIZE)
#define HID_MAX_REPORT_SIZE		1024
#define HID_IN_EP			0x81
#define HID_OUT_EP			0x02
#define LIBUSB_REQ_RECIPIENT(request_type)	((request_type) & 0x1F)
#define LIBUSB_REQ_TYPE(request_type)		((request_type) & (0x03 << 5))
#define LIBUSB_REQ_IN(request_type)		((request_type) & LIBUSB_ENDPOINT_IN)
#define LIBUSB_REQ_OUT(request_type)		(!LIBUSB_REQ_IN(request_type))

// The following are used for HID reports IOCTLs
#define HID_CTL_CODE(id) \
	CTL_CODE (FILE_DEVICE_KEYBOARD, (id), METHOD_NEITHER, FILE_ANY_ACCESS)
#define HID_BUFFER_CTL_CODE(id) \
	CTL_CODE (FILE_DEVICE_KEYBOARD, (id), METHOD_BUFFERED, FILE_ANY_ACCESS)
#define HID_IN_CTL_CODE(id) \
	CTL_CODE (FILE_DEVICE_KEYBOARD, (id), METHOD_IN_DIRECT, FILE_ANY_ACCESS)
#define HID_OUT_CTL_CODE(id) \
	CTL_CODE (FILE_DEVICE_KEYBOARD, (id), METHOD_OUT_DIRECT, FILE_ANY_ACCESS)

#define IOCTL_HID_GET_FEATURE		HID_OUT_CTL_CODE(100)
#define IOCTL_HID_GET_INPUT_REPORT	HID_OUT_CTL_CODE(104)
#define IOCTL_HID_SET_FEATURE		HID_IN_CTL_CODE(100)
#define IOCTL_HID_SET_OUTPUT_REPORT	HID_IN_CTL_CODE(101)


/*
 * Windows DDK API definitions. Most of it copied from MinGW's includes
 */
typedef DWORD DEVNODE, DEVINST;
typedef DEVNODE *PDEVNODE, *PDEVINST;
typedef DWORD RETURN_TYPE;
typedef RETURN_TYPE CONFIGRET;

#if !defined(USB_GET_NODE_CONNECTION_INFORMATION_EX)
#define USB_GET_NODE_CONNECTION_INFORMATION_EX	274
#endif
#if !defined(USB_GET_HUB_CAPABILITIES_EX)
#define USB_GET_HUB_CAPABILITIES_EX		276
#endif
#if !defined(USB_GET_NODE_CONNECTION_INFORMATION_EX_V2)
#define USB_GET_NODE_CONNECTION_INFORMATION_EX_V2	279
#endif

#ifndef METHOD_BUFFERED
#define METHOD_BUFFERED				0
#endif
#ifndef FILE_ANY_ACCESS
#define FILE_ANY_ACCESS				0x00000000
#endif
#ifndef FILE_DEVICE_UNKNOWN
#define FILE_DEVICE_UNKNOWN			0x00000022
#endif
#ifndef FILE_DEVICE_USB
#define FILE_DEVICE_USB				FILE_DEVICE_UNKNOWN
#endif

#ifndef CTL_CODE
#define CTL_CODE(DeviceType, Function, Method, Access) \
	(((DeviceType) << 16) | ((Access) << 14) | ((Function) << 2) | (Method))
#endif

#define IOCTL_USB_GET_HUB_CAPABILITIES_EX \
	CTL_CODE( FILE_DEVICE_USB, USB_GET_HUB_CAPABILITIES_EX, METHOD_BUFFERED, FILE_ANY_ACCESS)

#define IOCTL_USB_GET_HUB_CAPABILITIES \
	CTL_CODE(FILE_DEVICE_USB, USB_GET_HUB_CAPABILITIES, METHOD_BUFFERED, FILE_ANY_ACCESS)

#define IOCTL_USB_GET_DESCRIPTOR_FROM_NODE_CONNECTION \
	CTL_CODE(FILE_DEVICE_USB, USB_GET_DESCRIPTOR_FROM_NODE_CONNECTION, METHOD_BUFFERED, FILE_ANY_ACCESS)

#define IOCTL_USB_GET_ROOT_HUB_NAME \
	CTL_CODE(FILE_DEVICE_USB, HCD_GET_ROOT_HUB_NAME, METHOD_BUFFERED, FILE_ANY_ACCESS)

#define IOCTL_USB_GET_NODE_INFORMATION \
	CTL_CODE(FILE_DEVICE_USB, USB_GET_NODE_INFORMATION, METHOD_BUFFERED, FILE_ANY_ACCESS)

#define IOCTL_USB_GET_NODE_CONNECTION_INFORMATION_EX \
	CTL_CODE(FILE_DEVICE_USB, USB_GET_NODE_CONNECTION_INFORMATION_EX, METHOD_BUFFERED, FILE_ANY_ACCESS)

#define IOCTL_USB_GET_NODE_CONNECTION_INFORMATION_EX_V2 \
	CTL_CODE(FILE_DEVICE_USB, USB_GET_NODE_CONNECTION_INFORMATION_EX_V2, METHOD_BUFFERED, FILE_ANY_ACCESS)

#define IOCTL_USB_GET_NODE_CONNECTION_ATTRIBUTES \
	CTL_CODE(FILE_DEVICE_USB, USB_GET_NODE_CONNECTION_ATTRIBUTES, METHOD_BUFFERED, FILE_ANY_ACCESS)

#define IOCTL_USB_GET_NODE_CONNECTION_NAME \
	CTL_CODE(FILE_DEVICE_USB, USB_GET_NODE_CONNECTION_NAME, METHOD_BUFFERED, FILE_ANY_ACCESS)


static  struct windows_device_priv *_device_priv(struct libusb_device *dev)
{
	return (struct windows_device_priv *)dev->os_priv;
}

static  struct windows_device_priv *windows_device_priv_init(struct libusb_device *dev)
{
	struct windows_device_priv *p = _device_priv(dev);
	int i;

	p->depth = 0;
	p->port = 0;
	p->path = NULL;
	p->apib = &usb_api_backend[USB_API_UNSUPPORTED];
	p->sub_api = SUB_API_NOTSET;
	p->hid = NULL;
	p->active_config = 0;
	p->config_descriptor = NULL;
	memset(&p->dev_descriptor, 0, sizeof(USB_DEVICE_DESCRIPTOR));
	for (i = 0; i < USB_MAXINTERFACES; i++) {
		p->usb_interface[i].path = NULL;
		p->usb_interface[i].apib = &usb_api_backend[USB_API_UNSUPPORTED];
		p->usb_interface[i].sub_api = SUB_API_NOTSET;
		p->usb_interface[i].nb_endpoints = 0;
		p->usb_interface[i].endpoint = NULL;
		p->usb_interface[i].restricted_functionality = false;
	}

	return p;
}

static  struct windows_device_handle_priv *_device_handle_priv(
	struct libusb_device_handle *handle)
{
	return (struct windows_device_handle_priv *)handle->os_priv;
}

typedef union _USB_PROTOCOLS {
	uint64 ul;
	struct {
		uint64 Usb110:1;
		uint64 Usb200:1;
		uint64 Usb300:1;
		uint64 ReservedMBZ:29;
	};
} USB_PROTOCOLS, *PUSB_PROTOCOLS;

typedef union _USB_NODE_CONNECTION_INFORMATION_EX_V2_FLAGS {
	uint64 ul;
	struct {
		uint64 DeviceIsOperatingAtSuperSpeedOrHigher:1;
		uint64 DeviceIsSuperSpeedCapableOrHigher:1;
		uint64 ReservedMBZ:30;
	};
} USB_NODE_CONNECTION_INFORMATION_EX_V2_FLAGS, *PUSB_NODE_CONNECTION_INFORMATION_EX_V2_FLAGS;

typedef struct _HIDP_VALUE_CAPS {
	USAGE UsagePage;
	uint8 ReportID;
	bool IsAlias;
	uint16 BitField;
	uint16 LinkCollection;
	USAGE LinkUsage;
	USAGE LinkUsagePage;
	bool IsRange;
	bool IsStringRange;
	bool IsDesignatorRange;
	bool IsAbsolute;
	bool HasNull;
	uint8 Reserved;
	uint16 BitSize;
	uint16 ReportCount;
	uint16 Reserved2[5];
	uint64 UnitsExp;
	uint64 Units;
	LONG LogicalMin, LogicalMax;
	LONG PhysicalMin, PhysicalMax;
	union {
		struct {
			USAGE UsageMin, UsageMax;
			uint16 StringMin, StringMax;
			uint16 DesignatorMin, DesignatorMax;
			uint16 DataIndexMin, DataIndexMax;
		} Range;
		struct {
			USAGE Usage, Reserved1;
			uint16 StringIndex, Reserved2;
			uint16 DesignatorIndex, Reserved3;
			uint16 DataIndex, Reserved4;
		} NotRange;
	} u;
} HIDP_VALUE_CAPS, *PHIDP_VALUE_CAPS;

typedef struct USB_HUB_NAME_FIXED {
	union {
		USB_ROOT_HUB_NAME_FIXED root;
		USB_NODE_CONNECTION_NAME_FIXED node;
	} u;
} USB_HUB_NAME_FIXED;