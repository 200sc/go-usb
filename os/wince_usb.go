package os

/*
 * Windows CE backend for libusb 1.0
 * Copyright © 2011-2013 RealVNC Ltd.
 * Portions taken from Windows backend, which is
 * Copyright © 2009-2010 Pete Batard <pbatard@gmail.com>
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

const (
	MAX_DEVICE_COUNT = 256
	// Collection of flags which can be used when issuing transfer requests
	/* Indicates that the transfer direction is 'in' */
	UKW_TF_IN_TRANSFER = 0x00000001
	/* Indicates that the transfer direction is 'out' */
	UKW_TF_OUT_TRANSFER = 0x00000000
	/* Specifies that the transfer should complete as soon as possible,
	 * even if no OVERLAPPED structure has been provided. */
	UKW_TF_NO_WAIT = 0x00000100
	/* Indicates that transfers shorter than the buffer are ok */
	UKW_TF_SHORT_TRANSFER_OK = 0x00000200
	UKW_TF_SEND_TO_DEVICE    = 0x00010000
	UKW_TF_SEND_TO_INTERFACE = 0x00020000
	UKW_TF_SEND_TO_ENDPOINT  = 0x00040000
	/* Don't block when waiting for memory allocations */
	UKW_TF_DONT_BLOCK_FOR_MEM = 0x00080000

	/* Value to use when dealing with configuration values, such as UkwGetConfigDescriptor,
	 * to specify the currently active configuration for the device. */
	UKW_ACTIVE_CONFIGURATION = -1

	// Used to determine if an endpoint status really is halted on a failed transfer.
	STATUS_HALT_FLAG = 0x1
)

// This is a modified dump of the types in the ceusbkwrapper.h library header
// with functions transformed into extern pointers.
//
// This backend dynamically loads ceusbkwrapper.dll and doesn't include
// ceusbkwrapper.h directly to simplify the build process. The kernel
// side wrapper driver is built using the platform image build tools,
// which makes it difficult to reference directly from the libusb build
// system.

type UKW_DEVICE_DESCRIPTOR struct {
	bLength            uint8
	bDescriptorType    uint8
	bcdUSB             uint16
	bDeviceClass       uint8
	bDeviceSubClass    uint8
	bDeviceProtocol    uint8
	bMaxPacketSize0    uint8
	idVendor           uint16
	idProduct          uint16
	bcdDevice          uint16
	iManufacturer      uint8
	iProduct           uint8
	iSerialNumber      uint8
	bNumConfigurations uint8
}

type UKW_CONTROL_HEADER struct {
	bmRequestType uint8
	bRequest      uint8
	wValue        uint16
	wIndex        uint16
	wLength       uint16
}

type wince_device_priv struct {
	dev  UKW_DEVICE
	desc UKW_DEVICE_DESCRIPTOR
}

type wince_transfer_priv struct {
	pollable_fd      winfd
	interface_number uint8
}
