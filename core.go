package usb

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
	if !dev_handle.dev.attached) {
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
return libusb_reset_device(dev_handle *libusb_device_handle) int {
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

	if (!dev_handle.dev.attached) {
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
	if !dev_handle.dev.attached) {
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

	if (!dev_handle.dev.attached)
		return LIBUSB_ERROR_NO_DEVICE

	if (usbi_backend.detach_kernel_driver)
		return usbi_backend.detach_kernel_driver(dev_handle, interface_number)
	else
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

	if (!dev_handle.dev.attached)
		return LIBUSB_ERROR_NO_DEVICE

	if (usbi_backend.attach_kernel_driver)
		return usbi_backend.attach_kernel_driver(dev_handle, interface_number)
	else
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
	if (usbi_backend.caps & USBI_CAP_SUPPORTS_DETACH_KERNEL_DRIVER) == 0{
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