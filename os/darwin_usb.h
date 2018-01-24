// Go todo: ????
/* IOUSBInterfaceInferface */
#if defined (kIOUSBInterfaceInterfaceID700) && MAC_OS_X_VERSION_MIN_REQUIRED >= MAC_OS_X_VERSION_10_9

#define usb_interface_t IOUSBInterfaceInterface700
#define InterfaceInterfaceID kIOUSBInterfaceInterfaceID700
#define InterfaceVersion 700

#elif defined (kIOUSBInterfaceInterfaceID550) && MAC_OS_X_VERSION_MIN_REQUIRED >= MAC_OS_X_VERSION_10_9

#define usb_interface_t IOUSBInterfaceInterface550
#define InterfaceInterfaceID kIOUSBInterfaceInterfaceID550
#define InterfaceVersion 550

#elif defined (kIOUSBInterfaceInterfaceID500)

#define usb_interface_t IOUSBInterfaceInterface500
#define InterfaceInterfaceID kIOUSBInterfaceInterfaceID500
#define InterfaceVersion 500

#elif defined (kIOUSBInterfaceInterfaceID300)

#define usb_interface_t IOUSBInterfaceInterface300
#define InterfaceInterfaceID kIOUSBInterfaceInterfaceID300
#define InterfaceVersion 300

#elif defined (kIOUSBInterfaceInterfaceID245)

#define usb_interface_t IOUSBInterfaceInterface245
#define InterfaceInterfaceID kIOUSBInterfaceInterfaceID245
#define InterfaceVersion 245

#elif defined (kIOUSBInterfaceInterfaceID220)

#define usb_interface_t IOUSBInterfaceInterface220
#define InterfaceInterfaceID kIOUSBInterfaceInterfaceID220
#define InterfaceVersion 220

#else

#error "IOUSBFamily is too old. Please upgrade your OS"

#endif

/* IOUSBDeviceInterface */
#if defined (kIOUSBDeviceInterfaceID500) && MAC_OS_X_VERSION_MIN_REQUIRED >= MAC_OS_X_VERSION_10_9

#define usb_device_t    IOUSBDeviceInterface500
#define DeviceInterfaceID kIOUSBDeviceInterfaceID500
#define DeviceVersion 500

#elif defined (kIOUSBDeviceInterfaceID320)

#define usb_device_t    IOUSBDeviceInterface320
#define DeviceInterfaceID kIOUSBDeviceInterfaceID320
#define DeviceVersion 320

#elif defined (kIOUSBDeviceInterfaceID300)

#define usb_device_t    IOUSBDeviceInterface300
#define DeviceInterfaceID kIOUSBDeviceInterfaceID300
#define DeviceVersion 300

#elif defined (kIOUSBDeviceInterfaceID245)

#define usb_device_t    IOUSBDeviceInterface245
#define DeviceInterfaceID kIOUSBDeviceInterfaceID245
#define DeviceVersion 245

#elif defined (kIOUSBDeviceInterfaceID220)
#define usb_device_t    IOUSBDeviceInterface197
#define DeviceInterfaceID kIOUSBDeviceInterfaceID197
#define DeviceVersion 197

#else

#error "IOUSBFamily is too old. Please upgrade your OS"

#endif
// end go todo ????