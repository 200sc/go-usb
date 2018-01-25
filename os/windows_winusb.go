package os

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

const SPDRP_ADDRESS	= 28
const SPDRP_INSTALL_STATE = 34
const LIBUSB_COMPLETED = (LIBUSB_SUCCESS + 1)

var WINUSBX_DRV_NAMES = []string{"libusbK", "libusb0", "WinUSB"}

type libusb_hid_request_type uint8
const (
	HID_REQ_GET_REPORT libusb_hid_request_type = 0x01,
	HID_REQ_GET_IDLE libusb_hid_request_type = 0x02,
	HID_REQ_GET_PROTOCOL libusb_hid_request_type = 0x03,
	HID_REQ_SET_REPORT libusb_hid_request_type = 0x09,
	HID_REQ_SET_IDLE libusb_hid_request_type = 0x0A,
	HID_REQ_SET_PROTOCOL libusb_hid_request_type = 0x0B
)

type libusb_hid_report_type uint8
const (
	HID_REPORT_TYPE_INPUT libusb_hid_report_type = 0x01,
	HID_REPORT_TYPE_OUTPUT libusb_hid_report_type = 0x02,
	HID_REPORT_TYPE_FEATURE libusb_hid_report_type = 0x03
)

type USB_CONNECTION_STATUS uint8 
const (
	NoDeviceConnected USB_CONNECTION_STATUS = iota
	DeviceConnected USB_CONNECTION_STATUS
	DeviceFailedEnumeration USB_CONNECTION_STATUS
	DeviceGeneralFailure USB_CONNECTION_STATUS
	DeviceCausedOvercurrent USB_CONNECTION_STATUS
	DeviceNotEnoughPower USB_CONNECTION_STATUS
	DeviceNotEnoughBandwidth USB_CONNECTION_STATUS
	DeviceHubNestedTooDeeply USB_CONNECTION_STATUS
	DeviceInLegacyHub USB_CONNECTION_STATUS
)

type USB_HUB_NODE uint8
const (
	UsbHub USB_HUB_NODE = iota
	UsbMIParent USB_HUB_NODE
)

type USBD_PIPE_TYPE uint8
const (
	UsbdPipeTypeControl USBD_PIPE_TYPE = iota
	UsbdPipeTypeIsochronous USBD_PIPE_TYPE
	UsbdPipeTypeBulk USBD_PIPE_TYPE
	UsbdPipeTypeInterrupt USBD_PIPE_TYPE
)

/* /!\ These must match the ones from the official libusbk.h */
type _KUSB_FNID uint8
const (
	KUSB_FNID_Init _KUSB_FNID = iota
	KUSB_FNID_Free _KUSB_FNID
	KUSB_FNID_ClaimInterface _KUSB_FNID
	KUSB_FNID_ReleaseInterface _KUSB_FNID
	KUSB_FNID_SetAltInterface _KUSB_FNID
	KUSB_FNID_GetAltInterface _KUSB_FNID
	KUSB_FNID_GetDescriptor _KUSB_FNID
	KUSB_FNID_ControlTransfer _KUSB_FNID
	KUSB_FNID_SetPowerPolicy _KUSB_FNID
	KUSB_FNID_GetPowerPolicy _KUSB_FNID
	KUSB_FNID_SetConfiguration _KUSB_FNID
	KUSB_FNID_GetConfiguration _KUSB_FNID
	KUSB_FNID_ResetDevice _KUSB_FNID
	KUSB_FNID_Initialize _KUSB_FNID
	KUSB_FNID_SelectInterface _KUSB_FNID
	KUSB_FNID_GetAssociatedInterface _KUSB_FNID
	KUSB_FNID_Clone _KUSB_FNID
	KUSB_FNID_QueryInterfaceSettings _KUSB_FNID
	KUSB_FNID_QueryDeviceInformation _KUSB_FNID
	KUSB_FNID_SetCurrentAlternateSetting _KUSB_FNID
	KUSB_FNID_GetCurrentAlternateSetting _KUSB_FNID
	KUSB_FNID_QueryPipe _KUSB_FNID
	KUSB_FNID_SetPipePolicy _KUSB_FNID
	KUSB_FNID_GetPipePolicy _KUSB_FNID
	KUSB_FNID_ReadPipe _KUSB_FNID
	KUSB_FNID_WritePipe _KUSB_FNID
	KUSB_FNID_ResetPipe _KUSB_FNID
	KUSB_FNID_AbortPipe _KUSB_FNID
	KUSB_FNID_FlushPipe _KUSB_FNID
	KUSB_FNID_IsoReadPipe _KUSB_FNID
	KUSB_FNID_IsoWritePipe _KUSB_FNID
	KUSB_FNID_GetCurrentFrameNumber _KUSB_FNID
	KUSB_FNID_GetOverlappedResult _KUSB_FNID
	KUSB_FNID_GetProperty _KUSB_FNID
	KUSB_FNID_COUNT _KUSB_FNID
)

type _HIDP_REPORT_TYPE uint8
const (
	HidP_Input _HIDP_REPORT_TYPE = iota
	HidP_Output _HIDP_REPORT_TYPE
	HidP_Feature _HIDP_REPORT_TYPE
)

const(
	MAX_CTRL_BUFFER_LENGTH = 4096
	MAX_USB_DEVICES = 256
	MAX_USB_STRING_LENGTH = 128
	MAX_HID_REPORT_SIZE = 1024
	MAX_HID_DESCRIPTOR_SIZE = 256
	MAX_GUID_STRING_LENGTH = 40
	MAX_PATH_LENGTH = 128
	MAX_KEY_LENGTH	= 256
	LIST_SEPARATOR	= ';'
   )
   
   /*
	* Multiple USB API backend support
	*/
   const (
	USB_API_UNSUPPORTED = 0
	USB_API_HUB = 1
	USB_API_COMPOSITE = 2
	USB_API_WINUSBX = 3
	USB_API_HID = 4
	USB_API_MAX = 5
   // The following is used to indicate if the HID or composite extra props have already been set.
	USB_API_SET = (1 << USB_API_MAX)
   
   // Sub-APIs for WinUSB-like driver APIs (WinUSB, libusbK, libusb-win32 through the libusbK DLL)
   // Must have the same values as the KUSB_DRVID enum from libusbk.h
	SUB_API_NOTSET = -1
	SUB_API_LIBUSBK = 0
	SUB_API_LIBUSB0 = 1
	SUB_API_WINUSB	= 2
	SUB_API_MAX = 3
   )
   
   const(
	   LIBUSB_DT_HID_SIZE = 9 
	   HID_MAX_REPORT_SIZE	= 1024
	   HID_IN_EP = 0x81
	   HID_OUT_EP = 0x02
   )
   
   const(
	   CR_SUCCESS = 0x00000000
	   CR_NO_SUCH_DEVNODE = 0x0000000D
   
	USB_DEVICE_DESCRIPTOR_TYPE	= LIBUSB_DT_DEVICE
	USB_CONFIGURATION_DESCRIPTOR_TYPE = LIBUSB_DT_CONFIG
	USB_STRING_DESCRIPTOR_TYPE	= LIBUSB_DT_STRING
	USB_INTERFACE_DESCRIPTOR_TYPE = LIBUSB_DT_INTERFACE
	USB_ENDPOINT_DESCRIPTOR_TYPE = LIBUSB_DT_ENDPOINT
   
	USB_REQUEST_GET_STATUS	= LIBUSB_REQUEST_GET_STATUS
	USB_REQUEST_CLEAR_FEATURE = LIBUSB_REQUEST_CLEAR_FEATURE
	USB_REQUEST_SET_FEATURE = LIBUSB_REQUEST_SET_FEATURE
	USB_REQUEST_SET_ADDRESS = LIBUSB_REQUEST_SET_ADDRESS
	USB_REQUEST_GET_DESCRIPTOR	= LIBUSB_REQUEST_GET_DESCRIPTOR
	USB_REQUEST_SET_DESCRIPTOR	= LIBUSB_REQUEST_SET_DESCRIPTOR
	USB_REQUEST_GET_CONFIGURATION = LIBUSB_REQUEST_GET_CONFIGURATION
	USB_REQUEST_SET_CONFIGURATION = LIBUSB_REQUEST_SET_CONFIGURATION
	USB_REQUEST_GET_INTERFACE = LIBUSB_REQUEST_GET_INTERFACE
	USB_REQUEST_SET_INTERFACE = LIBUSB_REQUEST_SET_INTERFACE
	USB_REQUEST_SYNC_FRAME	= LIBUSB_REQUEST_SYNCH_FRAME
   
	   USB_GET_NODE_INFORMATION = 258
	   USB_GET_DESCRIPTOR_FROM_NODE_CONNECTION = 260
	   USB_GET_NODE_CONNECTION_NAME = 261
	   USB_GET_HUB_CAPABILITIES = 271
   )
   
   const(
	   SHORT_PACKET_TERMINATE = 0x01
	AUTO_CLEAR_STALL = 0x02
	PIPE_TRANSFER_TIMEOUT	= 0x03
	IGNORE_SHORT_PACKETS = 0x04
	ALLOW_PARTIAL_READS = 0x05
	AUTO_FLUSH = 0x06
	RAW_IO = 0x07
	MAXIMUM_TRANSFER_SIZE	= 0x08
	AUTO_SUSPEND = 0x81
	SUSPEND_DELAY = 0x83
	DEVICE_SPEED = 0x01
	LowSpeed = 0x01
	FullSpeed = 0x02
	HighSpeed = 0x03
   )
   
   const HIDP_STATUS_SUCCESS = 0x110000


   struct windows_usb_api_backend {
	const uint8 id
	const char *designation
	const char **driver_name_list // Driver name, without .sys, e.g. "usbccgp"
	const uint8 nb_driver_names
	int (*init)(int sub_api, struct libusb_context *ctx)
	int (*exit)(int sub_api)
	int (*open)(int sub_api, struct libusb_device_handle *dev_handle)
	void (*close)(int sub_api, struct libusb_device_handle *dev_handle)
	int (*configure_endpoints)(int sub_api, struct libusb_device_handle *dev_handle, int iface)
	int (*claim_interface)(int sub_api, struct libusb_device_handle *dev_handle, int iface)
	int (*set_interface_altsetting)(int sub_api, struct libusb_device_handle *dev_handle, int iface, int altsetting)
	int (*release_interface)(int sub_api, struct libusb_device_handle *dev_handle, int iface)
	int (*clear_halt)(int sub_api, struct libusb_device_handle *dev_handle, uint8 endpoint)
	int (*reset_device)(int sub_api, struct libusb_device_handle *dev_handle)
	int (*submit_bulk_transfer)(int sub_api, struct usbi_transfer *itransfer)
	int (*submit_iso_transfer)(int sub_api, struct usbi_transfer *itransfer)
	int (*submit_control_transfer)(int sub_api, struct usbi_transfer *itransfer)
	int (*abort_control)(int sub_api, struct usbi_transfer *itransfer)
	int (*abort_transfers)(int sub_api, struct usbi_transfer *itransfer)
	int (*copy_transfer_data)(int sub_api, struct usbi_transfer *itransfer, uint32 io_size)
}

/*
 * private structures definition
 * with  pseudo constructors/destructors
 */

// TODO (v2+): move hid desc to libusb.h?
struct libusb_hid_descriptor {
	uint8 bLength
	uint8 bDescriptorType
	uint16 bcdHID
	uint8 bCountryCode
	uint8 bNumDescriptors
	uint8 bClassDescriptorType
	uint16 wClassDescriptorLength
}

struct hid_device_priv {
	uint16 vid
	uint16 pid
	uint8 config
	uint8 nb_interfaces
	bool uses_report_ids[3] // input, ouptput, feature
	uint16 input_report_size
	uint16 output_report_size
	uint16 feature_report_size
	WCHAR string[3][MAX_USB_STRING_LENGTH]
	uint8 string_index[3] // man, prod, ser
}

struct windows_device_priv {
	uint8 depth // distance to HCD
	uint8 port  // port number on the hub
	uint8 active_config
	struct windows_usb_api_backend const *apib
	char *path  // device interface path
	int sub_api // for WinUSB-like APIs
	struct {
		char *path // each interface needs a device interface path,
		struct windows_usb_api_backend const *apib // an API backend (multiple drivers support),
		int sub_api
		int8_t nb_endpoints // and a set of endpoint addresses (USB_MAXENDPOINTS)
		uint8 *endpoint
		bool restricted_functionality  // indicates if the interface functionality is restricted
                                                // by Windows (eg. HID keyboards or mice cannot do R/W)
	} usb_interface[USB_MAXINTERFACES]
	struct hid_device_priv *hid
	USB_DEVICE_DESCRIPTOR dev_descriptor
	uint8 **config_descriptor // list of pointers to the cached config descriptors
}

struct interface_handle_t {
	HANDLE dev_handle // WinUSB needs an extra handle for the file
	HANDLE api_handle // used by the API to communicate with the device
}

struct windows_device_handle_priv {
	int active_interface
	struct interface_handle_t interface_handle[USB_MAXINTERFACES]
	int autoclaim_count[USB_MAXINTERFACES] // For auto-release
}

// used for async polling functions
struct windows_transfer_priv {
	struct winfd pollable_fd
	uint8 interface_number
	uint8 *hid_buffer // 1 byte extended data buffer, required for HID
	uint8 *hid_dest   // transfer buffer destination, required for HID
	int hid_expected_size
}

// used to match a device driver (including filter drivers) against a supported API
struct driver_lookup {
	char list[MAX_KEY_LENGTH + 1] // REG_MULTI_SZ list of services (driver) names
	const DWORD reg_prop          // SPDRP registry key to use to retrieve list
	const char* designation       // internal designation (for debug output)
}

type struct USB_INTERFACE_DESCRIPTOR {
	uint8 bLength
	uint8 bDescriptorType
	uint8 bInterfaceNumber
	uint8 bAlternateSetting
	uint8 bNumEndpoints
	uint8 bInterfaceClass
	uint8 bInterfaceSubClass
	uint8 bInterfaceProtocol
	uint8 iInterface
} USB_INTERFACE_DESCRIPTOR

type struct USB_CONFIGURATION_DESCRIPTOR_SHORT {
	struct {
		uint64 ConnectionIndex
		struct {
			uint8 bmRequest
			uint8 bRequest
			uint16 wValue
			uint16 wIndex
			uint16 wLength
		} SetupPacket
	} req
	USB_CONFIGURATION_DESCRIPTOR data
} USB_CONFIGURATION_DESCRIPTOR_SHORT

type struct USB_ENDPOINT_DESCRIPTOR {
	uint8 bLength
	uint8 bDescriptorType
	uint8 bEndpointAddress
	uint8 bmAttributes
	uint16 wMaxPacketSize
	uint8 bInterval
} USB_ENDPOINT_DESCRIPTOR

type struct USB_DESCRIPTOR_REQUEST {
	uint64 ConnectionIndex
	struct {
		uint8 bmRequest
		uint8 bRequest
		uint16 wValue
		uint16 wIndex
		uint16 wLength
	} SetupPacket
//	uint8 Data[0]
} USB_DESCRIPTOR_REQUEST

type struct USB_HUB_DESCRIPTOR {
	uint8 bDescriptorLength
	uint8 bDescriptorType
	uint8 bNumberOfPorts
	uint16 wHubCharacteristics
	uint8 bPowerOnToPowerGood
	uint8 bHubControlCurrent
	uint8 bRemoveAndPowerMask[64]
} USB_HUB_DESCRIPTOR

type struct USB_ROOT_HUB_NAME {
	uint64 ActualLength
	WCHAR RootHubName[1]
} USB_ROOT_HUB_NAME

type struct USB_ROOT_HUB_NAME_FIXED {
	uint64 ActualLength
	WCHAR RootHubName[MAX_PATH_LENGTH]
} USB_ROOT_HUB_NAME_FIXED

type struct USB_NODE_CONNECTION_NAME {
	uint64 ConnectionIndex
	uint64 ActualLength
	WCHAR NodeName[1]
} USB_NODE_CONNECTION_NAME

type struct USB_NODE_CONNECTION_NAME_FIXED {
	uint64 ConnectionIndex
	uint64 ActualLength
	WCHAR NodeName[MAX_PATH_LENGTH]
} USB_NODE_CONNECTION_NAME_FIXED

type struct USB_HUB_INFORMATION {
	USB_HUB_DESCRIPTOR HubDescriptor
	bool HubIsBusPowered
} USB_HUB_INFORMATION

type struct USB_MI_PARENT_INFORMATION {
	uint64 NumberOfInterfaces
} USB_MI_PARENT_INFORMATION

type struct USB_NODE_INFORMATION {
	USB_HUB_NODE NodeType
	union {
		USB_HUB_INFORMATION HubInformation
		USB_MI_PARENT_INFORMATION MiParentInformation
	} u
} USB_NODE_INFORMATION

type struct USB_PIPE_INFO {
	USB_ENDPOINT_DESCRIPTOR EndpointDescriptor
	uint64 ScheduleOffset
} USB_PIPE_INFO

type struct USB_NODE_CONNECTION_INFORMATION_EX {
	uint64 ConnectionIndex
	USB_DEVICE_DESCRIPTOR DeviceDescriptor
	uint8 CurrentConfigurationValue
	uint8 Speed
	bool DeviceIsHub
	uint16 DeviceAddress
	uint64 NumberOfOpenPipes
	USB_CONNECTION_STATUS ConnectionStatus
//	USB_PIPE_INFO PipeList[0]
} USB_NODE_CONNECTION_INFORMATION_EX

type struct _USB_NODE_CONNECTION_INFORMATION_EX_V2 {
	uint64 ConnectionIndex
	uint64 Length
	USB_PROTOCOLS SupportedUsbProtocols
	USB_NODE_CONNECTION_INFORMATION_EX_V2_FLAGS Flags
} USB_NODE_CONNECTION_INFORMATION_EX_V2

type struct USB_HUB_CAP_FLAGS {
	uint64 HubIsHighSpeedCapable:1
	uint64 HubIsHighSpeed:1
	uint64 HubIsMultiTtCapable:1
	uint64 HubIsMultiTt:1
	uint64 HubIsRoot:1
	uint64 HubIsArmedWakeOnConnect:1
	uint64 ReservedMBZ:26
} USB_HUB_CAP_FLAGS

type struct USB_HUB_CAPABILITIES {
	uint64 HubIs2xCapable:1
} USB_HUB_CAPABILITIES

type struct USB_HUB_CAPABILITIES_EX {
	USB_HUB_CAP_FLAGS CapabilityFlags
} USB_HUB_CAPABILITIES_EX

type struct {
	USBD_PIPE_TYPE PipeType
	uint8 PipeId
	uint16 MaximumPacketSize
	uint8 Interval
} WINUSB_PIPE_INFORMATION

type struct {
	uint8 request_type
	uint8 request
	uint16 value
	uint16 index
	uint16 length
} WINUSB_SETUP_PACKET

type void *WINUSB_INTERFACE_HANDLE, *PWINUSB_INTERFACE_HANDLE

type  WinUsb_AbortPipe_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID
) bool
type  WinUsb_ControlTransfer_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	WINUSB_SETUP_PACKET SetupPacket,
	PUCHAR Buffer,
	uint64 BufferLength,
	PULONG LengthTransferred,
	LPOVERLAPPED Overlapped
) bool
type  WinUsb_FlushPipe_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID
) bool
type  WinUsb_Free_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle
) bool
type  WinUsb_GetAssociatedInterface_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 AssociatedInterfaceIndex,
	PWINUSB_INTERFACE_HANDLE AssociatedInterfaceHandle
) bool
type  WinUsb_GetCurrentAlternateSetting_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	PUCHAR AlternateSetting
) bool
type  WinUsb_GetDescriptor_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 DescriptorType,
	uint8 Index,
	uint16 LanguageID,
	PUCHAR Buffer,
	uint64 BufferLength,
	PULONG LengthTransferred
) bool
type  WinUsb_GetOverlappedResult_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	LPOVERLAPPED lpOverlapped,
	LPDWORD lpNumberOfBytesTransferred,
	BOOL bWait
) bool
type  WinUsb_GetPipePolicy_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID,
	uint64 PolicyType,
	PULONG ValueLength,
	PVOID Value
) bool
type  WinUsb_GetPowerPolicy_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint64 PolicyType,
	PULONG ValueLength,
	PVOID Value
) bool
type  WinUsb_Initialize_t func(
	HANDLE DeviceHandle,
	PWINUSB_INTERFACE_HANDLE InterfaceHandle
) bool
type  WinUsb_QueryDeviceInformation_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint64 InformationType,
	PULONG BufferLength,
	PVOID Buffer
) bool
type  WinUsb_QueryInterfaceSettings_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 AlternateSettingNumber,
	PUSB_INTERFACE_DESCRIPTOR UsbAltInterfaceDescriptor
) bool
type  WinUsb_QueryPipe_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 AlternateInterfaceNumber,
	uint8 PipeIndex,
	PWINUSB_PIPE_INFORMATION PipeInformation
) bool
type  WinUsb_ReadPipe_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID,
	PUCHAR Buffer,
	uint64 BufferLength,
	PULONG LengthTransferred,
	LPOVERLAPPED Overlapped
) bool
type  WinUsb_ResetPipe_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID
) bool
type  WinUsb_SetCurrentAlternateSetting_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 AlternateSetting
) bool
type  WinUsb_SetPipePolicy_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID,
	uint64 PolicyType,
	uint64 ValueLength,
	PVOID Value
) bool
type  WinUsb_SetPowerPolicy_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint64 PolicyType,
	uint64 ValueLength,
	PVOID Value
) bool
type  WinUsb_WritePipe_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle,
	uint8 PipeID,
	PUCHAR Buffer,
	uint64 BufferLength,
	PULONG LengthTransferred,
	LPOVERLAPPED Overlapped
) bool
type  WinUsb_ResetDevice_t func(
	WINUSB_INTERFACE_HANDLE InterfaceHandle
) bool 

type KLIB_VERSION struct {
	int Major
	int Minor
	int Micro
	int Nano
} 

type LibK_GetProcAddress_t func(
	PVOID *ProcAddress, 
	uint64 DriverID, 
	uint64 FunctionID
) bool

type LibK_GetVersion_t func(
	PKLIB_VERSION Version
)

type winusb_interface struct {
	bool initialized
	WinUsb_AbortPipe_t AbortPipe
	WinUsb_ControlTransfer_t ControlTransfer
	WinUsb_FlushPipe_t FlushPipe
	WinUsb_Free_t Free
	WinUsb_GetAssociatedInterface_t GetAssociatedInterface
	WinUsb_GetCurrentAlternateSetting_t GetCurrentAlternateSetting
	WinUsb_GetDescriptor_t GetDescriptor
	WinUsb_GetOverlappedResult_t GetOverlappedResult
	WinUsb_GetPipePolicy_t GetPipePolicy
	WinUsb_GetPowerPolicy_t GetPowerPolicy
	WinUsb_Initialize_t Initialize
	WinUsb_QueryDeviceInformation_t QueryDeviceInformation
	WinUsb_QueryInterfaceSettings_t QueryInterfaceSettings
	WinUsb_QueryPipe_t QueryPipe
	WinUsb_ReadPipe_t ReadPipe
	WinUsb_ResetPipe_t ResetPipe
	WinUsb_SetCurrentAlternateSetting_t SetCurrentAlternateSetting
	WinUsb_SetPipePolicy_t SetPipePolicy
	WinUsb_SetPowerPolicy_t SetPowerPolicy
	WinUsb_WritePipe_t WritePipe
	WinUsb_ResetDevice_t ResetDevice
}

/* hid.dll interface */

type interface{} PHIDP_PREPARSED_DATA

type HIDD_ATTRIBUTES struct {
	uint64 Size
	uint16 VendorID
	uint16 ProductID
	uint16 VersionNumber
} 

type USAGE uint16 
type HIDP_CAPS struct {
	USAGE Usage
	USAGE UsagePage
	uint16 InputReportByteLength
	uint16 OutputReportByteLength
	uint16 FeatureReportByteLength
	uint16 Reserved[17]
	uint16 NumberLinkCollectionNodes
	uint16 NumberInputButtonCaps
	uint16 NumberInputValueCaps
	uint16 NumberInputDataIndices
	uint16 NumberOutputButtonCaps
	uint16 NumberOutputValueCaps
	uint16 NumberOutputDataIndices
	uint16 NumberFeatureButtonCaps
	uint16 NumberFeatureValueCaps
	uint16 NumberFeatureDataIndices
} 

const (
	LIBUSB_DT_HID_SIZE	=	9
	HID_MAX_CONFIG_DESC_SIZE = (LIBUSB_DT_CONFIG_SIZE + LIBUSB_DT_INTERFACE_SIZE + LIBUSB_DT_HID_SIZE + 2 * LIBUSB_DT_ENDPOINT_SIZE)
	HID_MAX_REPORT_SIZE =	1024
	HID_IN_EP =			0x81
	HID_OUT_EP	=		0x02
)

// Handle code for HID interface that have been claimed ("dibs")
const INTERFACE_CLAIMED 0xD1B5
// Additional return code for HID operations that completed synchronously
// http://msdn.microsoft.com/en-us/library/ff545978.aspx
// http://msdn.microsoft.com/en-us/library/ff545972.aspx
// http://msdn.microsoft.com/en-us/library/ff545982.aspx
var GUID_DEVINTERFACE_USB_HOST_CONTROLLER GUID = { 0x3ABF6F2D, 0x71C4, 0x462A, {0x8A, 0x92, 0x1E, 0x68, 0x61, 0xE6, 0xAF, 0x27} }
var GUID_DEVINTERFACE_USB_DEVICE GUID = { 0xA5DCBF10, 0x6530, 0x11D2, {0x90, 0x1F, 0x00, 0xC0, 0x4F, 0xB9, 0x51, 0xED} }
var GUID_DEVINTERFACE_USB_HUB GUID = { 0xF18A0E88, 0xC30C, 0x11D0, {0x88, 0x15, 0x00, 0xA0, 0xC9, 0x06, 0xBE, 0xD8} }
var GUID_DEVINTERFACE_LIBUSB0_FILTER GUID = { 0xF9F3FF14, 0xAE21, 0x48A0, {0x8A, 0x25, 0x80, 0x11, 0xA7, 0xA9, 0x31, 0xD9} }

/*
 * Windows DDK API definitions. Most of it copied from MinGW's includes
 */
const (
	USB_GET_NODE_CONNECTION_INFORMATION_EX =	274
	USB_GET_HUB_CAPABILITIES_EX	=	276
	USB_GET_NODE_CONNECTION_INFORMATION_EX_V2 =	279
	METHOD_BUFFERED	=			0
	FILE_ANY_ACCESS	=			0x00000000
	FILE_DEVICE_UNKNOWN =			0x00000022
	FILE_DEVICE_USB	=			FILE_DEVICE_UNKNOWN
)

func CTL_CODE(DeviceType, Function, Method, Access int) int {
	return (((DeviceType) << 16) | ((Access) << 14) | ((Function) << 2) | (Method))
}

var (
 IOCTL_USB_GET_HUB_CAPABILITIES_EX = CTL_CODE( FILE_DEVICE_USB, USB_GET_HUB_CAPABILITIES_EX, METHOD_BUFFERED, FILE_ANY_ACCESS)
 IOCTL_USB_GET_HUB_CAPABILITIES = CTL_CODE(FILE_DEVICE_USB, USB_GET_HUB_CAPABILITIES, METHOD_BUFFERED, FILE_ANY_ACCESS)
 IOCTL_USB_GET_DESCRIPTOR_FROM_NODE_CONNECTION = CTL_CODE(FILE_DEVICE_USB, USB_GET_DESCRIPTOR_FROM_NODE_CONNECTION, METHOD_BUFFERED, FILE_ANY_ACCESS)
 IOCTL_USB_GET_ROOT_HUB_NAME = CTL_CODE(FILE_DEVICE_USB, HCD_GET_ROOT_HUB_NAME, METHOD_BUFFERED, FILE_ANY_ACCESS)
 IOCTL_USB_GET_NODE_INFORMATION = CTL_CODE(FILE_DEVICE_USB, USB_GET_NODE_INFORMATION, METHOD_BUFFERED, FILE_ANY_ACCESS)
 IOCTL_USB_GET_NODE_CONNECTION_INFORMATION_EX = CTL_CODE(FILE_DEVICE_USB, USB_GET_NODE_CONNECTION_INFORMATION_EX, METHOD_BUFFERED, FILE_ANY_ACCESS)
 IOCTL_USB_GET_NODE_CONNECTION_INFORMATION_EX_V2 = CTL_CODE(FILE_DEVICE_USB, USB_GET_NODE_CONNECTION_INFORMATION_EX_V2, METHOD_BUFFERED, FILE_ANY_ACCESS)
 IOCTL_USB_GET_NODE_CONNECTION_ATTRIBUTES = CTL_CODE(FILE_DEVICE_USB, USB_GET_NODE_CONNECTION_ATTRIBUTES, METHOD_BUFFERED, FILE_ANY_ACCESS)
 IOCTL_USB_GET_NODE_CONNECTION_NAME = CTL_CODE(FILE_DEVICE_USB, USB_GET_NODE_CONNECTION_NAME, METHOD_BUFFERED, FILE_ANY_ACCESS)
)

// The following are used for HID reports IOCTLs
func HID_CTL_CODE(id int) int { return CTL_CODE(FILE_DEVICE_KEYBOARD, (id), METHOD_NEITHER, FILE_ANY_ACCESS) }
func HID_BUFFER_CTL_CODE(id int) int { return CTL_CODE(FILE_DEVICE_KEYBOARD, (id), METHOD_BUFFERED, FILE_ANY_ACCESS) }
func HID_IN_CTL_CODE(id int) int { return CTL_CODE(FILE_DEVICE_KEYBOARD, (id), METHOD_IN_DIRECT, FILE_ANY_ACCESS) }
func HID_OUT_CTL_CODE(id int) int { return CTL_CODE(FILE_DEVICE_KEYBOARD, (id), METHOD_OUT_DIRECT, FILE_ANY_ACCESS) }

var (
 IOCTL_HID_GET_FEATURE	=	HID_OUT_CTL_CODE(100)
 IOCTL_HID_GET_INPUT_REPORT =	HID_OUT_CTL_CODE(104)
 IOCTL_HID_SET_FEATURE =		HID_IN_CTL_CODE(100)
 IOCTL_HID_SET_OUTPUT_REPORT =	HID_IN_CTL_CODE(101)
)


func LIBUSB_REQ_RECIPIENT(request_type int) int { return ((request_type) & 0x1F) }
func LIBUSB_REQ_TYPE(request_type int) int { return	((request_type) & (0x03 << 5)) }
func LIBUSB_REQ_IN(request_type int) int { return ((request_type) & LIBUSB_ENDPOINT_IN) }
func LIBUSB_REQ_OUT(request_type int) int { return (!LIBUSB_REQ_IN(request_type)) }

func _device_handle_priv(handle *libusb_device_handle) *windows_device_handle_priv {
	return handle.os_priv
}

func _device_priv(dev *libusb_device) *windows_device_priv {
	return dev.os_priv
}

func windows_device_priv_init(dev *libusb_device) *windows_device_priv {
	p := _device_priv(dev)
	p.apib = &usb_api_backend[USB_API_UNSUPPORTED]
	p.sub_api = SUB_API_NOTSET
	for (i := 0; i < USB_MAXINTERFACES; i++) {
		p.usb_interface[i].apib = &usb_api_backend[USB_API_UNSUPPORTED]
		p.usb_interface[i].sub_api = SUB_API_NOTSET
	}
	return p
}

type HIDP_VALUE_CAPS struct {
	UsagePage USAGE
	ReportID uint8
	IsAlias bool
	BitField uint16
	LinkCollection uint16
	LinkUsage USAGE
	LinkUsagePage USAGE
	IsRange bool
	IsStringRange bool
	IsDesignatorRange bool
	IsAbsolute bool
	HasNull bool
	Reserved uint8
	BitSize uint16
	ReportCount uint16
	Reserved2 [5]uint16
	UnitsExp uint64
	Units uint64
	LogicalMin, LogicalMax int32
	PhysicalMin, PhysicalMax int32
	u Range 
}

type Range struct {
	UsageMin, UsageMax USAGE
	StringMin, StringMax uint16
	DesignatorMin, DesignatorMax uint16
	DataIndexMin, DataIndexMax uint16
}

type USB_HUB_NAME_FIXED struct {
	root_or_node USB_ROOT_HUB_NAME_FIXED
}

type USB_PROTOCOLS struct {
	Usb110 bool
	Usb200 bool
	Usb300 bool 
	reservedMBZ [29]bool
}

type USB_NODE_CONNECTION_INFORMATION_EX_V2_FLAGS struct {
	DeviceIsOperatingAtSuperSpeedOrHigher bool
	DeviceIsSuperSpeedCapableOrHigher bool
	reservedMBZ [30]bool
}