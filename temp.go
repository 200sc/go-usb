package usb

// These need to be defined by the os package, but are defined here for the time being

var usbi_backend usbi_os_backend

var usbi_write func(int, interface{}, int) libusb_error
var usbi_read func(int, interface{}, int) libusb_error
