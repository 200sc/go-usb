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
#if !defined(SPDRP_ADDRESS)
#define SPDRP_ADDRESS		28
#endif
#if !defined(SPDRP_INSTALL_STATE)
#define SPDRP_INSTALL_STATE	34
#endif

// Handle code for HID interface that have been claimed ("dibs")
#define INTERFACE_CLAIMED	((HANDLE)(intptr_t)0xD1B5)
// Additional return code for HID operations that completed synchronously
#define LIBUSB_COMPLETED	(LIBUSB_SUCCESS + 1)

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

#define WINUSBX_DRV_NAMES	{"libusbK", "libusb0", "WinUSB"}

struct windows_usb_api_backend {
	const uint8_t id;
	const char *designation;
	const char **driver_name_list; // Driver name, without .sys, e.g. "usbccgp"
	const uint8_t nb_driver_names;
	int (*init)(int sub_api, struct libusb_context *ctx);
	int (*exit)(int sub_api);
	int (*open)(int sub_api, struct libusb_device_handle *dev_handle);
	void (*close)(int sub_api, struct libusb_device_handle *dev_handle);
	int (*configure_endpoints)(int sub_api, struct libusb_device_handle *dev_handle, int iface);
	int (*claim_interface)(int sub_api, struct libusb_device_handle *dev_handle, int iface);
	int (*set_interface_altsetting)(int sub_api, struct libusb_device_handle *dev_handle, int iface, int altsetting);
	int (*release_interface)(int sub_api, struct libusb_device_handle *dev_handle, int iface);
	int (*clear_halt)(int sub_api, struct libusb_device_handle *dev_handle, unsigned char endpoint);
	int (*reset_device)(int sub_api, struct libusb_device_handle *dev_handle);
	int (*submit_bulk_transfer)(int sub_api, struct usbi_transfer *itransfer);
	int (*submit_iso_transfer)(int sub_api, struct usbi_transfer *itransfer);
	int (*submit_control_transfer)(int sub_api, struct usbi_transfer *itransfer);
	int (*abort_control)(int sub_api, struct usbi_transfer *itransfer);
	int (*abort_transfers)(int sub_api, struct usbi_transfer *itransfer);
	int (*copy_transfer_data)(int sub_api, struct usbi_transfer *itransfer, uint32_t io_size);
};

/*
 * private structures definition
 * with  pseudo constructors/destructors
 */

// TODO (v2+): move hid desc to libusb.h?
struct libusb_hid_descriptor {
	uint8_t bLength;
	uint8_t bDescriptorType;
	uint16_t bcdHID;
	uint8_t bCountryCode;
	uint8_t bNumDescriptors;
	uint8_t bClassDescriptorType;
	uint16_t wClassDescriptorLength;
};

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

struct hid_device_priv {
	uint16_t vid;
	uint16_t pid;
	uint8_t config;
	uint8_t nb_interfaces;
	bool uses_report_ids[3]; // input, ouptput, feature
	uint16_t input_report_size;
	uint16_t output_report_size;
	uint16_t feature_report_size;
	WCHAR string[3][MAX_USB_STRING_LENGTH];
	uint8_t string_index[3]; // man, prod, ser
};

struct windows_device_priv {
	uint8_t depth; // distance to HCD
	uint8_t port;  // port number on the hub
	uint8_t active_config;
	struct windows_usb_api_backend const *apib;
	char *path;  // device interface path
	int sub_api; // for WinUSB-like APIs
	struct {
		char *path; // each interface needs a device interface path,
		struct windows_usb_api_backend const *apib; // an API backend (multiple drivers support),
		int sub_api;
		int8_t nb_endpoints; // and a set of endpoint addresses (USB_MAXENDPOINTS)
		uint8_t *endpoint;
		bool restricted_functionality;  // indicates if the interface functionality is restricted
                                                // by Windows (eg. HID keyboards or mice cannot do R/W)
	} usb_interface[USB_MAXINTERFACES];
	struct hid_device_priv *hid;
	USB_DEVICE_DESCRIPTOR dev_descriptor;
	unsigned char **config_descriptor; // list of pointers to the cached config descriptors
};

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

static  void windows_device_priv_release(struct libusb_device *dev)
{

}

struct interface_handle_t {
	HANDLE dev_handle; // WinUSB needs an extra handle for the file
	HANDLE api_handle; // used by the API to communicate with the device
};

struct windows_device_handle_priv {
	int active_interface;
	struct interface_handle_t interface_handle[USB_MAXINTERFACES];
	int autoclaim_count[USB_MAXINTERFACES]; // For auto-release
};

static  struct windows_device_handle_priv *_device_handle_priv(
	struct libusb_device_handle *handle)
{
	return (struct windows_device_handle_priv *)handle->os_priv;
}

// used for async polling functions
struct windows_transfer_priv {
	struct winfd pollable_fd;
	uint8_t interface_number;
	uint8_t *hid_buffer; // 1 byte extended data buffer, required for HID
	uint8_t *hid_dest;   // transfer buffer destination, required for HID
	size_t hid_expected_size;
};

// used to match a device driver (including filter drivers) against a supported API
struct driver_lookup {
	char list[MAX_KEY_LENGTH + 1]; // REG_MULTI_SZ list of services (driver) names
	const DWORD reg_prop;          // SPDRP registry key to use to retrieve list
	const char* designation;       // internal designation (for debug output)
};

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

typedef struct USB_INTERFACE_DESCRIPTOR {
	uint8 bLength;
	uint8 bDescriptorType;
	uint8 bInterfaceNumber;
	uint8 bAlternateSetting;
	uint8 bNumEndpoints;
	uint8 bInterfaceClass;
	uint8 bInterfaceSubClass;
	uint8 bInterfaceProtocol;
	uint8 iInterface;
} USB_INTERFACE_DESCRIPTOR, *PUSB_INTERFACE_DESCRIPTOR;

typedef struct USB_CONFIGURATION_DESCRIPTOR_SHORT {
	struct {
		uint64 ConnectionIndex;
		struct {
			uint8 bmRequest;
			uint8 bRequest;
			uint16 wValue;
			uint16 wIndex;
			uint16 wLength;
		} SetupPacket;
	} req;
	USB_CONFIGURATION_DESCRIPTOR data;
} USB_CONFIGURATION_DESCRIPTOR_SHORT;

typedef struct USB_ENDPOINT_DESCRIPTOR {
	uint8 bLength;
	uint8 bDescriptorType;
	uint8 bEndpointAddress;
	uint8 bmAttributes;
	uint16 wMaxPacketSize;
	uint8 bInterval;
} USB_ENDPOINT_DESCRIPTOR, *PUSB_ENDPOINT_DESCRIPTOR;

typedef struct USB_DESCRIPTOR_REQUEST {
	uint64 ConnectionIndex;
	struct {
		uint8 bmRequest;
		uint8 bRequest;
		uint16 wValue;
		uint16 wIndex;
		uint16 wLength;
	} SetupPacket;
//	uint8 Data[0];
} USB_DESCRIPTOR_REQUEST, *PUSB_DESCRIPTOR_REQUEST;

typedef struct USB_HUB_DESCRIPTOR {
	uint8 bDescriptorLength;
	uint8 bDescriptorType;
	uint8 bNumberOfPorts;
	uint16 wHubCharacteristics;
	uint8 bPowerOnToPowerGood;
	uint8 bHubControlCurrent;
	uint8 bRemoveAndPowerMask[64];
} USB_HUB_DESCRIPTOR, *PUSB_HUB_DESCRIPTOR;

typedef struct USB_ROOT_HUB_NAME {
	uint64 ActualLength;
	WCHAR RootHubName[1];
} USB_ROOT_HUB_NAME, *PUSB_ROOT_HUB_NAME;

typedef struct USB_ROOT_HUB_NAME_FIXED {
	uint64 ActualLength;
	WCHAR RootHubName[MAX_PATH_LENGTH];
} USB_ROOT_HUB_NAME_FIXED;

typedef struct USB_NODE_CONNECTION_NAME {
	uint64 ConnectionIndex;
	uint64 ActualLength;
	WCHAR NodeName[1];
} USB_NODE_CONNECTION_NAME, *PUSB_NODE_CONNECTION_NAME;

typedef struct USB_NODE_CONNECTION_NAME_FIXED {
	uint64 ConnectionIndex;
	uint64 ActualLength;
	WCHAR NodeName[MAX_PATH_LENGTH];
} USB_NODE_CONNECTION_NAME_FIXED;

typedef struct USB_HUB_NAME_FIXED {
	union {
		USB_ROOT_HUB_NAME_FIXED root;
		USB_NODE_CONNECTION_NAME_FIXED node;
	} u;
} USB_HUB_NAME_FIXED;

typedef struct USB_HUB_INFORMATION {
	USB_HUB_DESCRIPTOR HubDescriptor;
	bool HubIsBusPowered;
} USB_HUB_INFORMATION, *PUSB_HUB_INFORMATION;

typedef struct USB_MI_PARENT_INFORMATION {
	uint64 NumberOfInterfaces;
} USB_MI_PARENT_INFORMATION, *PUSB_MI_PARENT_INFORMATION;

typedef struct USB_NODE_INFORMATION {
	USB_HUB_NODE NodeType;
	union {
		USB_HUB_INFORMATION HubInformation;
		USB_MI_PARENT_INFORMATION MiParentInformation;
	} u;
} USB_NODE_INFORMATION, *PUSB_NODE_INFORMATION;

typedef struct USB_PIPE_INFO {
	USB_ENDPOINT_DESCRIPTOR EndpointDescriptor;
	uint64 ScheduleOffset;
} USB_PIPE_INFO, *PUSB_PIPE_INFO;

typedef struct USB_NODE_CONNECTION_INFORMATION_EX {
	uint64 ConnectionIndex;
	USB_DEVICE_DESCRIPTOR DeviceDescriptor;
	uint8 CurrentConfigurationValue;
	uint8 Speed;
	bool DeviceIsHub;
	uint16 DeviceAddress;
	uint64 NumberOfOpenPipes;
	USB_CONNECTION_STATUS ConnectionStatus;
//	USB_PIPE_INFO PipeList[0];
} USB_NODE_CONNECTION_INFORMATION_EX, *PUSB_NODE_CONNECTION_INFORMATION_EX;

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

typedef struct _USB_NODE_CONNECTION_INFORMATION_EX_V2 {
	uint64 ConnectionIndex;
	uint64 Length;
	USB_PROTOCOLS SupportedUsbProtocols;
	USB_NODE_CONNECTION_INFORMATION_EX_V2_FLAGS Flags;
} USB_NODE_CONNECTION_INFORMATION_EX_V2, *PUSB_NODE_CONNECTION_INFORMATION_EX_V2;

typedef struct USB_HUB_CAP_FLAGS {
	uint64 HubIsHighSpeedCapable:1;
	uint64 HubIsHighSpeed:1;
	uint64 HubIsMultiTtCapable:1;
	uint64 HubIsMultiTt:1;
	uint64 HubIsRoot:1;
	uint64 HubIsArmedWakeOnConnect:1;
	uint64 ReservedMBZ:26;
} USB_HUB_CAP_FLAGS, *PUSB_HUB_CAP_FLAGS;

typedef struct USB_HUB_CAPABILITIES {
	uint64 HubIs2xCapable:1;
} USB_HUB_CAPABILITIES, *PUSB_HUB_CAPABILITIES;

typedef struct USB_HUB_CAPABILITIES_EX {
	USB_HUB_CAP_FLAGS CapabilityFlags;
} USB_HUB_CAPABILITIES_EX, *PUSB_HUB_CAPABILITIES_EX;

typedef struct {
	USBD_PIPE_TYPE PipeType;
	uint8 PipeId;
	uint16 MaximumPacketSize;
	uint8 Interval;
} WINUSB_PIPE_INFORMATION, *PWINUSB_PIPE_INFORMATION;

typedef struct {
	uint8 request_type;
	uint8 request;
	uint16 value;
	uint16 index;
	uint16 length;
} WINUSB_SETUP_PACKET, *PWINUSB_SETUP_PACKET;

typedef void *WINUSB_INTERFACE_HANDLE, *PWINUSB_INTERFACE_HANDLE;

typedef BOOL (WINAPI *WinUsb_AbortPipe_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID
);
typedef BOOL (WINAPI *WinUsb_ControlTransfer_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	WINUSB_SETUP_PACKET SetupPacket,
	PUCHAR Buffer,
	uint64 BufferLength,
	PULONG LengthTransferred,
	LPOVERLAPPED Overlapped
);
typedef BOOL (WINAPI *WinUsb_FlushPipe_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID
);
typedef BOOL (WINAPI *WinUsb_Free_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle
);
typedef BOOL (WINAPI *WinUsb_GetAssociatedInterface_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 AssociatedInterfaceIndex,
	PWINUSB_INTERFACE_HANDLE AssociatedInterfaceHandle
);
typedef BOOL (WINAPI *WinUsb_GetCurrentAlternateSetting_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	PUCHAR AlternateSetting
);
typedef BOOL (WINAPI *WinUsb_GetDescriptor_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 DescriptorType,
	uint8 Index,
	uint16 LanguageID,
	PUCHAR Buffer,
	uint64 BufferLength,
	PULONG LengthTransferred
);
typedef BOOL (WINAPI *WinUsb_GetOverlappedResult_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	LPOVERLAPPED lpOverlapped,
	LPDWORD lpNumberOfBytesTransferred,
	BOOL bWait
);
typedef BOOL (WINAPI *WinUsb_GetPipePolicy_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID,
	uint64 PolicyType,
	PULONG ValueLength,
	PVOID Value
);
typedef BOOL (WINAPI *WinUsb_GetPowerPolicy_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint64 PolicyType,
	PULONG ValueLength,
	PVOID Value
);
typedef BOOL (WINAPI *WinUsb_Initialize_t)(
	HANDLE DeviceHandle,
	PWINUSB_INTERFACE_HANDLE InterfaceHandle
);
typedef BOOL (WINAPI *WinUsb_QueryDeviceInformation_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint64 InformationType,
	PULONG BufferLength,
	PVOID Buffer
);
typedef BOOL (WINAPI *WinUsb_QueryInterfaceSettings_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 AlternateSettingNumber,
	PUSB_INTERFACE_DESCRIPTOR UsbAltInterfaceDescriptor
);
typedef BOOL (WINAPI *WinUsb_QueryPipe_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 AlternateInterfaceNumber,
	uint8 PipeIndex,
	PWINUSB_PIPE_INFORMATION PipeInformation
);
typedef BOOL (WINAPI *WinUsb_ReadPipe_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID,
	PUCHAR Buffer,
	uint64 BufferLength,
	PULONG LengthTransferred,
	LPOVERLAPPED Overlapped
);
typedef BOOL (WINAPI *WinUsb_ResetPipe_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID
);
typedef BOOL (WINAPI *WinUsb_SetCurrentAlternateSetting_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 AlternateSetting
);
typedef BOOL (WINAPI *WinUsb_SetPipePolicy_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID,
	uint64 PolicyType,
	uint64 ValueLength,
	PVOID Value
);
typedef BOOL (WINAPI *WinUsb_SetPowerPolicy_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint64 PolicyType,
	uint64 ValueLength,
	PVOID Value
);
typedef BOOL (WINAPI *WinUsb_WritePipe_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID,
	PUCHAR Buffer,
	uint64 BufferLength,
	PULONG LengthTransferred,
	LPOVERLAPPED Overlapped
);
typedef BOOL (WINAPI *WinUsb_ResetDevice_t)(
	WINUSB_INTERFACE_HANDLE InterfaceHandle
);

typedef struct _KLIB_VERSION {
	INT Major;
	INT Minor;
	INT Micro;
	INT Nano;
} KLIB_VERSION;
typedef KLIB_VERSION* PKLIB_VERSION;

typedef BOOL (WINAPI *LibK_GetProcAddress_t)(
	PVOID *ProcAddress,
	uint64 DriverID,
	uint64 FunctionID
);

typedef VOID (WINAPI *LibK_GetVersion_t)(
	PKLIB_VERSION Version
);

struct winusb_interface {
	bool initialized;
	WinUsb_AbortPipe_t AbortPipe;
	WinUsb_ControlTransfer_t ControlTransfer;
	WinUsb_FlushPipe_t FlushPipe;
	WinUsb_Free_t Free;
	WinUsb_GetAssociatedInterface_t GetAssociatedInterface;
	WinUsb_GetCurrentAlternateSetting_t GetCurrentAlternateSetting;
	WinUsb_GetDescriptor_t GetDescriptor;
	WinUsb_GetOverlappedResult_t GetOverlappedResult;
	WinUsb_GetPipePolicy_t GetPipePolicy;
	WinUsb_GetPowerPolicy_t GetPowerPolicy;
	WinUsb_Initialize_t Initialize;
	WinUsb_QueryDeviceInformation_t QueryDeviceInformation;
	WinUsb_QueryInterfaceSettings_t QueryInterfaceSettings;
	WinUsb_QueryPipe_t QueryPipe;
	WinUsb_ReadPipe_t ReadPipe;
	WinUsb_ResetPipe_t ResetPipe;
	WinUsb_SetCurrentAlternateSetting_t SetCurrentAlternateSetting;
	WinUsb_SetPipePolicy_t SetPipePolicy;
	WinUsb_SetPowerPolicy_t SetPowerPolicy;
	WinUsb_WritePipe_t WritePipe;
	WinUsb_ResetDevice_t ResetDevice;
};

/* hid.dll interface */

typedef void * PHIDP_PREPARSED_DATA;

typedef struct {
	uint64 Size;
	uint16 VendorID;
	uint16 ProductID;
	uint16 VersionNumber;
} HIDD_ATTRIBUTES, *PHIDD_ATTRIBUTES;

typedef uint16 USAGE;
typedef struct {
	USAGE Usage;
	USAGE UsagePage;
	uint16 InputReportByteLength;
	uint16 OutputReportByteLength;
	uint16 FeatureReportByteLength;
	uint16 Reserved[17];
	uint16 NumberLinkCollectionNodes;
	uint16 NumberInputButtonCaps;
	uint16 NumberInputValueCaps;
	uint16 NumberInputDataIndices;
	uint16 NumberOutputButtonCaps;
	uint16 NumberOutputValueCaps;
	uint16 NumberOutputDataIndices;
	uint16 NumberFeatureButtonCaps;
	uint16 NumberFeatureValueCaps;
	uint16 NumberFeatureDataIndices;
} HIDP_CAPS, *PHIDP_CAPS;

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