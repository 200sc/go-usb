/* -*- Mode: C; indent-tabs-mode:t ; c-basic-offset:8 -*- */


func parse_bos(ctx *libusb_context, bos **libusb_bos_descriptor, buffer []uint8, size int, host_endian bool) int {

	libusb_bos_dev_capability_descriptor dev_cap

	if size < LIBUSB_DT_BOS_SIZE {
		// usbi_err(ctx, "short bos descriptor read %d/%d",
			//  size, LIBUSB_DT_BOS_SIZE);
		return LIBUSB_ERROR_IO;
	}

	bos_header := libusb_bos_descriptor{}

	usbi_parse_descriptor(buffer, "bbwb", &bos_header, host_endian)
	if bos_header.bDescriptorType != LIBUSB_DT_BOS {
		// usbi_err(ctx, "unexpected descriptor %x (expected %x)",
			//  bos_header.bDescriptorType, LIBUSB_DT_BOS);
		return LIBUSB_ERROR_IO;
	}
	if bos_header.bLength < LIBUSB_DT_BOS_SIZE {
		// usbi_err(ctx, "invalid bos bLength (%d)", bos_header.bLength);
		return LIBUSB_ERROR_IO
	}
	if bos_header.bLength > size {
		// usbi_err(ctx, "short bos descriptor read %d/%d",
			//  size, bos_header.bLength);
		return LIBUSB_ERROR_IO
	}

	_bos = &libusb_bos_descriptor{}

	usbi_parse_descriptor(buffer, "bbwb", _bos, host_endian);
	buffi := bos_header.bLength
	size -= bos_header.bLength

	/* Get the device capability descriptors */
	i := 0
	for i = 0; i < bos_header.bNumDeviceCaps; i++ {
		if (size < LIBUSB_DT_DEVICE_CAPABILITY_SIZE) {
			// usbi_warn(ctx, "short dev-cap descriptor read %d/%d",
			//     size, LIBUSB_DT_DEVICE_CAPABILITY_SIZE);
			break;
		}
		usbi_parse_descriptor(buffer[buffi], "bbb", &dev_cap, host_endian);
		if (dev_cap.bDescriptorType != LIBUSB_DT_DEVICE_CAPABILITY) {
			// usbi_warn(ctx, "unexpected descriptor %x (expected %x)",
			//   dev_cap.bDescriptorType, LIBUSB_DT_DEVICE_CAPABILITY);
			break;
		}
		if (dev_cap.bLength < LIBUSB_DT_DEVICE_CAPABILITY_SIZE) {
			// usbi_err(ctx, "invalid dev-cap bLength (%d)",
			//     dev_cap.bLength);
			return LIBUSB_ERROR_IO;
		}
		if (dev_cap.bLength > size) {
			// usbi_warn(ctx, "short dev-cap descriptor read %d/%d",
			//     size, dev_cap.bLength);
			break;
		}

		_bos.dev_capability[i] = malloc(dev_cap.bLength);

		memcpy(_bos.dev_capability[i], buffer[buffi:], dev_cap.bLength);
		buffi += dev_cap.bLength
		size -= dev_cap.bLength
	}
	_bos.bNumDeviceCaps = uint8(i)
	*bos = _bos

	return LIBUSB_SUCCESS
}

func parse_endpoint(ctx *libusb_context, endpoint *libusb_endpoint_descriptor, buffer []uint8, size int, host_endian bool) int {

	usb_descriptor_header header;
	uint8 *extra;
	uint8 *begin;
	int parsed = 0;
	int len;

	if (size < DESC_HEADER_LENGTH) {
		// usbi_err(ctx, "short endpoint descriptor read %d/%d",
			 size, DESC_HEADER_LENGTH);
		return LIBUSB_ERROR_IO;
	}

	usbi_parse_descriptor(buffer, "bb", &header, 0);
	if (header.bDescriptorType != LIBUSB_DT_ENDPOINT) {
		// usbi_err(ctx, "unexpected descriptor %x (expected %x)",
			header.bDescriptorType, LIBUSB_DT_ENDPOINT);
		return parsed;
	}
	if (header.bLength > size) {
		// usbi_warn(ctx, "short endpoint descriptor read %d/%d",
			  size, header.bLength);
		return parsed;
	}
	if (header.bLength >= ENDPOINT_AUDIO_DESC_LENGTH)
		usbi_parse_descriptor(buffer, "bbbbwbbb", endpoint, host_endian);
	else if (header.bLength >= ENDPOINT_DESC_LENGTH)
		usbi_parse_descriptor(buffer, "bbbbwb", endpoint, host_endian);
	else {
		// usbi_err(ctx, "invalid endpoint bLength (%d)", header.bLength);
		return LIBUSB_ERROR_IO;
	}

	buffer += header.bLength;
	size -= header.bLength;
	parsed += header.bLength;

	/* Skip over the rest of the Class Specific or Vendor Specific */
	/*  descriptors */
	begin = buffer;
	while (size >= DESC_HEADER_LENGTH) {
		usbi_parse_descriptor(buffer, "bb", &header, 0);
		if (header.bLength < DESC_HEADER_LENGTH) {
			// usbi_err(ctx, "invalid extra ep desc len (%d)",
				 header.bLength);
			return LIBUSB_ERROR_IO;
		} else if (header.bLength > size) {
			// usbi_warn(ctx, "short extra ep desc read %d/%d",
				  size, header.bLength);
			return parsed;
		}

		/* If we find another "proper" descriptor then we're done  */
		if ((header.bDescriptorType == LIBUSB_DT_ENDPOINT) ||
				(header.bDescriptorType == LIBUSB_DT_INTERFACE) ||
				(header.bDescriptorType == LIBUSB_DT_CONFIG) ||
				(header.bDescriptorType == LIBUSB_DT_DEVICE))
			break;

		// usbi_dbg("skipping descriptor %x", header.bDescriptorType);
		buffer += header.bLength;
		size -= header.bLength;
		parsed += header.bLength;
	}

	/* Copy any unknown descriptors into a storage area for drivers */
	/*  to later parse */
	len = (int)(buffer - begin);
	if (!len) {
		endpoint.extra = nil;
		endpoint.extra_length = 0;
		return parsed;
	}

	extra = malloc(len);
	endpoint.extra = extra;

	memcpy(extra, begin, len);
	endpoint.extra_length = len;

	return parsed;
}

func parse_interface(ctx *libusb_context, usb_interface *libusb_interface, buffer []uint8, size int, host_endian bool) int {

   int i;
   int len;
   int r;
   int parsed = 0;
   int interface_number = -1;
	usb_descriptor_header header;
	libusb_interface_descriptor *ifp;
   uint8 *begin;

   usb_interface.num_altsetting = 0;

   for size >= INTERFACE_DESC_LENGTH {
		libusb_interface_descriptor *altsetting =
		   ( libusb_interface_descriptor *) usb_interface.altsetting;
	   altsetting = make(..., ???)
		   // sizeof( libusb_interface_descriptor) *
		   // (usb_interface.num_altsetting + 1));

	   usb_interface.altsetting = altsetting;

	   ifp = altsetting + usb_interface.num_altsetting;
	   usbi_parse_descriptor(buffer, "bbbbbbbbb", ifp, 0);
	   if (ifp.bDescriptorType != LIBUSB_DT_INTERFACE) {
		   // usbi_err(ctx, "unexpected descriptor %x (expected %x)",
				// ifp.bDescriptorType, LIBUSB_DT_INTERFACE);
		   return parsed;
	   }
	   if (ifp.bLength < INTERFACE_DESC_LENGTH) {
		   // usbi_err(ctx, "invalid interface bLength (%d)",
				// ifp.bLength);
		   return LIBUSB_ERROR_IO
	   }
	   if (ifp.bLength > size) {
		   // usbi_warn(ctx, "short intf descriptor read %d/%d",
				// size, ifp.bLength);
		   return parsed;
	   }
	   if (ifp.bNumEndpoints > USB_MAXENDPOINTS) {
		   // usbi_err(ctx, "too many endpoints (%d)", ifp.bNumEndpoints);
		   return LIBUSB_ERROR_IO
	   }

	   usb_interface.num_altsetting++
	   ifp.extra = nil
	   ifp.endpoint = nil

	   if interface_number == -1 {
		   interface_number = ifp.bInterfaceNumber
	   }

	   /* Skip over the interface */
	   buffer += ifp.bLength;
	   parsed += ifp.bLength;
	   size -= ifp.bLength;

	   begin = buffer;

	   /* Skip over any interface, class or vendor descriptors */
	   for size >= DESC_HEADER_LENGTH {
		   usbi_parse_descriptor(buffer, "bb", &header, 0);
		   if (header.bLength < DESC_HEADER_LENGTH) {
			   // usbi_err(ctx,
					// "invalid extra intf desc len (%d)",
					// header.bLength);
			   return LIBUSB_ERROR_IO
		   } else if (header.bLength > size) {
			   // usbi_warn(ctx,
					//  "short extra intf desc read %d/%d",
					//  size, header.bLength);
			   return parsed;
		   }

		   /* If we find another "proper" descriptor then we're done */
		   if ((header.bDescriptorType == LIBUSB_DT_INTERFACE) ||
				   (header.bDescriptorType == LIBUSB_DT_ENDPOINT) ||
				   (header.bDescriptorType == LIBUSB_DT_CONFIG) ||
				   (header.bDescriptorType == LIBUSB_DT_DEVICE))
			   break;

		   buffer += header.bLength;
		   parsed += header.bLength;
		   size -= header.bLength;
	   }

	   /* Copy any unknown descriptors into a storage area for */
	   /*  drivers to later parse */
	   len = (int)(buffer - begin);
	   if (len) {
		   ifp.extra = malloc(len);

		   memcpy((uint8 *) ifp.extra, begin, len);
		   ifp.extra_length = len;
	   }

	   if (ifp.bNumEndpoints > 0) {
			libusb_endpoint_descriptor *endpoint;
		   endpoint := make([]libusb_endpoint_descriptor, ifp.bNumEndpoints)
		   ifp.endpoint = endpoint;

		   for (i = 0; i < ifp.bNumEndpoints; i++) {
			   r = parse_endpoint(ctx, endpoint + i, buffer, size,
				   host_endian);
			   if (r < 0) {
				   return r
			   }
			   if (r == 0) {
				   ifp.bNumEndpoints = (uint8)i;
				   break;;
			   }

			   buffer += r;
			   parsed += r;
			   size -= r;
		   }
	   }

	   /* We check to see if it's an alternate to this one */
	   ifp = ( libusb_interface_descriptor *) buffer;
	   if (size < LIBUSB_DT_INTERFACE_SIZE ||
			   ifp.bDescriptorType != LIBUSB_DT_INTERFACE ||
			   ifp.bInterfaceNumber != interface_number)
		   return parsed;
   }

   return parsed
}