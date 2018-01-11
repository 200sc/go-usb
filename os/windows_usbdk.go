package os

/*
* windows UsbDk backend for libusb 1.0
* Copyright © 2014 Red Hat, Inc.

* Authors:
* Dmitry Fleytman <dmitry@daynix.com>
* Pavel Gurvich <pavel@daynix.com>
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

type TransferResult uint8
const(
	TransferFailure TransferResult = iota
	TransferSuccess TransferResult
	TransferSuccessAsync TransferResult
)

type USB_DK_DEVICE_SPEED
const(
	NoSpeed USB_DK_DEVICE_SPEED = iota
	LowSpeed USB_DK_DEVICE_SPEED
	FullSpeed USB_DK_DEVICE_SPEED
	HighSpeed USB_DK_DEVICE_SPEED
	SuperSpee USB_DK_DEVICE_SPEED
)

type USB_DK_TRANSFER_TYPE
const(
	ControlTransferType USB_DK_TRANSFER_TYPE = iota
	BulkTransferType USB_DK_TRANSFER_TYPE
	IntertuptTransferType USB_DK_TRANSFER_TYPE
	IsochronousTransferType USB_DK_TRANSFER_TYPE
)

func UsbDkFillIDStruct(ID *USB_DK_DEVICE_ID, DeviceID PCWCHAR, InstanceID PCWCHAR) {
	wcsncpy_s(ID.DeviceID, DeviceID, MAX_DEVICE_ID_LEN)
	wcsncpy_s(ID.InstanceID, InstanceID, MAX_DEVICE_ID_LEN)
}

type USB_DK_DEVICE_ID struct {
	DeviceID, InstanceID string 
}

type USB_DK_DEVICE_INFO struct {
	ID USB_DK_DEVICE_ID
	FilterID, Port, Speed uint64
	DeviceDescriptor USB_DEVICE_DESCRIPTOR
}

type USB_DK_CONFIG_DESCRIPTOR_REQUEST struct {
	ID USB_DK_DEVICE_ID
	Index uint64
}

type USB_DK_ISO_TRANSFER_RESULT struct  {
	ActualLength, TransferResult uint64
}

type USB_DK_GEN_TRANSFER_RESULT struct {
	BytesTransferred, UsbdStatus uint64
}

type USB_DK_TRANSFER_RESULT struct {
	USB_DK_GEN_TRANSFER_RESULT GenResult
	IsochronousResultsArray interface{} //[]USB_DK_ISO_TRANSFER_RESULT
}

type USB_DK_TRANSFER_REQUEST struct  {
	uint64 EndpointAddress
	Buffer []byte
	uint64 BufferLength
	uint64 TransferType
	uint64 IsochronousPacketsArraySize
	IsochronousPacketsArray interface{}
	Result USB_DK_TRANSFER_RESULT
}

type USBDK_GET_DEVICES_LIST func(*PUSB_DK_DEVICE_INFO, *uint32) bool
type USBDK_RELEASE_DEVICES_LIST func(PUSB_DK_DEVICE_INFO)
type USBDK_START_REDIRECT func(PUSB_DK_DEVICE_ID) HANDLE
type USBDK_STOP_REDIRECT func(HANDLE) bool
type USBDK_GET_CONFIGURATION_DESCRIPTOR func(PUSB_DK_CONFIG_DESCRIPTOR_REQUEST, PUSB_CONFIGURATION_DESCRIPTOR, *uint32) bool
type USBDK_RELEASE_CONFIGURATION_DESCRIPTOR func(PUSB_CONFIGURATION_DESCRIPTOR)
type USBDK_WRITE_PIPE USBDKUsePipe
type USBDK_READ_PIPE USBDKUsePipe
type USBDK_ABORT_PIPE USBDKStopPipe
type USBDK_RESET_PIPE USBDKStopPipe
type USBDK_SET_ALTSETTING func(HANDLE, uint64, uint64) bool
type USBDK_RESET_DEVICE func(HANDLE) bool
type USBDK_GET_REDIRECTOR_SYSTEM_HANDLE func(HANDLE) HANDLE

type USBDKUsePipe func(HANDLE, PUSB_DK_TRANSFER_REQUEST, LPOVERLAPPED) TransferResult
type USBDKStopPipe func(HANDLE, uint16) bool