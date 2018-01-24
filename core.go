package usb

import (
	"strconv"
	"sync"
	"time"
)

/*
 * Core functions for libusb
 * Copyright © 2012-2013 Nathan Hjelm <hjelmn@cs.unm.edu>
 * Copyright © 2007-2008 Daniel Drake <dsd@gentoo.org>
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

var (
	usbi_default_context   *libusb_context
	default_context_refcnt int
	timestamp_origin       time.Time
	active_contexts_list   list_head
	default_context_lock   = sync.Mutex{}
	active_contexts_lock   = sync.Mutex{}
)

/* append a device to the discovered devices collection. may realloc itself,
 * returning new discdevs. returns nil on realloc failure. */
func discovered_devs_append(discdevs *discovered_devs, dev *libusb_device) *discovered_devs {
	discdevs.devices = append(discdevs.devices, libusb_ref_device(dev))
	return discdevs
}

/* Allocate a new device with a specific session ID. The returned device has
 * a reference count of 1. */
func usbi_alloc_device(ctx *libusb_context, session_id uint64) *libusb_device {
	priv_size := usbi_backend.device_priv_size
	dev := &libusb_device{}

	dev.ctx = ctx
	dev.refcnt = 1
	dev.session_data = session_id
	dev.speed = LIBUSB_SPEED_UNKNOWN

	if !libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG) {
		usbi_connect_device(dev)
	}

	return dev
}

func usbi_connect_device(dev *libusb_device) {
	ctx := dev.ctx

	dev.attached = 1

	dev.ctx.usb_devs_lock.Lock()
	list_add(&dev.list, &dev.ctx.usb_devs)
	dev.ctx.usb_devs_lock.Unlock()

	/* Signal that an event has occurred for this device if we support hotplug AND
	 * the hotplug message list is ready. This prevents an event from getting raised
	 * during initial enumeration. */
	if libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG) && dev.ctx.hotplug_msgs.next != nil {
		usbi_hotplug_notification(ctx, dev, LIBUSB_HOTPLUG_EVENT_DEVICE_ARRIVED)
	}
}

/** \ingroup libusb_lib
 * Deinitialize libusb. Should be called after closing all open devices and
 * before your application terminates.
 * \param ctx the context to deinitialize, or NULL for the default context
 */
func libusb_exit(ctx *libusb_context) {

	// usbi_dbg("")
	ctx = USBI_GET_CONTEXT(ctx)

	/* if working with default context, only actually do the deinitialization
	 * if we're the last user */
	default_context_lock.Lock()
	if ctx == usbi_default_context {
		default_context_refcnt--
		if default_context_refcnt > 0 {
			// usbi_dbg("not destroying default context")
			default_context_lock.Unlock()
			return
		}
	}
	default_context_lock.Unlock()

	active_contexts_lock.Lock()
	list_del(&ctx.list)
	active_contexts_lock.Unlock()

	if libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG) {
		usbi_hotplug_deregister_all(ctx)

		/*
		 * Ensure any pending unplug events are read from the hotplug
		 * pipe. The usb_device-s hold in the events are no longer part
		 * of usb_devs, but the events still hold a reference!
		 *
		 * Note we don't do this if the application has left devices
		 * open (which implies a buggy app) to avoid packet completion
		 * handlers running when the app does not expect them to run.
		 */
		var tv timeval
		if list_empty(&ctx.open_devs) {
			libusb_handle_events_timeout(ctx, &tv)
		}
	}

	/* a few sanity checks. don't bother with locking because unless
	 * there is an application bug, nobody will be accessing these. */

	usbi_io_exit(ctx)
	usbi_backend.exit()
}

/** \ingroup libusb_misc
 * Check at runtime if the loaded library has a given capability.
 * This call should be performed after \ref libusb_init(), to ensure the
 * backend has updated its capability set.
 *
 * \param capability the \ref libusb_capability to check for
 * \returns nonzero if the running library has the capability, 0 otherwise
 */
func libusb_has_capability(capability uint32) int {
	switch capability {
	case LIBUSB_CAP_HAS_CAPABILITY:
		return 1
	case LIBUSB_CAP_HAS_HOTPLUG:
		return ^(usbi_backend.get_device_list)
	case LIBUSB_CAP_HAS_HID_ACCESS:
		return (usbi_backend.caps & USBI_CAP_HAS_HID_ACCESS)
	case LIBUSB_CAP_SUPPORTS_DETACH_KERNEL_DRIVER:
		return (usbi_backend.caps & USBI_CAP_SUPPORTS_DETACH_KERNEL_DRIVER)
	}
	return 0
}

/** \ingroup libusb_dev
 * Clear the halt/stall condition for an endpoint. Endpoints with halt status
 * are unable to receive or transmit data until the halt condition is stalled.
 *
 * You should cancel all pending transfers before attempting to clear the halt
 * condition.
 *
 * This is a blocking function.
 *
 * \param dev_handle a device handle
 * \param endpoint the endpoint to clear halt status
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the endpoint does not exist
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns another LIBUSB_ERROR code on other failure
 */
func libusb_clear_halt(dev_handle *libusb_device_handle, endpoint uint8) int {
	// usbi_dbg("endpoint %x", endpoint)
	if !dev_handle.dev.attached {
		return LIBUSB_ERROR_NO_DEVICE
	}

	return usbi_backend.clear_halt(dev_handle, endpoint)
}

/** \ingroup libusb_dev
 * Perform a USB port reset to reinitialize a device. The system will attempt
 * to restore the previous configuration and alternate settings after the
 * reset has completed.
 *
 * If the reset fails, the descriptors change, or the previous state cannot be
 * restored, the device will appear to be disconnected and reconnected. This
 * means that the device handle is no longer valid (you should close it) and
 * rediscover the device. A return code of LIBUSB_ERROR_NOT_FOUND indicates
 * when this is the case.
 *
 * This is a blocking function which usually incurs a noticeable delay.
 *
 * \param dev_handle a handle of the device to reset
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if re-enumeration is required, or if the
 * device has been disconnected
 * \returns another LIBUSB_ERROR code on other failure
 */
func libusb_reset_device(dev_handle *libusb_device_handle) int {
	// usbi_dbg("")
	if !dev_handle.dev.attached {
		return LIBUSB_ERROR_NO_DEVICE
	}

	return usbi_backend.reset_device(dev_handle)
}

/** \ingroup libusb_asyncio
 * Allocate up to num_streams usb bulk streams on the specified endpoints. This
 * function takes an array of endpoints rather then a single endpoint because
 * some protocols require that endpoints are setup with similar stream ids.
 * All endpoints passed in must belong to the same interface.
 *
 * Note this function may return less streams then requested. Also note that the
 * same number of streams are allocated for each endpoint in the endpoint array.
 *
 * Stream id 0 is reserved, and should not be used to communicate with devices.
 * If libusb_alloc_streams() returns with a value of N, you may use stream ids
 * 1 to N.
 *
 * Since version 1.0.19, \ref LIBUSB_API_VERSION >= 0x01000103
 *
 * \param dev_handle a device handle
 * \param num_streams number of streams to try to allocate
 * \param endpoints array of endpoints to allocate streams on
 * \param num_endpoints length of the endpoints array
 * \returns number of streams allocated, or a LIBUSB_ERROR code on failure
 */
func libusb_alloc_streams(dev_handle *libusb_device_handle, num_streams uint32, endpoints []uint8, num_endpoints int) int {
	// usbi_dbg("streams %u eps %d", (unsigned) num_streams, num_endpoints)

	if !dev_handle.dev.attached {
		return LIBUSB_ERROR_NO_DEVICE
	}

	if usbi_backend.alloc_streams {
		return usbi_backend.alloc_streams(dev_handle, num_streams, endpoints, num_endpoints)
	}
	return LIBUSB_ERROR_NOT_SUPPORTED
}

/** \ingroup libusb_asyncio
 * Free usb bulk streams allocated with libusb_alloc_streams().
 *
 * Note streams are automatically free-ed when releasing an interface.
 *
 * Since version 1.0.19, \ref LIBUSB_API_VERSION >= 0x01000103
 *
 * \param dev_handle a device handle
 * \param endpoints array of endpoints to free streams on
 * \param num_endpoints length of the endpoints array
 * \returns LIBUSB_SUCCESS, or a LIBUSB_ERROR code on failure
 */
func libusb_free_streams(dev_handle *libusb_device_handle,
	endpoints []uint8, num_endpoints int) int {
	// usbi_dbg("eps %d", num_endpoints)

	if !dev_handle.dev.attached {
		return LIBUSB_ERROR_NO_DEVICE
	}

	if usbi_backend.free_streams {
		return usbi_backend.free_streams(dev_handle, endpoints, num_endpoints)
	}
	return LIBUSB_ERROR_NOT_SUPPORTED
}

/** \ingroup libusb_asyncio
 * Attempts to allocate a block of persistent DMA memory suitable for transfers
 * against the given device. If successful, will return a block of memory
 * that is suitable for use as "buffer" in \ref libusb_transfer against this
 * device. Using this memory instead of regular memory means that the host
 * controller can use DMA directly into the buffer to increase performance, and
 * also that transfers can no longer fail due to kernel memory fragmentation.
 *
 * Note that this means you should not modify this memory (or even data on
 * the same cache lines) when a transfer is in progress, although it is legal
 * to have several transfers going on within the same memory block.
 *
 * Will return NULL on failure. Many systems do not support such zerocopy
 * and will always return NULL. Memory allocated with this function must be
 * freed with \ref libusb_dev_mem_free. Specifically, this means that the
 * flag \ref LIBUSB_TRANSFER_FREE_BUFFER cannot be used to free memory allocated
 * with this function.
 *
 * Since version 1.0.21, \ref LIBUSB_API_VERSION >= 0x01000105
 *
 * \param dev_handle a device handle
 * \param length size of desired data buffer
 * \returns a pointer to the newly allocated memory, or NULL on failure
 */

func libusb_dev_mem_alloc(dev_handle *libusb_device_handle, length int) []uint8 {
	if !dev_handle.dev.attached {
		return nil
	}

	if usbi_backend.dev_mem_alloc {
		return usbi_backend.dev_mem_alloc(dev_handle, length)
	}
	return nil
}

/** \ingroup libusb_asyncio
 * Free device memory allocated with libusb_dev_mem_alloc().
 *
 * \param dev_handle a device handle
 * \param buffer pointer to the previously allocated memory
 * \param length size of previously allocated memory
 * \returns LIBUSB_SUCCESS, or a LIBUSB_ERROR code on failure
 */
func libusb_dev_mem_free(dev_handle *libusb_device_handle, buffer []uint8, length int) int {
	if usbi_backend.dev_mem_free {
		return usbi_backend.dev_mem_free(dev_handle, buffer, length)
	}
	return LIBUSB_ERROR_NOT_SUPPORTED
}

/** \ingroup libusb_dev
 * Determine if a kernel driver is active on an interface. If a kernel driver
 * is active, you cannot claim the interface, and libusb will be unable to
 * perform I/O.
 *
 * This functionality is not available on Windows.
 *
 * \param dev_handle a device handle
 * \param interface_number the interface to check
 * \returns 0 if no kernel driver is active
 * \returns 1 if a kernel driver is active
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns LIBUSB_ERROR_NOT_SUPPORTED on platforms where the functionality
 * is not available
 * \returns another LIBUSB_ERROR code on other failure
 * \see libusb_detach_kernel_driver()
 */
func libusb_kernel_driver_active(dev_handle *libusb_device_handle, interface_number int) int {
	// usbi_dbg("interface %d", interface_number)

	if !dev_handle.dev.attached {
		return LIBUSB_ERROR_NO_DEVICE
	}

	if usbi_backend.kernel_driver_active {
		return usbi_backend.kernel_driver_active(dev_handle, interface_number)
	}
	return LIBUSB_ERROR_NOT_SUPPORTED
}

/** \ingroup libusb_dev
 * Detach a kernel driver from an interface. If successful, you will then be
 * able to claim the interface and perform I/O.
 *
 * This functionality is not available on Darwin or Windows.
 *
 * Note that libusb itself also talks to the device through a special kernel
 * driver, if this driver is already attached to the device, this call will
 * not detach it and return LIBUSB_ERROR_NOT_FOUND.
 *
 * \param dev_handle a device handle
 * \param interface_number the interface to detach the driver from
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if no kernel driver was active
 * \returns LIBUSB_ERROR_INVALID_PARAM if the interface does not exist
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns LIBUSB_ERROR_NOT_SUPPORTED on platforms where the functionality
 * is not available
 * \returns another LIBUSB_ERROR code on other failure
 * \see libusb_kernel_driver_active()
 */
func libusb_detach_kernel_driver(dev_handle *libusb_device_handle, interface_number int) int {
	// usbi_dbg("interface %d", interface_number)

	if !dev_handle.dev.attached {
		return LIBUSB_ERROR_NO_DEVICE
	}

	if usbi_backend.detach_kernel_driver {
		return usbi_backend.detach_kernel_driver(dev_handle, interface_number)
	}
	return LIBUSB_ERROR_NOT_SUPPORTED
}

/** \ingroup libusb_dev
 * Re-attach an interface's kernel driver, which was previously detached
 * using libusb_detach_kernel_driver(). This call is only effective on
 * Linux and returns LIBUSB_ERROR_NOT_SUPPORTED on all other platforms.
 *
 * This functionality is not available on Darwin or Windows.
 *
 * \param dev_handle a device handle
 * \param interface_number the interface to attach the driver from
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if no kernel driver was active
 * \returns LIBUSB_ERROR_INVALID_PARAM if the interface does not exist
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns LIBUSB_ERROR_NOT_SUPPORTED on platforms where the functionality
 * is not available
 * \returns LIBUSB_ERROR_BUSY if the driver cannot be attached because the
 * interface is claimed by a program or driver
 * \returns another LIBUSB_ERROR code on other failure
 * \see libusb_kernel_driver_active()
 */
func libusb_attach_kernel_driver(dev_handle *libusb_device_handle, interface_number int) int {
	// usbi_dbg("interface %d", interface_number)

	if !dev_handle.dev.attached {
		return LIBUSB_ERROR_NO_DEVICE
	}

	if usbi_backend.attach_kernel_driver {
		return usbi_backend.attach_kernel_driver(dev_handle, interface_number)
	}
	return LIBUSB_ERROR_NOT_SUPPORTED
}

/** \ingroup libusb_dev
 * Enable/disable libusb's automatic kernel driver detachment. When this is
 * enabled libusb will automatically detach the kernel driver on an interface
 * when claiming the interface, and attach it when releasing the interface.
 *
 * Automatic kernel driver detachment is disabled on newly opened device
 * handles by default.
 *
 * On platforms which do not have LIBUSB_CAP_SUPPORTS_DETACH_KERNEL_DRIVER
 * this function will return LIBUSB_ERROR_NOT_SUPPORTED, and libusb will
 * continue as if this function was never called.
 *
 * \param dev_handle a device handle
 * \param enable whether to enable or disable auto kernel driver detachment
 *
 * \returns LIBUSB_SUCCESS on success
 * \returns LIBUSB_ERROR_NOT_SUPPORTED on platforms where the functionality
 * is not available
 * \see libusb_claim_interface()
 * \see libusb_release_interface()
 * \see libusb_set_configuration()
 */
func libusb_set_auto_detach_kernel_driver(dev_handle *libusb_device_handle, enable int) {
	if (usbi_backend.caps & USBI_CAP_SUPPORTS_DETACH_KERNEL_DRIVER) == 0 {
		return LIBUSB_ERROR_NOT_SUPPORTED
	}

	dev_handle.auto_detach_kernel_driver = enable
	return LIBUSB_SUCCESS
}

/** \ingroup libusb_lib
 * Set log message verbosity.
 *
 * The default level is LIBUSB_LOG_LEVEL_NONE, which means no messages are ever
 * printed. If you choose to increase the message verbosity level, ensure
 * that your application does not close the stdout/stderr file descriptors.
 *
 * You are advised to use level LIBUSB_LOG_LEVEL_WARNING. libusb is conservative
 * with its message logging and most of the time, will only log messages that
 * explain error conditions and other oddities. This will help you debug
 * your software.
 *
 * If the LIBUSB_DEBUG environment variable was set when libusb was
 * initialized, this function does nothing: the message verbosity is fixed
 * to the value in the environment variable.
 *
 * If libusb was compiled without any message logging, this function does
 * nothing: you'll never get any messages.
 *
 * If libusb was compiled with verbose debug message logging, this function
 * does nothing: you'll always get messages from all levels.
 *
 * \param ctx the context to operate on, or nil for the default context
 * \param level debug level to set
 */
func libusb_set_debug(ctx *libusb_context, level int) {
	ctx = USBI_GET_CONTEXT(ctx)
	if !ctx.debug_fixed {
		ctx.debug = level
	}
}

func usbi_disconnect_device(dev *libusb_device) {
	ctx := dev.ctx

	dev.lock.Lock()
	dev.attached = 0
	dev.lock.Unlock()

	ctx.usb_devs_lock.Lock()
	list_del(&dev.list)
	ctx.usb_devs_lock.Unlock()

	/* Signal that an event has occurred for this device if we support hotplug AND
	 * the hotplug message list is ready. This prevents an event from getting raised
	 * during initial enumeration. libusb_handle_events will take care of dereferencing
	 * the device. */
	if libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG) && dev.ctx.hotplug_msgs.next != nil {
		usbi_hotplug_notification(ctx, dev, LIBUSB_HOTPLUG_EVENT_DEVICE_LEFT)
	}
}

/* Perform some final sanity checks on a newly discovered device. If this
 * function fails (negative return code), the device should not be added
 * to the discovered device list. */
func usbi_sanitize_device(dev *libusb_device) int {
	r := usbi_device_cache_descriptor(dev)
	if r < 0 {
		return r
	}

	num_configurations := dev.device_descriptor.bNumConfigurations
	if num_configurations > USB_MAXCONFIG {
		// usbi_err(dev), "too many configurations".ctx
		return LIBUSB_ERROR_IO
	} //else if 0 == num_configurations
	// usbi_dbg("zero configurations, maybe an unauthorized device")

	dev.num_configurations = num_configurations
	return 0
}

/* Examine libusb's internal list of known devices, looking for one with
 * a specific session ID. Returns the matching device if it was found, and
 * nil otherwise. */
func usbi_get_device_by_session_id(ctx *libusb_context, session_id uint64) *libusb_device {
	var ret *libusb_device
	ctx.usb_devs_lock.Lock()
	for pos := list_entry((&ctx.usb_devs).next, list, libusb_device); &dev.libusb_device != (&ctx.usb_devs); dev = list_entry(dev.libusb_device.next, list, libusb_device) {
		if dev.session_data == session_id {
			ret = libusb_ref_device(dev)
			break
		}
	}
	ctx.usb_devs_lock.Unlock()

	return ret
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
 * nil-terminated.
 *
 * \param ctx the context to operate on, or nil for the default context
 * \param list output location for a list of devices. Must be later freed with
 * libusb_free_device_list().
 * \returns the number of devices in the outputted list, or any
 * \ref libusb_error according to errors encountered by the backend.
 */
func libusb_get_device_list(ctx *libusb_context, list ***libusb_device) int {
	discdevs := discovered_devs_alloc()
	var r, i, ln int
	ctx = USBI_GET_CONTEXT(ctx)

	if libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG) {
		/* backend provides hotplug support */
		var dev *libusb_device

		if usbi_backend.hotplug_poll != nil {
			usbi_backend.hotplug_poll()
		}

		ctx.usb_devs_lock.Lock()

		for dev = list_entry((&ctx.usb_devs).next, list, libusb_device); &dev.libusb_device != (&ctx.usb_devs); dev = list_entry(dev.libusb_device.next, list, libusb_device) {
			discdevs = discovered_devs_append(discdevs, dev)
		}

		ctx.usb_devs_lock.Unlock()
	} else {
		/* backend does not provide hotplug support */
		r = usbi_backend.get_device_list(ctx, &discdevs)
	}

	if r < 0 {
		if discdevs != nil {
			discovered_devs_free(discdevs)
		}
		return r
	}

	/* convert discovered_devs into a list */
	ln = discdevs.len
	ret := make([]*libusb_device, len+1)

	for i := 0; i < ln; i++ {
		ret[i] = libusb_ref_device(discdevs.devices[i])
	}
	*list = ret

	if discdevs != nil {
		discovered_devs_free(discdevs)
	}
	return ln
}

/** \ingroup libusb_dev
 * Frees a list of devices previously discovered using
 * libusb_get_device_list(). If the unref_devices parameter is set, the
 * reference count of each device in the list is decremented by 1.
 * \param list the list to free
 * \param unref_devices whether to unref the devices in the list
 */
func libusb_free_device_list(list **libusb_device, unref_devices int) {
	if list == nil {
		return
	}

	if unref_devices != 0 {
		i := 0
		dev := list[i]
		for dev != nil {
			libusb_unref_device(dev)
			i++
		}
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
func libusb_get_port_numbers(dev *libusb_device, port_numbers []uint8) int {
	ctx := dev.ctx
	i := len(port_numbers)

	// HCDs can be listed as devices with port #0
	for dev != nil && dev.port_number != 0 {
		port_numbers[i] = dev.port_number
		dev = dev.parent_dev
	}
	if i < len(port_numbers) {
		copy(port_numbers, port_numbers[i:])
	}
	return len(port_numbers) - i
}

/** \ingroup libusb_dev
 * Deprecated please use libusb_get_port_numbers instead.
 */
func libusb_get_port_path(ctx *libusb_context, dev *libusb_device, port_numbers []uint8) int {
	return libusb_get_port_numbers(dev, port_numbers)
}

func find_endpoint(config *libusb_config_descriptor, endpoint uint8) *libusb_endpoint_descriptor {
	for iface_idx := 0; iface_idx < config.bNumInterfaces; iface_idx++ {
		iface := &config.iface[iface_idx]
		for altsetting_idx := 0; altsetting_idx < iface.num_altsetting; altsetting_idx++ {
			altsetting := &iface.altsetting[altsetting_idx]
			for ep_idx := 0; ep_idx < altsetting.bNumEndpoints; ep_idx++ {
				ep := &altsetting.endpoint[ep_idx]
				if ep.bEndpointAddress == endpoint {
					return ep
				}
			}
		}
	}
	return nil
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
func libusb_get_max_packet_size(dev *libusb_device, endpoint uint8) int {
	var config libusb_config_descriptor

	r := libusb_get_active_config_descriptor(dev, &config)
	if r < 0 {
		// usbi_err(dev.ctx,
		// "could not retrieve active config descriptor")
		return LIBUSB_ERROR_OTHER
	}

	ep := find_endpoint(config, endpoint)
	if !ep {
		return LIBUSB_ERROR_NOT_FOUND
	}

	r = ep.wMaxPacketSize

	return r
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
func libusb_get_max_iso_packet_size(dev *libusb_device, endpoint uint8) int {
	var config libusb_config_descriptor

	r := libusb_get_active_config_descriptor(dev, &config)
	if r < 0 {
		// usbi_err(dev.ctx,
		// "could not retrieve active config descriptor")
		return LIBUSB_ERROR_OTHER
	}

	ep := find_endpoint(config, endpoint)
	if ep == nil {
		return LIBUSB_ERROR_NOT_FOUND
	}

	val := ep.wMaxPacketSize
	ep_type := libusb_transfer_type((ep.bmAttributes & 0x3))

	r = val & 0x07ff
	if ep_type == LIBUSB_TRANSFER_TYPE_ISOCHRONOUS || ep_type == LIBUSB_TRANSFER_TYPE_INTERRUPT {
		r *= (1 + ((val >> 11) & 3))
	}

	return r
}

/** \ingroup libusb_dev
 * Increment the reference count of a device.
 * \param dev the device to reference
 * \returns the same device
 */
func libusb_ref_device(libusb_device *dev) *libusb_device {
	dev.lock.Lock()
	dev.refcnt++
	dev.lock.Unlock()
	return dev
}

/** \ingroup libusb_dev
 * Decrement the reference count of a device. If the decrement operation
 * causes the reference count to reach zero, the device shall be destroyed.
 * \param dev the device to unreference
 */
func libusb_unref_device(dev *libusb_device) {

	if dev == nil {
		return
	}

	dev.lock.Lock()
	dev.refcnt--
	refcnt := dev.refcnt
	dev.lock.Unlock()

	if refcnt == 0 {
		// usbi_dbg("destroy device %d.%d", dev.bus_number, dev.device_address)

		libusb_unref_device(dev.parent_dev)

		if usbi_backend.destroy_device != nil {
			usbi_backend.destroy_device(dev)
		}

		if !libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG) {
			/* backend does not support hotplug */
			usbi_disconnect_device(dev)
		}
	}
}

/*
 * Signal the event pipe so that the event handling thread will be
 * interrupted to process an internal event.
 */
func usbi_signal_event(ctx *libusb_context) int {
	var dummy uint8

	/* read some data on event pipe to clear it */
	r := usbi_write(ctx.event_pipe[1], &dummy, 1)
	if r != 1 {
		// usbi_warn(ctx, "internal signalling read failed")
		return LIBUSB_ERROR_IO
	}

	return 0
}

/*
 * Clear the event pipe so that the event handling will no longer be
 * interrupted.
 */
func usbi_clear_event(ctx *libusb_context) int {
	var dummy uint8

	/* read some data on event pipe to clear it */
	r := usbi_read(ctx.event_pipe[0], &dummy, 1)
	if r != 1 {
		// usbi_warn(ctx, "internal signalling read failed")
		return LIBUSB_ERROR_IO
	}

	return 0
}

/** \ingroup libusb_dev
 * Open a device and obtain a device handle. A handle allows you to perform
 * I/O on the device in question.
 *
 * Internally, this function adds a reference to the device and makes it
 * available to you through libusb_get_device(). This reference is removed
 * during libusb_close().
 *
 * This is a non-blocking function no requests are sent over the bus.
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
func libusb_open(dev *libusb_device, dev_handle **libusb_device_handle) int {
	ctx := dev.ctx
	priv_size := usbi_backend.device_handle_priv_size
	// usbi_dbg("open %d.%d", dev.bus_number, dev.device_address)

	if !dev.attached {
		return LIBUSB_ERROR_NO_DEVICE
	}

	_dev_handle := &libusb_device_handle{}
	_dev_handle.dev = libusb_ref_device(dev)
	_dev_handle.os_priv = make([]uint8, priv_size)

	r := usbi_backend.open(_dev_handle)
	if r < 0 {
		// usbi_dbg("open %d.%d returns %d", dev.bus_number, dev.device_address, r)
		libusb_unref_device(dev)
		return r
	}

	ctx.open_devs_lock.Lock()
	list_add(&_dev_handle.list, &ctx.open_devs)
	ctx.open_devs_lock.Unlock()
	*dev_handle = _dev_handle

	return 0
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
 * \param ctx the context to operate on, or nil for the default context
 * \param vendor_id the idVendor value to search for
 * \param product_id the idProduct value to search for
 * \returns a device handle for the first found device, or nil on error
 * or if the device could not be found. */

func libusb_open_device_with_vid_pid(ctx *libusb_context, vendor_id, product_id uint16) *libusb_device_handle {

	if libusb_get_device_list(ctx, &devs) < 0 {
		return nil
	}

	var found *libusb_device
	var devs **libusb_device
	var dev *libusb_device
	var dev_handle *libusb_device_handl
	var r int

	dev = devs[i]
	for i := 0; dev != nil; i++ {
		desc := libusb_get_device_descriptor(dev)
		if r < 0 {
			libusb_free_device_list(devs, 1)
			return dev_handle
		}
		if desc.idVendor == vendor_id && desc.idProduct == product_id {
			found = dev
			break
		}
		dev = devs[i]
	}

	if found != nil {
		r = libusb_open(found, &dev_handle)
		if r < 0 {
			dev_handle = nil
		}
	}

	libusb_free_device_list(devs, 1)
	return dev_handle
}

func do_close(ctx *libusb_context, dev_handle *libusb_device_handle) {
	/* remove any transfers in flight that are for this device */
	ctx.flying_transfers_lock.Lock()

	/* safe iteration because transfers may be being deleted */
	for itransfer, tmp := list_entry((&ctx.flying_transfers).next, usbi_transfer, list), list_entry(pos.member.next, usbi_transfer, list); &itransfer.list != (&ctx.flying_transfers); itransfer, tmp = tmp, list_entry(tmp.member.next, usbi_transfer, list) {

		transfer := itransfer.libusbTransfer

		if transfer.dev_handle != dev_handle {
			continue
		}

		// itransfer.lock.Lock()
		// if (itransfer.state_flags & USBI_TRANSFER_DEVICE_DISAPPEARED) == 0 {
		// usbi_err(ctx, "Device handle closed while transfer was still being processed, but the device is still connected as far as we know")

		// if (itransfer.state_flags & USBI_TRANSFER_CANCELLING)
		// usbi_warn(ctx, "A cancellation for an in-flight transfer hasn't completed but closing the device handle")
		// else
		// usbi_err(ctx, "A cancellation hasn't even been scheduled on the transfer for which the device is closing")
		// }
		// itransfer.lock.Unlock()

		/* remove from the list of in-flight transfers and make sure
		 * we don't accidentally use the device handle in the future
		 * (or that such accesses will be easily caught and identified as a crash)
		 */
		list_del(&itransfer.list)
		transfer.dev_handle = nil

		/* it is up to the user to free up the actual transfer struct.  this is
		 * just making sure that we don't attempt to process the transfer after
		 * the device handle is invalid
		 */
		// usbi_dbg("Removed transfer %p from the in-flight list because device handle %p closed",
		//  transfer, dev_handle)
	}
	ctx.flying_transfers_lock.Unlock()

	ctx.open_devs_lock.Lock()
	list_del(&dev_handle.list)
	ctx.open_devs_lock.Unlock()

	usbi_backend.close(dev_handle)
	libusb_unref_device(dev_handle.dev)
}

/** \ingroup libusb_dev
 * Close a device handle. Should be called on all open handles before your
 * application exits.
 *
 * Internally, this function destroys the reference that was added by
 * libusb_open() on the given device.
 *
 * This is a non-blocking function no requests are sent over the bus.
 *
 * \param dev_handle the device handle to close
 */
func libusb_close(libusb_device_handle *dev_handle) {
	if dev_handle == nil {
		return
	}
	// usbi_dbg("")

	ctx := dev_handle.dev.ctx
	handling_events := true
	pending_events := false

	/* Similarly to libusb_open(), we want to interrupt all event handlers
	 * at this point. More importantly, we want to perform the actual close of
	 * the device while holding the event handling lock (preventing any other
	 * thread from doing event handling) because we will be removing a file
	 * descriptor from the polling loop. If this is being called by the current
	 * event handler, we can bypass the interruption code because we already
	 * hold the event handling lock. */

	if !handling_events {
		/* Record that we are closing a device.
		 * Only signal an event if there are no prior pending events. */
		ctx.event_data_lock.Lock()
		pending_events = usbi_pending_events(ctx)
		ctx.device_close++
		if !pending_events {
			usbi_signal_event(ctx)
		}
		ctx.event_data_lock.Unlock()

		/* take event handling lock */
		libusb_lock_events(ctx)
	}

	/* Close the device */
	do_close(ctx, dev_handle)

	if !handling_events {
		/* We're done with closing this device.
		 * Clear the event pipe if there are no further pending events. */
		ctx.event_data_lock.Lock()
		ctx.device_close--
		pending_events = usbi_pending_events(ctx)
		if !pending_events {
			usbi_clear_event(ctx)
		}
		ctx.event_data_lock.Unlock()

		/* Release event handling lock and wake up event waiters */
		libusb_unlock_events(ctx)
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
func libusb_get_configuration(dev_handle *libusb_device_handle, config *int) int {
	r := LIBUSB_ERROR_NOT_SUPPORTED

	if usbi_backend.get_configuration != nil {
		r = usbi_backend.get_configuration(dev_handle, config)
	}

	if r == LIBUSB_ERROR_NOT_SUPPORTED {
		var tmp uint8
		// usbi_dbg("falling back to control message")
		r = libusb_control_transfer(dev_handle, LIBUSB_ENDPOINT_IN,
			LIBUSB_REQUEST_GET_CONFIGURATION, 0, 0, &tmp, 1, 1000)
		if r == 0 {
			// usbi_err(dev_handle), "zero bytes returned in ctrl transfer?".dev.ctx
			r = LIBUSB_ERROR_IO
		} else if r == 1 {
			r = 0
			*config = int(tmp)
		} //else {
		// 	// usbi_dbg("control failed, error %d", r)
		// }
	}

	// if (r == 0)
	// 	// usbi_dbg("active config %d", *config)

	return r
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
func libusb_set_configuration(dev_handle *libusb_device_handle, configuration int) int {
	return usbi_backend.set_configuration(dev_handle, configuration)
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
 * Claiming of interfaces is a purely logical operation it does not cause
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
func libusb_claim_interface(dev_handle *libusb_device_handle, interface_number int) int {

	// usbi_dbg("interface %d", interface_number)
	if interface_number >= USB_MAXINTERFACES {
		return LIBUSB_ERROR_INVALID_PARAM
	}

	if !dev_handle.dev.attached {
		return LIBUSB_ERROR_NO_DEVICE
	}

	dev_handle.lock.Lock()
	if dev_handle.claimed_interfaces&(1<<interface_number) != 0 {
		dev_handle.lock.Unlock()
		return r
	}

	r := usbi_backend.claim_interface(dev_handle, interface_number)
	if r == 0 {
		dev_handle.claimed_interfaces |= 1 << interface_number
	}

	dev_handle.lock.Unlock()
	return r
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
func libusb_release_interface(dev_handle *libusb_device_handle, interface_number int) int {

	// usbi_dbg("interface %d", interface_number)
	if interface_number >= USB_MAXINTERFACES {
		return LIBUSB_ERROR_INVALID_PARAM
	}

	&dev_handle.lock.Lock()
	if (dev_handle.claimed_interfaces & (1 << interface_number)) == 0 {
		dev_handle.lock.Unlock()
		return LIBUSB_ERROR_NOT_FOUND
	}

	r := usbi_backend.release_interface(dev_handle, interface_number)
	if r == 0 {
		dev_handle.claimed_interfaces &= ^(1 << interface_number)
	}

	dev_handle.lock.Unlock()
	return r
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
func libusb_set_interface_alt_setting(dev_handle *libusb_device_handle, interface_number, alternate_setting int) int {
	// usbi_dbg("interface %d altsetting %d",
	// interface_number, alternate_setting)
	if interface_number >= USB_MAXINTERFACES {
		return LIBUSB_ERROR_INVALID_PARAM
	}

	dev_handle.lock.Lock()
	if !dev_handle.dev.attached {
		dev_handle.lock.Unlock()
		return LIBUSB_ERROR_NO_DEVICE
	}

	if (dev_handle.claimed_interfaces & (1 << interface_number)) == 0 {
		dev_handle.lock.Unlock()
		return LIBUSB_ERROR_NOT_FOUND
	}
	dev_handle.lock.Unlock()

	return usbi_backend.set_interface_altsetting(dev_handle, interface_number, alternate_setting)
}

var initOnce = sync.Once{}
var timeStampOriginInit = sync.Once{}

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
func libusb_init(context **libusb_context) int {
	var r int

	default_context_lock.Lock()

	timeStampOriginInit.Do(func() {
		timestamp_origin = time.Now()
	})

	if context == nil && usbi_default_context != nil {
		// usbi_dbg("reusing default context")
		default_context_refcnt++
		default_context_lock.Unlock()
		return 0
	}

	ctx := &libusb_context{}

	dbg := os.GetEnv("LIBUSB_DEBUG")
	if dbg != "" {
		i, err := strconv.Atoi(dbg)
		if err != nil {
			ctx.debug = i
			ctx.debug_fixed = 1
		}
	}

	/* default context should be initialized before calling // usbi_dbg */
	if usbi_default_context == nil {
		usbi_default_context = ctx
		default_context_refcnt++
		// usbi_dbg("created default context")
	}

	list_init(ctx.usb_devs)
	list_init(ctx.open_devs)
	list_init(ctx.hotplug_cbs)

	active_contexts_lock.Lock()
	initOnce.Do(list_init(active_contexts_list))
	list_add(&ctx.list, &active_contexts_list)
	active_contexts_lock.Unlock()

	if usbi_backend.init != nil {
		r = usbi_backend.init(ctx)
		if r != 0 {
			goto err_free_ctx
		}
	}

	r = usbi_io_init(ctx)
	if r >= 0 {
		default_context_lock.Unlock()

		if context != nil {
			*context = ctx
		}

		return 0
	}

	if usbi_backend.exit != nil {
		usbi_backend.exit()
	}
err_free_ctx:
	if ctx == usbi_default_context {
		usbi_default_context = nil
		default_context_refcnt--
	}

	active_contexts_lock.Lock()
	list_del(&ctx.list)
	active_contexts_lock.Unlock()

	ctx.usb_devs_lock.Lock()

	var dev, next *libusb_device
	for dev, next := list_entry((&ctx.usb_devs).next, libusb_device, list), list_entry(dev.member.next, libusb_device, list); &dev.list != (&ctx.usb_devs); dev, next = next, list_entry(next.member.next, libusb_device, list) {
		list_del(&dev.list)
		libusb_unref_device(dev)
	}
	ctx.usb_devs_lock.Unlock()

	default_context_lock.Unlock()
	return r
}
