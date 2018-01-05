#define MAX_DEVICE_COUNT            256

// This is a modified dump of the types in the ceusbkwrapper.h library header
// with functions transformed into extern pointers.
//
// This backend dynamically loads ceusbkwrapper.dll and doesn't include
// ceusbkwrapper.h directly to simplify the build process. The kernel
// side wrapper driver is built using the platform image build tools,
// which makes it difficult to reference directly from the libusb build
// system.
struct UKW_DEVICE_PRIV;
typedef struct UKW_DEVICE_PRIV *UKW_DEVICE;
typedef UKW_DEVICE *PUKW_DEVICE, *LPUKW_DEVICE;

typedef struct {
	UINT8 bLength;
	UINT8 bDescriptorType;
	UINT16 bcdUSB;
	UINT8 bDeviceClass;
	UINT8 bDeviceSubClass;
	UINT8 bDeviceProtocol;
	UINT8 bMaxPacketSize0;
	UINT16 idVendor;
	UINT16 idProduct;
	UINT16 bcdDevice;
	UINT8 iManufacturer;
	UINT8 iProduct;
	UINT8 iSerialNumber;
	UINT8 bNumConfigurations;
} UKW_DEVICE_DESCRIPTOR, *PUKW_DEVICE_DESCRIPTOR, *LPUKW_DEVICE_DESCRIPTOR;

typedef struct {
	UINT8 bmRequestType;
	UINT8 bRequest;
	UINT16 wValue;
	UINT16 wIndex;
	UINT16 wLength;
} UKW_CONTROL_HEADER, *PUKW_CONTROL_HEADER, *LPUKW_CONTROL_HEADER;

// Collection of flags which can be used when issuing transfer requests
/* Indicates that the transfer direction is 'in' */
#define UKW_TF_IN_TRANSFER        0x00000001
/* Indicates that the transfer direction is 'out' */
#define UKW_TF_OUT_TRANSFER       0x00000000
/* Specifies that the transfer should complete as soon as possible,
 * even if no OVERLAPPED structure has been provided. */
#define UKW_TF_NO_WAIT            0x00000100
/* Indicates that transfers shorter than the buffer are ok */
#define UKW_TF_SHORT_TRANSFER_OK  0x00000200
#define UKW_TF_SEND_TO_DEVICE     0x00010000
#define UKW_TF_SEND_TO_INTERFACE  0x00020000
#define UKW_TF_SEND_TO_ENDPOINT   0x00040000
/* Don't block when waiting for memory allocations */
#define UKW_TF_DONT_BLOCK_FOR_MEM 0x00080000

/* Value to use when dealing with configuration values, such as UkwGetConfigDescriptor, 
 * to specify the currently active configuration for the device. */
#define UKW_ACTIVE_CONFIGURATION -1

// Used to determine if an endpoint status really is halted on a failed transfer.
#define STATUS_HALT_FLAG 0x1

struct wince_device_priv {
	UKW_DEVICE dev;
	UKW_DEVICE_DESCRIPTOR desc;
};

struct wince_transfer_priv {
	struct winfd pollable_fd;
	uint8 interface_number;
};

