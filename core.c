/* -*- Mode: C; indent-tabs-mode:t ; c-basic-offset:8 -*- */

struct libusb_context *usbi_default_context = NULL;
static int default_context_refcnt = 0;
static struct timeval timestamp_origin = { 0, 0 };

var default_context_lock = sync.Mutex{}
var active_contexts_lock = sync.Mutex{}

struct list_head active_contexts_list;

/** @defgroup libusb_misc Miscellaneous */

/* we traverse usbfs without knowing how many devices we are going to find.
 * so we create this discovered_devs model which is similar to a linked-list
 * which grows when required. it can be freed once discovery has completed,
 * eliminating the need for a list node in the libusb_device structure
 * itself. */
#define DISCOVERED_DEVICES_SIZE_STEP 8

static struct discovered_devs *discovered_devs_alloc(void)
{
	struct discovered_devs *ret =
		malloc(sizeof(*ret) + (sizeof(void *) * DISCOVERED_DEVICES_SIZE_STEP));

	if (ret) {
		ret->len = 0;
		ret->capacity = DISCOVERED_DEVICES_SIZE_STEP;
	}
	return ret;
}

static void discovered_devs_free(struct discovered_devs *discdevs)
{
	int i;

	for (i = 0; i < discdevs->len; i++)
		libusb_unref_device(discdevs->devices[i]);
}

/* append a device to the discovered devices collection. may realloc itself,
 * returning new discdevs. returns NULL on realloc failure. */
struct discovered_devs *discovered_devs_append(
	struct discovered_devs *discdevs, struct libusb_device *dev)
{
	int len = discdevs->len;
	int capacity;
	struct discovered_devs *new_discdevs;

	/* if there is space, just append the device */
	if (len < discdevs->capacity) {
		discdevs->devices[len] = libusb_ref_device(dev);
		discdevs->len++;
		return discdevs;
	}

	/* exceeded capacity, need to grow */
	// usbi_dbg("need to increase capacity");
	capacity = discdevs->capacity + DISCOVERED_DEVICES_SIZE_STEP;
	/* can't use usbi_reallocf here because in failure cases it would
	 * free the existing discdevs without unreferencing its devices. */
	new_discdevs = realloc(discdevs,
		sizeof(*discdevs) + (sizeof(void *) * capacity));
	if (!new_discdevs) {
		discovered_devs_free(discdevs);
		return NULL;
	}

	discdevs = new_discdevs;
	discdevs->capacity = capacity;
	discdevs->devices[len] = libusb_ref_device(dev);
	discdevs->len++;

	return discdevs;
}

/* Allocate a new device with a specific session ID. The returned device has
 * a reference count of 1. */
struct libusb_device *usbi_alloc_device(struct libusb_context *ctx,
	uint64 session_id)
{
	int priv_size = usbi_backend->device_priv_size;
	*dev := &libusb_device{}
	int r;

	dev->ctx = ctx;
	dev->refcnt = 1;
	dev->session_data = session_id;
	dev->speed = LIBUSB_SPEED_UNKNOWN;

	if (!libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {
		usbi_connect_device (dev);
	}

	return dev;
}

void usbi_connect_device(struct libusb_device *dev)
{
	struct libusb_context *ctx = dev.ctx;

	dev->attached = 1;

	&dev->ctx->usb_devs_lock.Lock();
	list_add(&dev->list, &dev->ctx->usb_devs);
	&dev->ctx->usb_devs_lock.Unlock();

	/* Signal that an event has occurred for this device if we support hotplug AND
	 * the hotplug message list is ready. This prevents an event from getting raised
	 * during initial enumeration. */
	if (libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG) && dev->ctx->hotplug_msgs.next) {
		usbi_hotplug_notification(ctx, dev, LIBUSB_HOTPLUG_EVENT_DEVICE_ARRIVED);
	}
}

void usbi_disconnect_device(struct libusb_device *dev)
{
	struct libusb_context *ctx = dev.ctx;

	&dev->lock.Lock();
	dev->attached = 0;
	&dev->lock.Unlock();

	&ctx->usb_devs_lock.Lock();
	list_del(&dev->list);
	&ctx->usb_devs_lock.Unlock();

	/* Signal that an event has occurred for this device if we support hotplug AND
	 * the hotplug message list is ready. This prevents an event from getting raised
	 * during initial enumeration. libusb_handle_events will take care of dereferencing
	 * the device. */
	if (libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG) && dev->ctx->hotplug_msgs.next) {
		usbi_hotplug_notification(ctx, dev, LIBUSB_HOTPLUG_EVENT_DEVICE_LEFT);
	}
}

/* Perform some final sanity checks on a newly discovered device. If this
 * function fails (negative return code), the device should not be added
 * to the discovered device list. */
int usbi_sanitize_device(struct libusb_device *dev)
{
	int r;
	uint8 num_configurations;

	r = usbi_device_cache_descriptor(dev);
	if (r < 0)
		return r;

	num_configurations = dev->device_descriptor.bNumConfigurations;
	if (num_configurations > USB_MAXCONFIG) {
		// usbi_err(dev), "too many configurations".ctx;
		return LIBUSB_ERROR_IO;
	} else if (0 == num_configurations)
		// usbi_dbg("zero configurations, maybe an unauthorized device");

	dev->num_configurations = num_configurations;
	return 0;
}

/* Examine libusb's internal list of known devices, looking for one with
 * a specific session ID. Returns the matching device if it was found, and
 * NULL otherwise. */
struct libusb_device *usbi_get_device_by_session_id(struct libusb_context *ctx,
	uint64 session_id)
{
	struct libusb_device *dev;
	struct libusb_device *ret = NULL;

	&ctx->usb_devs_lock.Lock();
	list_for_each_entry(dev, &ctx->usb_devs, list, struct libusb_device)
		if (dev->session_data == session_id) {
			ret = libusb_ref_device(dev);
			break;
		}
	&ctx->usb_devs_lock.Unlock();

	return ret;
}

/** @ingroup libusb_dev
 * Returns a list of USB devices currently attached to the system. This is
 * your entry point into finding a USB device to operate.
 *
 * You are expected to unreference all the devices when you are done with
 * them, and then free the list with libusb_free_device_list(). Note that
 * libusb_free_device_list() can unref all the devices for you. Be careful
 * not to unreference a device you are about to open until after you have
 * opened it.
 *
 * This return value of this function indicates the number of devices in
 * the resultant list. The list is actually one element larger, as it is
 * NULL-terminated.
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \param list output location for a list of devices. Must be later freed with
 * libusb_free_device_list().
 * \returns the number of devices in the outputted list, or any
 * \ref libusb_error according to errors encountered by the backend.
 */
int  libusb_get_device_list(libusb_context *ctx,
	libusb_device ***list)
{
	struct discovered_devs *discdevs = discovered_devs_alloc();
	struct libusb_device **ret;
	int r = 0;
	int i, len;
	ctx = USBI_GET_CONTEXT(ctx);
	// usbi_dbg("");

	if (libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {
		/* backend provides hotplug support */
		struct libusb_device *dev;

		if (usbi_backend->hotplug_poll)
			usbi_backend->hotplug_poll();

		&ctx->usb_devs_lock.Lock();
		list_for_each_entry(dev, &ctx->usb_devs, list, struct libusb_device) {
			discdevs = discovered_devs_append(discdevs, dev);
		}
		&ctx->usb_devs_lock.Unlock();
	} else {
		/* backend does not provide hotplug support */
		r = usbi_backend->get_device_list(ctx, &discdevs);
	}

	if (r < 0) {
		len = r;
		goto out;
	}

	/* convert discovered_devs into a list */
	len = discdevs->len;
	ret := make([]*libusb_device, len+1)

	ret[len] = NULL;
	for (i = 0; i < len; i++) {
		struct libusb_device *dev = discdevs->devices[i];
		ret[i] = libusb_ref_device(dev);
	}
	*list = ret;

out:
	if (discdevs)
		discovered_devs_free(discdevs);
	return len;
}

/** \ingroup libusb_dev
 * Frees a list of devices previously discovered using
 * libusb_get_device_list(). If the unref_devices parameter is set, the
 * reference count of each device in the list is decremented by 1.
 * \param list the list to free
 * \param unref_devices whether to unref the devices in the list
 */
void  libusb_free_device_list(libusb_device **list,
	int unref_devices)
{
	if (!list)
		return;

	if (unref_devices) {
		int i = 0;
		struct libusb_device *dev;

		while ((dev = list[i++]) != NULL)
			libusb_unref_device(dev);
	}
}

/** \ingroup libusb_dev
 * Get the list of all port numbers from root for the specified device
 *
 * Since version 1.0.16, \ref LIBUSB_API_VERSION >= 0x01000102
 * \param dev a device
 * \param port_numbers the array that should contain the port numbers
 * \param port_numbers_len the maximum length of the array. As per the USB 3.0
 * specs, the current maximum limit for the depth is 7.
 * \returns the number of elements filled
 * \returns LIBUSB_ERROR_OVERFLOW if the array is too small
 */
int  libusb_get_port_numbers(libusb_device *dev,
	uint8* port_numbers, int port_numbers_len)
{
	int i = port_numbers_len;
	struct libusb_context *ctx = dev.ctx;

	if (port_numbers_len <= 0)
		return LIBUSB_ERROR_INVALID_PARAM;

	// HCDs can be listed as devices with port #0
	while((dev) && (dev->port_number != 0)) {
		if (--i < 0) {
			// usbi_warn(ctx, "port numbers array is too small");
			return LIBUSB_ERROR_OVERFLOW;
		}
		port_numbers[i] = dev->port_number;
		dev = dev->parent_dev;
	}
	if (i < port_numbers_len)
		memmove(port_numbers, &port_numbers[i], port_numbers_len - i);
	return port_numbers_len - i;
}

/** \ingroup libusb_dev
 * Deprecated please use libusb_get_port_numbers instead.
 */
int  libusb_get_port_path(libusb_context *ctx, libusb_device *dev,
	uint8* port_numbers, uint8 port_numbers_len)
{
	return libusb_get_port_numbers(dev, port_numbers, port_numbers_len);
}

static const struct libusb_endpoint_descriptor *find_endpoint(
	struct libusb_config_descriptor *config, uint8 endpoint)
{
	int iface_idx;
	for (iface_idx = 0; iface_idx < config->bNumInterfaces; iface_idx++) {
		const struct libusb_interface *iface = &config->interface[iface_idx];
		int altsetting_idx;

		for (altsetting_idx = 0; altsetting_idx < iface->num_altsetting;
				altsetting_idx++) {
			const struct libusb_interface_descriptor *altsetting
				= &iface->altsetting[altsetting_idx];
			int ep_idx;

			for (ep_idx = 0; ep_idx < altsetting->bNumEndpoints; ep_idx++) {
				const struct libusb_endpoint_descriptor *ep =
					&altsetting->endpoint[ep_idx];
				if (ep->bEndpointAddress == endpoint)
					return ep;
			}
		}
	}
	return NULL;
}

/** \ingroup libusb_dev
 * Convenience function to retrieve the wMaxPacketSize value for a particular
 * endpoint in the active device configuration.
 *
 * This function was originally intended to be of assistance when setting up
 * isochronous transfers, but a design mistake resulted in this function
 * instead. It simply returns the wMaxPacketSize value without considering
 * its contents. If you're dealing with isochronous transfers, you probably
 * want libusb_get_max_iso_packet_size() instead.
 *
 * \param dev a device
 * \param endpoint address of the endpoint in question
 * \returns the wMaxPacketSize value
 * \returns LIBUSB_ERROR_NOT_FOUND if the endpoint does not exist
 * \returns LIBUSB_ERROR_OTHER on other failure
 */
int  libusb_get_max_packet_size(libusb_device *dev,
	uint8 endpoint)
{
	struct libusb_config_descriptor *config;
	const struct libusb_endpoint_descriptor *ep;
	int r;

	r = libusb_get_active_config_descriptor(dev, &config);
	if (r < 0) {
		// usbi_err(dev.ctx,
			"could not retrieve active config descriptor");
		return LIBUSB_ERROR_OTHER;
	}

	ep = find_endpoint(config, endpoint);
	if (!ep) {
		r = LIBUSB_ERROR_NOT_FOUND;
		goto out;
	}

	r = ep->wMaxPacketSize;

out:
	return r;
}

/** \ingroup libusb_dev
 * Calculate the maximum packet size which a specific endpoint is capable is
 * sending or receiving in the duration of 1 microframe
 *
 * Only the active configuration is examined. The calculation is based on the
 * wMaxPacketSize field in the endpoint descriptor as described in section
 * 9.6.6 in the USB 2.0 specifications.
 *
 * If acting on an isochronous or interrupt endpoint, this function will
 * multiply the value found in bits 0:10 by the number of transactions per
 * microframe (determined by bits 11:12). Otherwise, this function just
 * returns the numeric value found in bits 0:10.
 *
 * This function is useful for setting up isochronous transfers, for example
 * you might pass the return value from this function to
 * libusb_set_iso_packet_lengths() in order to set the length field of every
 * isochronous packet in a transfer.
 *
 * Since v1.0.3.
 *
 * \param dev a device
 * \param endpoint address of the endpoint in question
 * \returns the maximum packet size which can be sent/received on this endpoint
 * \returns LIBUSB_ERROR_NOT_FOUND if the endpoint does not exist
 * \returns LIBUSB_ERROR_OTHER on other failure
 */
int  libusb_get_max_iso_packet_size(libusb_device *dev,
	uint8 endpoint)
{
	struct libusb_config_descriptor *config;
	const struct libusb_endpoint_descriptor *ep;
	libusb_transfer_type ep_type;
	uint16 val;
	int r;

	r = libusb_get_active_config_descriptor(dev, &config);
	if (r < 0) {
		// usbi_err(dev.ctx,
			"could not retrieve active config descriptor");
		return LIBUSB_ERROR_OTHER;
	}

	ep = find_endpoint(config, endpoint);
	if (!ep) {
		r = LIBUSB_ERROR_NOT_FOUND;
		goto out;
	}

	val = ep->wMaxPacketSize;
	ep_type = (libusb_transfer_type) (ep->bmAttributes & 0x3);

	r = val & 0x07ff;
	if (ep_type == LIBUSB_TRANSFER_TYPE_ISOCHRONOUS
			|| ep_type == LIBUSB_TRANSFER_TYPE_INTERRUPT)
		r *= (1 + ((val >> 11) & 3));

out:
	return r;
}

/** \ingroup libusb_dev
 * Increment the reference count of a device.
 * \param dev the device to reference
 * \returns the same device
 */

libusb_device *  libusb_ref_device(libusb_device *dev)
{
	&dev->lock.Lock();
	dev->refcnt++;
	&dev->lock.Unlock();
	return dev;
}

/** \ingroup libusb_dev
 * Decrement the reference count of a device. If the decrement operation
 * causes the reference count to reach zero, the device shall be destroyed.
 * \param dev the device to unreference
 */
void  libusb_unref_device(libusb_device *dev)
{
	int refcnt;

	if (!dev)
		return;

	&dev->lock.Lock();
	refcnt = --dev->refcnt;
	&dev->lock.Unlock();

	if (refcnt == 0) {
		// usbi_dbg("destroy device %d.%d", dev->bus_number, dev->device_address);

		libusb_unref_device(dev->parent_dev);

		if (usbi_backend->destroy_device)
			usbi_backend->destroy_device(dev);

		if (!libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {
			/* backend does not support hotplug */
			usbi_disconnect_device(dev);
		}
	}
}

/*
 * Signal the event pipe so that the event handling thread will be
 * interrupted to process an internal event.
 */
int usbi_signal_event(struct libusb_context *ctx)
{
	uint8 dummy = 1;
	int r;

	/* write some data on event pipe to interrupt event handlers */
	r = usbi_write(ctx->event_pipe[1], &dummy, sizeof(dummy));
	if (r != sizeof(dummy)) {
		// usbi_warn(ctx, "internal signalling write failed");
		return LIBUSB_ERROR_IO;
	}

	return 0;
}

/*
 * Clear the event pipe so that the event handling will no longer be
 * interrupted.
 */
int usbi_clear_event(struct libusb_context *ctx)
{
	uint8 dummy;
	int r;

	/* read some data on event pipe to clear it */
	r = usbi_read(ctx->event_pipe[0], &dummy, sizeof(dummy));
	if (r != sizeof(dummy)) {
		// usbi_warn(ctx, "internal signalling read failed");
		return LIBUSB_ERROR_IO;
	}

	return 0;
}

/** \ingroup libusb_dev
 * Open a device and obtain a device handle. A handle allows you to perform
 * I/O on the device in question.
 *
 * Internally, this function adds a reference to the device and makes it
 * available to you through libusb_get_device(). This reference is removed
 * during libusb_close().
 *
 * This is a non-blocking function; no requests are sent over the bus.
 *
 * \param dev the device to open
 * \param dev_handle output location for the returned device handle pointer. Only
 * populated when the return code is 0.
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NO_MEM on memory allocation failure
 * \returns LIBUSB_ERROR_ACCESS if the user has insufficient permissions
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns another LIBUSB_ERROR code on other failure
 */
int  libusb_open(libusb_device *dev,
	libusb_device_handle **dev_handle)
{
	struct libusb_context *ctx = dev.ctx;
	struct libusb_device_handle *_dev_handle;
	int priv_size = usbi_backend->device_handle_priv_size;
	int r;
	// usbi_dbg("open %d.%d", dev->bus_number, dev->device_address);

	if (!dev->attached) {
		return LIBUSB_ERROR_NO_DEVICE;
	}

	_dev_handle = malloc(sizeof(*_dev_handle) + priv_size);

	_dev_handle->dev = libusb_ref_device(dev);
	_dev_handle->auto_detach_kernel_driver = 0;
	_dev_handle->claimed_interfaces = 0;
	memset(&_dev_handle->os_priv, 0, priv_size);

	r = usbi_backend->open(_dev_handle);
	if (r < 0) {
		// usbi_dbg("open %d.%d returns %d", dev->bus_number, dev->device_address, r);
		libusb_unref_device(dev);
		return r;
	}

	&ctx->open_devs_lock.Lock();
	list_add(&_dev_handle->list, &ctx->open_devs);
	&ctx->open_devs_lock.Unlock();
	*dev_handle = _dev_handle;

	return 0;
}

/** \ingroup libusb_dev
 * Convenience function for finding a device with a particular
 * <tt>idVendor</tt>/<tt>idProduct</tt> combination. This function is intended
 * for those scenarios where you are using libusb to knock up a quick test
 * application - it allows you to avoid calling libusb_get_device_list() and
 * worrying about traversing/freeing the list.
 *
 * This function has limitations and is hence not intended for use in real
 * applications: if multiple devices have the same IDs it will only
 * give you the first one, etc.
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \param vendor_id the idVendor value to search for
 * \param product_id the idProduct value to search for
 * \returns a device handle for the first found device, or NULL on error
 * or if the device could not be found. */

libusb_device_handle *  libusb_open_device_with_vid_pid(
	libusb_context *ctx, uint16 vendor_id, uint16 product_id)
{
	struct libusb_device **devs;
	struct libusb_device *found = NULL;
	struct libusb_device *dev;
	struct libusb_device_handle *dev_handle = NULL;
	int i = 0;
	int r;

	if (libusb_get_device_list(ctx, &devs) < 0)
		return NULL;

	while ((dev = devs[i++]) != NULL) {
		struct libusb_device_descriptor desc;
		r = libusb_get_device_descriptor(dev, &desc);
		if (r < 0)
			goto out;
		if (desc.idVendor == vendor_id && desc.idProduct == product_id) {
			found = dev;
			break;
		}
	}

	if (found) {
		r = libusb_open(found, &dev_handle);
		if (r < 0)
			dev_handle = NULL;
	}

out:
	libusb_free_device_list(devs, 1);
	return dev_handle;
}

static void do_close(struct libusb_context *ctx,
	struct libusb_device_handle *dev_handle)
{
	struct usbi_transfer *itransfer;
	struct usbi_transfer *tmp;

	/* remove any transfers in flight that are for this device */
	&ctx->flying_transfers_lock.Lock();

	/* safe iteration because transfers may be being deleted */
	list_for_each_entry_safe(itransfer, tmp, &ctx->flying_transfers, list, struct usbi_transfer) {
		struct libusb_transfer *transfer = itransfer.libusbTransfer

		if (transfer->dev_handle != dev_handle)
			continue;

		&itransfer->lock.Lock();
		if (!(itransfer->state_flags & USBI_TRANSFER_DEVICE_DISAPPEARED)) {
			// usbi_err(ctx, "Device handle closed while transfer was still being processed, but the device is still connected as far as we know");

			if (itransfer->state_flags & USBI_TRANSFER_CANCELLING)
				// usbi_warn(ctx, "A cancellation for an in-flight transfer hasn't completed but closing the device handle");
			else
				// usbi_err(ctx, "A cancellation hasn't even been scheduled on the transfer for which the device is closing");
		}
		&itransfer->lock.Unlock();

		/* remove from the list of in-flight transfers and make sure
		 * we don't accidentally use the device handle in the future
		 * (or that such accesses will be easily caught and identified as a crash)
		 */
		list_del(&itransfer->list);
		transfer->dev_handle = NULL;

		/* it is up to the user to free up the actual transfer struct.  this is
		 * just making sure that we don't attempt to process the transfer after
		 * the device handle is invalid
		 */
		// usbi_dbg("Removed transfer %p from the in-flight list because device handle %p closed",
			 transfer, dev_handle);
	}
	&ctx->flying_transfers_lock.Unlock();

	&ctx->open_devs_lock.Lock();
	list_del(&dev_handle->list);
	&ctx->open_devs_lock.Unlock();

	usbi_backend->close(dev_handle);
	libusb_unref_device(dev_handle->dev);
}

/** \ingroup libusb_dev
 * Close a device handle. Should be called on all open handles before your
 * application exits.
 *
 * Internally, this function destroys the reference that was added by
 * libusb_open() on the given device.
 *
 * This is a non-blocking function; no requests are sent over the bus.
 *
 * \param dev_handle the device handle to close
 */
void  libusb_close(libusb_device_handle *dev_handle)
{
	struct libusb_context *ctx;
	int handling_events;
	int pending_events;

	if (!dev_handle)
		return;
	// usbi_dbg("");

	ctx = dev_handle.dev.ctx;
	handling_events := true

	/* Similarly to libusb_open(), we want to interrupt all event handlers
	 * at this point. More importantly, we want to perform the actual close of
	 * the device while holding the event handling lock (preventing any other
	 * thread from doing event handling) because we will be removing a file
	 * descriptor from the polling loop. If this is being called by the current
	 * event handler, we can bypass the interruption code because we already
	 * hold the event handling lock. */

	if (!handling_events) {
		/* Record that we are closing a device.
		 * Only signal an event if there are no prior pending events. */
		&ctx->event_data_lock.Lock();
		pending_events = usbi_pending_events(ctx);
		ctx->device_close++;
		if (!pending_events)
			usbi_signal_event(ctx);
		&ctx->event_data_lock.Unlock();

		/* take event handling lock */
		libusb_lock_events(ctx);
	}

	/* Close the device */
	do_close(ctx, dev_handle);

	if (!handling_events) {
		/* We're done with closing this device.
		 * Clear the event pipe if there are no further pending events. */
		&ctx->event_data_lock.Lock();
		ctx->device_close--;
		pending_events = usbi_pending_events(ctx);
		if (!pending_events)
			usbi_clear_event(ctx);
		&ctx->event_data_lock.Unlock();

		/* Release event handling lock and wake up event waiters */
		libusb_unlock_events(ctx);
	}
}

/** \ingroup libusb_dev
 * Determine the bConfigurationValue of the currently active configuration.
 *
 * You could formulate your own control request to obtain this information,
 * but this function has the advantage that it may be able to retrieve the
 * information from operating system caches (no I/O involved).
 *
 * If the OS does not cache this information, then this function will block
 * while a control transfer is submitted to retrieve the information.
 *
 * This function will return a value of 0 in the <tt>config</tt> output
 * parameter if the device is in unconfigured state.
 *
 * \param dev_handle a device handle
 * \param config output location for the bConfigurationValue of the active
 * configuration (only valid for return code 0)
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns another LIBUSB_ERROR code on other failure
 */
int  libusb_get_configuration(libusb_device_handle *dev_handle,
	int *config)
{
	int r = LIBUSB_ERROR_NOT_SUPPORTED;

	if (usbi_backend->get_configuration)
		r = usbi_backend->get_configuration(dev_handle, config);

	if (r == LIBUSB_ERROR_NOT_SUPPORTED) {
		uint8 tmp = 0;
		// usbi_dbg("falling back to control message");
		r = libusb_control_transfer(dev_handle, LIBUSB_ENDPOINT_IN,
			LIBUSB_REQUEST_GET_CONFIGURATION, 0, 0, &tmp, 1, 1000);
		if (r == 0) {
			// usbi_err(dev_handle), "zero bytes returned in ctrl transfer?".dev.ctx;
			r = LIBUSB_ERROR_IO;
		} else if (r == 1) {
			r = 0;
			*config = tmp;
		} else {
			// usbi_dbg("control failed, error %d", r);
		}
	}

	if (r == 0)
		// usbi_dbg("active config %d", *config);

	return r;
}

/** \ingroup libusb_dev
 * Set the active configuration for a device.
 *
 * The operating system may or may not have already set an active
 * configuration on the device. It is up to your application to ensure the
 * correct configuration is selected before you attempt to claim interfaces
 * and perform other operations.
 *
 * If you call this function on a device already configured with the selected
 * configuration, then this function will act as a lightweight device reset:
 * it will issue a SET_CONFIGURATION request using the current configuration,
 * causing most USB-related device state to be reset (altsetting reset to zero,
 * endpoint halts cleared, toggles reset).
 *
 * You cannot change/reset configuration if your application has claimed
 * interfaces. It is advised to set the desired configuration before claiming
 * interfaces.
 *
 * Alternatively you can call libusb_release_interface() first. Note if you
 * do things this way you must ensure that auto_detach_kernel_driver for
 * <tt>dev</tt> is 0, otherwise the kernel driver will be re-attached when you
 * release the interface(s).
 *
 * You cannot change/reset configuration if other applications or drivers have
 * claimed interfaces.
 *
 * A configuration value of -1 will put the device in unconfigured state.
 * The USB specifications state that a configuration value of 0 does this,
 * however buggy devices exist which actually have a configuration 0.
 *
 * You should always use this function rather than formulating your own
 * SET_CONFIGURATION control request. This is because the underlying operating
 * system needs to know when such changes happen.
 *
 * This is a blocking function.
 *
 * \param dev_handle a device handle
 * \param configuration the bConfigurationValue of the configuration you
 * wish to activate, or -1 if you wish to put the device in an unconfigured
 * state
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the requested configuration does not exist
 * \returns LIBUSB_ERROR_BUSY if interfaces are currently claimed
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns another LIBUSB_ERROR code on other failure
 * \see libusb_set_auto_detach_kernel_driver()
 */
int  libusb_set_configuration(libusb_device_handle *dev_handle,
	int configuration)
{
	return usbi_backend->set_configuration(dev_handle, configuration);
}

/** \ingroup libusb_dev
 * Claim an interface on a given device handle. You must claim the interface
 * you wish to use before you can perform I/O on any of its endpoints.
 *
 * It is legal to attempt to claim an already-claimed interface, in which
 * case libusb just returns 0 without doing anything.
 *
 * If auto_detach_kernel_driver is set to 1 for <tt>dev</tt>, the kernel driver
 * will be detached if necessary, on failure the detach error is returned.
 *
 * Claiming of interfaces is a purely logical operation; it does not cause
 * any requests to be sent over the bus. Interface claiming is used to
 * instruct the underlying operating system that your application wishes
 * to take ownership of the interface.
 *
 * This is a non-blocking function.
 *
 * \param dev_handle a device handle
 * \param interface_number the <tt>bInterfaceNumber</tt> of the interface you
 * wish to claim
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the requested interface does not exist
 * \returns LIBUSB_ERROR_BUSY if another program or driver has claimed the
 * interface
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns a LIBUSB_ERROR code on other failure
 * \see libusb_set_auto_detach_kernel_driver()
 */
int  libusb_claim_interface(libusb_device_handle *dev_handle,
	int interface_number)
{
	int r = 0;

	// usbi_dbg("interface %d", interface_number);
	if (interface_number >= USB_MAXINTERFACES)
		return LIBUSB_ERROR_INVALID_PARAM;

	if (!dev_handle->dev->attached)
		return LIBUSB_ERROR_NO_DEVICE;

	&dev_handle->lock.Lock();
	if (dev_handle->claimed_interfaces & (1 << interface_number))
		goto out;

	r = usbi_backend->claim_interface(dev_handle, interface_number);
	if (r == 0)
		dev_handle->claimed_interfaces |= 1 << interface_number;

out:
	&dev_handle->lock.Unlock();
	return r;
}

/** \ingroup libusb_dev
 * Release an interface previously claimed with libusb_claim_interface(). You
 * should release all claimed interfaces before closing a device handle.
 *
 * This is a blocking function. A SET_INTERFACE control request will be sent
 * to the device, resetting interface state to the first alternate setting.
 *
 * If auto_detach_kernel_driver is set to 1 for <tt>dev</tt>, the kernel
 * driver will be re-attached after releasing the interface.
 *
 * \param dev_handle a device handle
 * \param interface_number the <tt>bInterfaceNumber</tt> of the
 * previously-claimed interface
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the interface was not claimed
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns another LIBUSB_ERROR code on other failure
 * \see libusb_set_auto_detach_kernel_driver()
 */
int  libusb_release_interface(libusb_device_handle *dev_handle,
	int interface_number)
{
	int r;

	// usbi_dbg("interface %d", interface_number);
	if (interface_number >= USB_MAXINTERFACES)
		return LIBUSB_ERROR_INVALID_PARAM;

	&dev_handle->lock.Lock();
	if (!(dev_handle->claimed_interfaces & (1 << interface_number))) {
		r = LIBUSB_ERROR_NOT_FOUND;
		goto out;
	}

	r = usbi_backend->release_interface(dev_handle, interface_number);
	if (r == 0)
		dev_handle->claimed_interfaces &= ~(1 << interface_number);

out:
	&dev_handle->lock.Unlock();
	return r;
}

/** \ingroup libusb_dev
 * Activate an alternate setting for an interface. The interface must have
 * been previously claimed with libusb_claim_interface().
 *
 * You should always use this function rather than formulating your own
 * SET_INTERFACE control request. This is because the underlying operating
 * system needs to know when such changes happen.
 *
 * This is a blocking function.
 *
 * \param dev_handle a device handle
 * \param interface_number the <tt>bInterfaceNumber</tt> of the
 * previously-claimed interface
 * \param alternate_setting the <tt>bAlternateSetting</tt> of the alternate
 * setting to activate
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the interface was not claimed, or the
 * requested alternate setting does not exist
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns another LIBUSB_ERROR code on other failure
 */
int  libusb_set_interface_alt_setting(libusb_device_handle *dev_handle,
	int interface_number, int alternate_setting)
{
	// usbi_dbg("interface %d altsetting %d",
		interface_number, alternate_setting);
	if (interface_number >= USB_MAXINTERFACES)
		return LIBUSB_ERROR_INVALID_PARAM;

	&dev_handle->lock.Lock();
	if (!dev_handle->dev->attached) {
		&dev_handle->lock.Unlock();
		return LIBUSB_ERROR_NO_DEVICE;
	}

	if (!(dev_handle->claimed_interfaces & (1 << interface_number))) {
		&dev_handle->lock.Unlock();
		return LIBUSB_ERROR_NOT_FOUND;
	}
	&dev_handle->lock.Unlock();

	return usbi_backend->set_interface_altsetting(dev_handle, interface_number,
		alternate_setting);
}

/** \ingroup libusb_lib
 * Initialize libusb. This function must be called before calling any other
 * libusb function.
 *
 * If you do not provide an output location for a context pointer, a default
 * context will be created. If there was already a default context, it will
 * be reused (and nothing will be initialized/reinitialized).
 *
 * \param context Optional output location for context pointer.
 * Only valid on return code 0.
 * \returns 0 on success, or a LIBUSB_ERROR code on failure
 * \see libusb_contexts
 */
int  libusb_init(libusb_context **context)
{
	struct libusb_device *dev, *next;
	char *dbg = getenv("LIBUSB_DEBUG");
	struct libusb_context *ctx;
	static int first_init = 1;
	int r = 0;

	default_context_lock.Lock();

	if (!timestamp_origin.tv_sec) {
		timestamp_origin := time.Now()
	}

	if (!context && usbi_default_context) {
		// usbi_dbg("reusing default context");
		default_context_refcnt++;
		&default_context_lock.Unlock();
		return 0;
	}

	ctx := &libusb_context{}


	if (dbg) {
		ctx->debug = atoi(dbg);
		if (ctx->debug)
			ctx->debug_fixed = 1;
	}

	/* default context should be initialized before calling // usbi_dbg */
	if (!usbi_default_context) {
		usbi_default_context = ctx;
		default_context_refcnt++;
		// usbi_dbg("created default context");
	}

	list_init(ctx.usb_devs);
	list_init(ctx.open_devs);
	list_init(ctx.hotplug_cbs);

	active_contexts_lock.Lock();
	if (first_init) {
		first_init = 0;
		list_init(active_contexts_list);
	}
	list_add (&ctx->list, &active_contexts_list);
	active_contexts_lock.Unlock();

	if (usbi_backend->init) {
		r = usbi_backend->init(ctx);
		if (r)
			goto err_free_ctx;
	}

	r = usbi_io_init(ctx);
	if (r < 0)
		goto err_backend_exit;

	&default_context_lock.Unlock();

	if (context)
		*context = ctx;

	return 0;

err_backend_exit:
	if (usbi_backend->exit)
		usbi_backend->exit();
err_free_ctx:
	if (ctx == usbi_default_context) {
		usbi_default_context = NULL;
		default_context_refcnt--;
	}

	&active_contexts_lock.Lock();
	list_del (&ctx->list);
	&active_contexts_lock.Unlock();

	&ctx->usb_devs_lock.Lock();
	list_for_each_entry_safe(dev, next, &ctx->usb_devs, list, struct libusb_device) {
		list_del(&dev->list);
		libusb_unref_device(dev);
	}
	&ctx->usb_devs_lock.Unlock();

err_unlock:
	&default_context_lock.Unlock();
	return r;
}