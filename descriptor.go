package usb

/*
 * USB descriptor handling functions for libusb
 * Copyright © 2007 Daniel Drake <dsd@gentoo.org>
 * Copyright © 2001 Johannes Erdfelt <johannes@erdfelt.com>
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

const DESC_HEADER_LENGTH = 2
const DEVICE_DESC_LENGTH = 18
const CONFIG_DESC_LENGTH = 9
const INTERFACE_DESC_LENGTH = 9
const ENDPOINT_DESC_LENGTH = 7
const ENDPOINT_AUDIO_DESC_LENGTH = 9

/** \ingroup libusb_desc
 * Get a Container ID descriptor
 *
 * \param ctx the context to operate on, or nil for the default context
 * \param dev_cap Device Capability descriptor with a bDevCapabilityType of
 * \ref libusb_capability_type::LIBUSB_BT_CONTAINER_ID
 * LIBUSB_BT_CONTAINER_ID
 * \param container_id output location for the Container ID descriptor.
 * Only valid if 0 was returned. Must be freed with
 * libusb_free_container_id_descriptor() after use.
 * \returns 0 on success
 * \returns a LIBUSB_ERROR code on error
 */
func libusb_get_container_id_descriptor(ctx *libusb_context,
	dev_cap *libusb_bos_dev_capability_descriptor,
	container_id **libusb_container_id_descriptor) int {

	_container_id * libusb_container_id_descriptor
	const int host_endian = 0

	if dev_cap.bDevCapabilityType != LIBUSB_BT_CONTAINER_ID {
		// usbi_err(ctx, "unexpected bDevCapabilityType %x (expected %x)",
		//  dev_cap.bDevCapabilityType,
		//  LIBUSB_BT_CONTAINER_ID);
		return LIBUSB_ERROR_INVALID_PARAM
	}
	if dev_cap.bLength < LIBUSB_BT_CONTAINER_ID_SIZE {
		// usbi_err(ctx, "short dev-cap descriptor read %d/%d",
		//  dev_cap.bLength, LIBUSB_BT_CONTAINER_ID_SIZE);
		return LIBUSB_ERROR_IO
	}

	idDest := &libusb_container_id_descriptor{}

	usbi_parse_descriptor([]uint8(dev_cap), "bbbbu", idDest, host_endian)

	*container_id = idDest
	return LIBUSB_SUCCESS
}

/** \ingroup libusb_desc
 * Retrieve a string descriptor in C style ASCII.
 *
 * Wrapper around libusb_get_string_descriptor(). Uses the first language
 * supported by the device.
 *
 * \param dev_handle a device handle
 * \param desc_index the index of the descriptor to retrieve
 * \param data output buffer for ASCII string descriptor
 * \param length size of data buffer
 * \returns number of bytes returned in data, or LIBUSB_ERROR code on failure
 */
func libusb_get_string_descriptor_ascii(dev_handle *libusb_device_handle, desc_index uint8, data []uint8, length int) int {

	/* Asking for the zero'th index is special - it returns a string
	 * descriptor that contains all the language IDs supported by the
	 * device. Typically there aren't many - often only one. Language
	 * IDs are 16 bit numbers, and they start at the third byte in the
	 * descriptor. There's also no point in trying to read descriptor 0
	 * with this function. See USB 2.0 specification section 9.6.7 for
	 * more information.
	 */

	if desc_index == 0 {
		return LIBUSB_ERROR_INVALID_PARAM
	}

	var tbuff [255]uint8 /* Some devices choke on size > 255 */

	r := libusb_get_string_descriptor(dev_handle, 0, 0, tbuf, len(tbuff))
	if r < 0 {
		return r
	}

	if r < 4 {
		return LIBUSB_ERROR_IO
	}

	var langid uint16
	langid = tbuf[2] | (tbuf[3] << 8)

	r = libusb_get_string_descriptor(dev_handle, desc_index, langid, tbuf, len(tbuf))
	if r < 0 {
		return r
	}

	if tbuf[1] != LIBUSB_DT_STRING {
		return LIBUSB_ERROR_IO
	}

	if tbuf[0] > r {
		return LIBUSB_ERROR_IO
	}

	di := 0
	for si := 2; si < tbuf[0]; si += 2 {
		if di >= (length - 1) {
			break
		}

		if (tbuf[si]&0x80) != 0 || (tbuf[si+1]) != 0 { /* non-ASCII */
			data[di] = '?'
			di++
		} else {
			data[di] = tbuf[si]
			di++
		}
	}

	data[di] = 0
	return di
}

/** \ingroup libusb_desc
 * Get a SuperSpeed USB Device Capability descriptor
 *
 * \param ctx the context to operate on, or nil for the default context
 * \param dev_cap Device Capability descriptor with a bDevCapabilityType of
 * \ref libusb_capability_type::LIBUSB_BT_SS_USB_DEVICE_CAPABILITY
 * LIBUSB_BT_SS_USB_DEVICE_CAPABILITY
 * \param ss_usb_device_cap output location for the SuperSpeed USB Device
 * Capability descriptor. Only valid if 0 was returned. Must be freed with
 * libusb_free_ss_usb_device_capability_descriptor() after use.
 * \returns 0 on success
 * \returns a LIBUSB_ERROR code on error
 */
func libusb_get_ss_usb_device_capability_descriptor(
	ctx *libusb_context,
	dev_cap *libusb_bos_dev_capability_descriptor,
	ss_usb_device_cap **libusb_ss_usb_device_capability_descriptor) int {

	if dev_cap.bDevCapabilityType != LIBUSB_BT_SS_USB_DEVICE_CAPABILITY {
		// usbi_err(ctx, "unexpected bDevCapabilityType %x (expected %x)",
		//  dev_cap.bDevCapabilityType,
		//  LIBUSB_BT_SS_USB_DEVICE_CAPABILITY);
		return LIBUSB_ERROR_INVALID_PARAM
	}
	if dev_cap.bLength < LIBUSB_BT_SS_USB_DEVICE_CAPABILITY_SIZE {
		// usbi_err(ctx, "short dev-cap descriptor read %d/%d",
		//  dev_cap.bLength, LIBUSB_BT_SS_USB_DEVICE_CAPABILITY_SIZE);
		return LIBUSB_ERROR_IO
	}

	capDest := &libusb_ss_device_capability_descriptor{}

	usbi_parse_descriptor([]uint8(dev_cap), "bbbbwbbw", capDest, 0)

	*ss_usb_device_cap = capDest
	return LIBUSB_SUCCESS
}

/** \ingroup libusb_desc
 * Get an USB 2.0 Extension descriptor
 *
 * \param ctx the context to operate on, or nil for the default context
 * \param dev_cap Device Capability descriptor with a bDevCapabilityType of
 * \ref libusb_capability_type::LIBUSB_BT_USB_2_0_EXTENSION
 * LIBUSB_BT_USB_2_0_EXTENSION
 * \param usb_2_0_extension output location for the USB 2.0 Extension
 * descriptor. Only valid if 0 was returned. Must be freed with
 * libusb_free_usb_2_0_extension_descriptor() after use.
 * \returns 0 on success
 * \returns a LIBUSB_ERROR code on error
 */
func libusb_get_usb_2_0_extension_descriptor(
	ctx *libusb_context,
	dev_cap *libusb_bos_dev_capability_descriptor,
	usb_2_0_extension **libusb_usb_2_0_extension_descriptor) int {

	if dev_cap.bDevCapabilityType != LIBUSB_BT_USB_2_0_EXTENSION {
		// usbi_err(ctx, "unexpected bDevCapabilityType %x (expected %x)",
		//  dev_cap.bDevCapabilityType,
		//  LIBUSB_BT_USB_2_0_EXTENSION);
		return LIBUSB_ERROR_INVALID_PARAM
	}
	if dev_cap.bLength < LIBUSB_BT_USB_2_0_EXTENSION_SIZE {
		// usbi_err(ctx, "short dev-cap descriptor read %d/%d",
		//  dev_cap.bLength, LIBUSB_BT_USB_2_0_EXTENSION_SIZE)
		return LIBUSB_ERROR_IO
	}

	extDest := &libusb_usb_2_0_extension_descriptor{}

	usbi_parse_descriptor([]uint8(dev_cap), "bbbd", extDest, 0)

	*usb_2_0_extension = extDest
	return LIBUSB_SUCCESS
}

/** \ingroup libusb_desc
 * Get a Binary Object Store (BOS) descriptor
 * This is a BLOCKING function, which will send requests to the device.
 *
 * \param dev_handle the handle of an open libusb device
 * \param bos output location for the BOS descriptor. Only valid if 0 was returned.
 * Must be freed with \ref libusb_free_bos_descriptor() after use.
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the device doesn't have a BOS descriptor
 * \returns another LIBUSB_ERROR code on error
 */
func libusb_get_bos_descriptor(dev_handle *libusb_device_handle, bos **libusb_bos_descriptor) int {

	bos_header := make([]uint8, LIBUSB_DT_BOS_SIZE)

	/* Read the BOS. This generates 2 requests on the bus,
	 * one for the header, and one for the full BOS */
	r := libusb_get_descriptor(dev_handle, LIBUSB_DT_BOS, 0, bos_header, LIBUSB_DT_BOS_SIZE)
	if r < 0 {
		// if (r != LIBUSB_ERROR_PIPE)
		// usbi_err(dev_handle), "failed to read BOS (%d)", r.dev.ctx;
		return r
	}
	if r < LIBUSB_DT_BOS_SIZE {
		// usbi_err(dev_handle.dev.ctx, "short BOS read %d/%d",
		//  r, LIBUSB_DT_BOS_SIZE)
		return LIBUSB_ERROR_IO
	}

	_bos := libusb_bos_descriptor{}
	host_endian := 0

	usbi_parse_descriptor(bos_header, "bbwb", &_bos, host_endian)
	// usbi_dbg("found BOS descriptor: size %d bytes, %d capabilities",
	//  _bos.wTotalLength, _bos.bNumDeviceCaps);
	bos_data := make([]uint8, _bos.wTotalLength)

	r = libusb_get_descriptor(dev_handle, LIBUSB_DT_BOS, 0, bos_data, _bos.wTotalLength)
	if r >= 0 {
		r = parse_bos(dev_handle.dev.ctx, bos, bos_data, r, host_endian)
	}
	// else usbi_err(dev_handle), "failed to read BOS (%d)", r.dev.ctx;

	return r
}
