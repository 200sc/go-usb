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