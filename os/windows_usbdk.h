typedef struct tag_USB_DK_DEVICE_ID {
	WCHAR DeviceID[MAX_DEVICE_ID_LEN];
	WCHAR InstanceID[MAX_DEVICE_ID_LEN];
} USB_DK_DEVICE_ID, *PUSB_DK_DEVICE_ID;

static  void UsbDkFillIDStruct(USB_DK_DEVICE_ID *ID, PCWCHAR DeviceID, PCWCHAR InstanceID)
{
	wcsncpy_s(ID->DeviceID, DeviceID, MAX_DEVICE_ID_LEN);
	wcsncpy_s(ID->InstanceID, InstanceID, MAX_DEVICE_ID_LEN);
}

typedef struct tag_USB_DK_DEVICE_INFO {
	USB_DK_DEVICE_ID ID;
	ULONG64 FilterID;
	ULONG64 Port;
	ULONG64 Speed;
	USB_DEVICE_DESCRIPTOR DeviceDescriptor;
} USB_DK_DEVICE_INFO, *PUSB_DK_DEVICE_INFO;

typedef struct tag_USB_DK_CONFIG_DESCRIPTOR_REQUEST {
	USB_DK_DEVICE_ID ID;
	ULONG64 Index;
} USB_DK_CONFIG_DESCRIPTOR_REQUEST, *PUSB_DK_CONFIG_DESCRIPTOR_REQUEST;

typedef struct tag_USB_DK_ISO_TRANSFER_RESULT {
	ULONG64 ActualLength;
	ULONG64 TransferResult;
} USB_DK_ISO_TRANSFER_RESULT, *PUSB_DK_ISO_TRANSFER_RESULT;

typedef struct tag_USB_DK_GEN_TRANSFER_RESULT {
	ULONG64 BytesTransferred;
	ULONG64 UsbdStatus; // USBD_STATUS code
} USB_DK_GEN_TRANSFER_RESULT, *PUSB_DK_GEN_TRANSFER_RESULT;

typedef struct tag_USB_DK_TRANSFER_RESULT {
	USB_DK_GEN_TRANSFER_RESULT GenResult;
	PVOID64 IsochronousResultsArray; // array of USB_DK_ISO_TRANSFER_RESULT
} USB_DK_TRANSFER_RESULT, *PUSB_DK_TRANSFER_RESULT;

typedef struct tag_USB_DK_TRANSFER_REQUEST {
	ULONG64 EndpointAddress;
	PVOID64 Buffer;
	ULONG64 BufferLength;
	ULONG64 TransferType;
	ULONG64 IsochronousPacketsArraySize;
	PVOID64 IsochronousPacketsArray;

	USB_DK_TRANSFER_RESULT Result;
} USB_DK_TRANSFER_REQUEST, *PUSB_DK_TRANSFER_REQUEST;

typedef BOOL (__cdecl *USBDK_GET_DEVICES_LIST)(
	PUSB_DK_DEVICE_INFO *DeviceInfo,
	PULONG DeviceNumber
);
typedef void (__cdecl *USBDK_RELEASE_DEVICES_LIST)(
	PUSB_DK_DEVICE_INFO DeviceInfo
);
typedef HANDLE (__cdecl *USBDK_START_REDIRECT)(
	PUSB_DK_DEVICE_ID DeviceId
);
typedef BOOL (__cdecl *USBDK_STOP_REDIRECT)(
	HANDLE DeviceHandle
);
typedef BOOL (__cdecl *USBDK_GET_CONFIGURATION_DESCRIPTOR)(
	PUSB_DK_CONFIG_DESCRIPTOR_REQUEST Request,
	PUSB_CONFIGURATION_DESCRIPTOR *Descriptor,
	PULONG Length
);
typedef void (__cdecl *USBDK_RELEASE_CONFIGURATION_DESCRIPTOR)(
	PUSB_CONFIGURATION_DESCRIPTOR Descriptor
);
typedef TransferResult (__cdecl *USBDK_WRITE_PIPE)(
	HANDLE DeviceHandle,
	PUSB_DK_TRANSFER_REQUEST Request,
	LPOVERLAPPED lpOverlapped
);
typedef TransferResult (__cdecl *USBDK_READ_PIPE)(
	HANDLE DeviceHandle,
	PUSB_DK_TRANSFER_REQUEST Request,
	LPOVERLAPPED lpOverlapped
);
typedef BOOL (__cdecl *USBDK_ABORT_PIPE)(
	HANDLE DeviceHandle,
	ULONG64 PipeAddress
);
typedef BOOL (__cdecl *USBDK_RESET_PIPE)(
	HANDLE DeviceHandle,
	ULONG64 PipeAddress
);
typedef BOOL (__cdecl *USBDK_SET_ALTSETTING)(
	HANDLE DeviceHandle,
	ULONG64 InterfaceIdx,
	ULONG64 AltSettingIdx
);
typedef BOOL (__cdecl *USBDK_RESET_DEVICE)(
	HANDLE DeviceHandle
);
typedef HANDLE (__cdecl *USBDK_GET_REDIRECTOR_SYSTEM_HANDLE)(
	HANDLE DeviceHandle
);
