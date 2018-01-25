# go-usb
An in-progress port of libusb to Go

# Current state

OS-Agnostic code is converted and needs to be passed over for compile errors:

Syntax: 
* [x] backend.go
* [x] core.go
* [x] descriptor.go
* [x] hotplug.go
* [x] io.go
* [x] io_unix.go
* [x] io_windows.go
* [x] libusb.go
* [x] libusbi.go
* [x] list.go
* [x] strerror.go
* [x] sync.go

Semantic:
* [x] backend.go
* [x] core.go
* [ ] descriptor.go
* [ ] hotplug.go
* [ ] io.go
* [ ] io_unix.go
* [ ] io_windows.go
* [ ] libusb.go
* [ ] libusbi.go
* [ ] list.go
* [ ] strerror.go
* [ ] sync.go

OS-Specific code is in the process of conversion

All of the small OS-specific files have been converted, the smallest remaining file at over 600 LOC. 

Before I start converting the larger files I want to look into whether the `poll` semantics that libusb uses are even a thing we need to care about, or if they can be replicated with channels. All references to `poll` in the libusb docs suggest that they are "fake" and just used for signalling, which sounds a heck of a lot like a `chan struct{}`.
