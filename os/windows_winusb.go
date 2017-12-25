package os

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