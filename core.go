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
