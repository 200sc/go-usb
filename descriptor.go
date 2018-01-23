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
	host_endian := false

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
	host_endian := false

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

/** \ingroup libusb_desc
 * Get an endpoints superspeed endpoint companion descriptor (if any)
 *
 * \param ctx the context to operate on, or nil for the default context
 * \param endpoint endpoint descriptor from which to get the superspeed
 * endpoint companion descriptor
 * \param ep_comp output location for the superspeed endpoint companion
 * descriptor. Only valid if 0 was returned. Must be freed with
 * libusb_free_ss_endpoint_companion_descriptor() after use.
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the configuration does not exist
 * \returns another LIBUSB_ERROR code on error
 */
 func libusb_get_ss_endpoint_companion_descriptor(
	ctx *libusb_context,
	endpoint *libusb_endpoint_descriptor,
	ep_comp **libusb_ss_endpoint_companion_descriptor) int {

	var header usb_descriptor_header
	buffer := endpoint.extra
	buffi := 0
	size := len(buffer)	

	*ep_comp = nil

	for size >= DESC_HEADER_LENGTH {
		usbi_parse_descriptor(buffer[buffi:], "bb", &header, 0)
		if (header.bLength < 2 || header.bLength > size) {
			// usbi_err(ctx, "invalid descriptor length %d",
				//  header.bLength);
			return LIBUSB_ERROR_IO
		}
		if (header.bDescriptorType != LIBUSB_DT_SS_ENDPOINT_COMPANION) {
			buffi += header.bLength
			size -= header.bLength
			continue
		}
		if (header.bLength < LIBUSB_DT_SS_ENDPOINT_COMPANION_SIZE) {
			// usbi_err(ctx, "invalid ss-ep-comp-desc length %d",
				//  header.bLength);
			return LIBUSB_ERROR_IO
		}
		*ep_comp = &&libusb_ss_endpoint_companion_descriptor{}

		usbi_parse_descriptor(buffer, "bbbbw", *ep_comp, 0)
		return LIBUSB_SUCCESS
	}
	return LIBUSB_ERROR_NOT_FOUND
}


/** \ingroup libusb_desc
 * Get a USB configuration descriptor with a specific bConfigurationValue.
 * This is a non-blocking function which does not involve any requests being
 * sent to the device.
 *
 * \param dev a device
 * \param bConfigurationValue the bConfigurationValue of the configuration you
 * wish to retrieve
 * \param config output location for the USB configuration descriptor. Only
 * valid if 0 was returned. Must be freed with libusb_free_config_descriptor()
 * after use.
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the configuration does not exist
 * \returns another LIBUSB_ERROR code on error
 * \see libusb_get_active_config_descriptor()
 * \see libusb_get_config_descriptor()
 */
 func libusb_get_config_descriptor_by_value(dev *libusb_device,
	bConfigurationValue uint8, config **libusb_config_descriptor) int {

	var r, idx int
	host_endian := false
	var buf []uint8

	if usbi_backend.get_config_descriptor_by_value {
		r = usbi_backend.get_config_descriptor_by_value(dev, bConfigurationValue, &buf, &host_endian)
		if r < 0 {
			return r
		}
		return raw_desc_to_config(dev.ctx, buf, r, host_endian, config)
	}

	r = usbi_get_config_index_by_value(dev, bConfigurationValue, &idx)
	if r < 0 {
		return r
	} else if idx < 0 {
		return LIBUSB_ERROR_NOT_FOUND
	} else {
		return libusb_get_config_descriptor(dev, uint8(idx), config)
	}
}

/* iterate through all configurations, returning the index of the configuration
 * matching a specific bConfigurationValue in the idx output parameter, or -1
 * if the config was not found.
 * returns 0 on success or a LIBUSB_ERROR code
 */
 func usbi_get_config_index_by_value(dev *libusb_device, bConfigurationValue uint8, idx *int) int {

	// usbi_dbg("value %d", bConfigurationValue);
	for i := 0; i < dev.num_configurations; i++ {

		var tmp [6]uint8

		host_endian := false
		r := usbi_backend.get_config_descriptor(dev, i, tmp, sizeof(tmp), &host_endian)
		if r < 0 {
			*idx = -1
			return r
		}
		if tmp[5] == bConfigurationValue {
			*idx = i
			return 0
		}
	}

	*idx = -1
	return 0
}

/** \ingroup libusb_desc
 * Get a USB configuration descriptor based on its index.
 * This is a non-blocking function which does not involve any requests being
 * sent to the device.
 *
 * \param dev a device
 * \param config_index the index of the configuration you wish to retrieve
 * \param config output location for the USB configuration descriptor. Only
 * valid if 0 was returned. Must be freed with libusb_free_config_descriptor()
 * after use.
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the configuration does not exist
 * \returns another LIBUSB_ERROR code on error
 * \see libusb_get_active_config_descriptor()
 * \see libusb_get_config_descriptor_by_value()
 */
 func libusb_get_config_descriptor(dev *libusb_device, config_index uint8, config **libusb_config_descriptor) int {

	var tmp [LIBUSB_DT_CONIFG_SIZE]uint8 
	
	host_endian := false

	// usbi_dbg("index %d", config_index);
	if config_index >= dev.num_configurations {
		return LIBUSB_ERROR_NOT_FOUND
	}

	r := usbi_backend.get_config_descriptor(dev, config_index, tmp, LIBUSB_DT_CONFIG_SIZE, &host_endian)
	if r < 0 {
		return r
	}
	if r < LIBUSB_DT_CONFIG_SIZE {
		// usbi_err(dev.ctx, "short config descriptor read %d/%d",
			//  r, LIBUSB_DT_CONFIG_SIZE);
		return LIBUSB_ERROR_IO
	}

	_config := libusb_config_descriptor{}

	usbi_parse_descriptor(tmp, "bbw", &_config, host_endian)

	buf := make([]uint8, _config.wTotalLength)
	
	r = usbi_backend.get_config_descriptor(dev, config_index, buf, _config.wTotalLength, &host_endian)
	if r >= 0 {
		r = raw_desc_to_config(dev.ctx, buf, r, host_endian, config)
	}

	return r
}

/** \ingroup libusb_desc
 * Get the USB configuration descriptor for the currently active configuration.
 * This is a non-blocking function which does not involve any requests being
 * sent to the device.
 *
 * \param dev a device
 * \param config output location for the USB configuration descriptor. Only
 * valid if 0 was returned. Must be freed with libusb_free_config_descriptor()
 * after use.
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the device is in unconfigured state
 * \returns another LIBUSB_ERROR code on error
 * \see libusb_get_config_descriptor
 */
 func libusb_get_active_config_descriptor(dev *libusb_device, config **libusb_config_descriptor) int {

	var _config libusb_config_descriptor 
	var tmp [LIBUSB_DT_CONFIG_SIZE]uint8

	host_endian := false

	r := usbi_backend.get_active_config_descriptor(dev, tmp, LIBUSB_DT_CONFIG_SIZE, &host_endian)
	if (r < 0) {
		return r
	}
	if (r < LIBUSB_DT_CONFIG_SIZE) {
		// usbi_err(dev.ctx, "short config descriptor read %d/%d",
			//  r, LIBUSB_DT_CONFIG_SIZE);
		return LIBUSB_ERROR_IO
	}

	usbi_parse_descriptor(tmp, "bbw", &_config, host_endian)
	buf := make([]uint8, _config.wTotalLength)

	r = usbi_backend.get_active_config_descriptor(dev, buf, _config.wTotalLength, &host_endian)
	if r >= 0 {
		r = raw_desc_to_config(dev.ctx, buf, r, host_endian, config)
	}

	return r
}

/** \ingroup libusb_desc
 * Get the USB device descriptor for a given device.
 *
 * This is a non-blocking function; the device descriptor is cached in memory.
 *
 * Note since libusb-1.0.16, \ref LIBUSB_API_VERSION >= 0x01000102, this
 * function always succeeds.
 *
 * \param dev the device
 * \param desc output location for the descriptor data
 * \returns 0 on success or a LIBUSB_ERROR code on failure
 */
 // GO: this was changed from using a memcpy to a new pointer to just returning the value
 func libusb_get_device_descriptor(dev *libusb_device) libusb_device_descriptor {
	return dev.device_descriptor
}

func usbi_device_cache_descriptor(dev *libusb_device) int {
	host_endian := false
	r := usbi_backend.get_device_descriptor(dev, []uint8(&dev.device_descriptor), &host_endian)
	if r < 0) {
		return r
	}

	if !host_endian {
		dev.device_descriptor.bcdUSB = libusb_le16_to_cpu(dev.device_descriptor.bcdUSB)
		dev.device_descriptor.idVendor = libusb_le16_to_cpu(dev.device_descriptor.idVendor)
		dev.device_descriptor.idProduct = libusb_le16_to_cpu(dev.device_descriptor.idProduct)
		dev.device_descriptor.bcdDevice = libusb_le16_to_cpu(dev.device_descriptor.bcdDevice)
	}

	return LIBUSB_SUCCESS
}

func raw_desc_to_config(ctx *libusb_context,
	buf []uint8, size int, host_endian bool,
	config **libusb_config_descriptor) int {

	_config := &libusb_config_descriptor{}

	r := parse_configuration(ctx, _config, buf, size, host_endian)
	if r < 0 {
		// usbi_err(ctx, "parse_configuration failed with error %d", r);
		return r
	} 
	// else if (r > 0) {
	// 	// usbi_warn(ctx, "still %d bytes of descriptor data left", r);
	// }
	
	*config = _config
	return LIBUSB_SUCCESS
}

func parse_configuration(ctx *libusb_context,
	config *libusb_config_descriptor, buffer []uint8,
	size int, host_endian bool) int {

   if (size < LIBUSB_DT_CONFIG_SIZE) {
	   // usbi_err(ctx, "short config descriptor read %d/%d",
			// size, LIBUSB_DT_CONFIG_SIZE)
	   return LIBUSB_ERROR_IO
   }

   var header usb_descriptor_header

   usbi_parse_descriptor(buffer, "bbwbbbbb", config, host_endian)
   if (config.bDescriptorType != LIBUSB_DT_CONFIG) {
	   // usbi_err(ctx, "unexpected descriptor %x (expected %x)",
			// config.bDescriptorType, LIBUSB_DT_CONFIG)
	   return LIBUSB_ERROR_IO
   }
   if (config.bLength < LIBUSB_DT_CONFIG_SIZE) {
	   // usbi_err(ctx, "invalid config bLength (%d)", config.bLength)
	   return LIBUSB_ERROR_IO
   }
   if (config.bLength > size) {
	   // usbi_err(ctx, "short config descriptor read %d/%d",
			// size, config.bLength)
	   return LIBUSB_ERROR_IO
   }
   if (config.bNumInterfaces > USB_MAXINTERFACES) {
	   // usbi_err(ctx, "too many interfaces (%d)", config.bNumInterfaces)
	   return LIBUSB_ERROR_IO
   }

   config.interface = make([]libusb_interface, config.bNumInterfaces)

   buffi := config.bLength
   size -= config.bLength

   config.extra = nil

   for (i := 0; i < config.bNumInterfaces; i++) {
	   int len
	   uint8 *begin

	   /* Skip over the rest of the Class Specific or Vendor */
	   /*  Specific descriptors */
	   begin = buffer
	   for (size >= DESC_HEADER_LENGTH) {
		   usbi_parse_descriptor(buffer[buffi:], "bb", &header, 0)

		   if (header.bLength < DESC_HEADER_LENGTH) {
			   // usbi_err(ctx,
					// "invalid extra config desc len (%d)",
					// header.bLength)
			   return LIBUSB_ERROR_IO
		   } else if (header.bLength > size) {
			   // usbi_warn(ctx,
					//  "short extra config desc read %d/%d",
					//  size, header.bLength)
			   config.bNumInterfaces = (uint8)i
			   return size
		   }

		   /* If we find another "proper" descriptor then we're done */
		   if ((header.bDescriptorType == LIBUSB_DT_ENDPOINT) ||
				   (header.bDescriptorType == LIBUSB_DT_INTERFACE) ||
				   (header.bDescriptorType == LIBUSB_DT_CONFIG) ||
				   (header.bDescriptorType == LIBUSB_DT_DEVICE))
			   break

		   // usbi_dbg("skipping descriptor 0x%x", header.bDescriptorType)
		   buffi += header.bLength
		   size -= header.bLength
	   }

	   /* Copy any unknown descriptors into a storage area for */
	   /*  drivers to later parse */
	   len = int(buffi - begin)
	   if len != 0 {
		   if len(config.extra) {
			   config.extra = make([]uint8, len)

			   copy(config.extra, begin)
		   }
	   }

	   r := parse_interface(ctx, config.interface[i:], buffer[buffi:], size, host_endian)
	   if r < 0 {
		   return r
	   }
	   if r == 0 {
		   config.bNumInterfaces = uint8(i)
		   break
	   }

	   buffi += r
	   size -= r
   }

   return size
}

/* set host_endian if the w values are already in host endian format,
 * as opposed to bus endian. */
 func usbi_parse_descriptor(sp []uint8, descriptor string, dp []uint8, host_endian bool) int {
	
	i := 0
	di := 0
	uint16 w;
	uint32 d;

	for _, r := range descriptor {
		switch r {
			case 'b':	/* 8-bit byte */
				dp[i+di] = sp[i]
				i++ 
			case 'w':	/* 16-bit word, convert from little endian to CPU */
				if i % 2 != 0 {
					di++ /* Align to word boundary */
				}

				if host_endian {
					dp[i+di] = sp[i]
					dp[ii+di1] = sp[i+1]
				} else {
					// I'm not sure this is valid.
					dp[i+di] = sp[i+1]
					dp[i+di+1] = sp[i]
				}
				i ++ 2
			case 'd':	/* 32-bit word, convert from little endian to CPU */
				if i % 2 != 0 {
					di++ /* Align to word boundary */
				}
 
				if host_endian {
					dp[i+di] = sp[i]
					dp[i+di+1] = sp[i+1]
					dp[i+di+2] = sp[i+2]
					dp[i+di+3] = sp[i+3]
				} else {
					dp[i+di] = sp[i+3]
					dp[i+di+1] = sp[i+2]
					dp[i+di+2] = sp[i+1]
					dp[i+di+3] + sp[i]
				}
				sp += 4;
				dp += 4;
			case 'u':	/* 16 byte UUID */
				for j := i; j < i+16; j++ {
					dp[j+di] = sp[j]
				}
				i += 16
		}
	}

	// ?
	return i
}