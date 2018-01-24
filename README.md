# go-usb
An in-progress port of libusb to Go

# Current state

OS-Agnostic code is converted and needs to be passed over for compile errors:

* [x] backend.go
* [x] core.go
* [x] descriptor.go
* [x] hotplug.go
* [x] io.go
* [ ] io_unix.go
* [ ] io_windows.go
* [ ] libusb.go
* [ ] libusbi.go
* [ ] list.go
* [ ] strerror.go
* [ ] sync.go

OS-Specific code is in the process of conversion
