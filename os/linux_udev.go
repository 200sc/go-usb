package os

/*
 * Linux usbfs backend for libusb
 * Copyright (C) 2007-2009 Daniel Drake <dsd@gentoo.org>
 * Copyright (c) 2001 Johannes Erdfelt <johannes@erdfelt.com>
 * Copyright (c) 2012-2013 Nathan Hjelm <hjelmn@mac.com>
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


 // **************
 // todo for this file to work
 // make go-libudev
 // **************


/* udev context */
var udev *udev_ctx
var udev_monitor_fd int = -1
var udev_control_pipe [2]int = {-1, -1}
// Go: renamed from udev_monitor
var _udev_monitor *udev_monitor

var udev_hotplug_event func(*udev_device)

func linux_udev_start_event_monitor() int {
	
	if udev_ctx != nil {
		panic("assert(udev_ctx == nil)")
	}

	udev_ctx = udev_new()
	if udev_ctx == nil {
		// usbi_err(nil, "could not create udev context")
		goto err
	}

	_udev_monitor = udev_monitor_new_from_netlink(udev_ctx, "udev")
	if _udev_monitor == nil {
		// usbi_err(nil, "could not initialize udev monitor")
		goto err_free_ctx
	}

	r := udev_monitor_filter_add_match_subsystem_devtype(_udev_monitor, "usb", "usb_device")
	if r != 0 {
		// usbi_err(nil, "could not initialize udev monitor filter for \"usb\" subsystem")
		goto err_free_monitor
	}

	if udev_monitor_enable_receiving(_udev_monitor) {
		// usbi_err(nil, "failed to enable the udev monitor")
		goto err_free_monitor
	}

	udev_monitor_fd = udev_monitor_get_fd(_udev_monitor)

	/* Some older versions of udev are not non-blocking by default,
	 * so make sure this is set */
	r = fcntl(udev_monitor_fd, F_GETFL)
	if r == -1 {
		// usbi_err(nil, "getting udev monitor fd flags (%d)", errno)
		goto err_free_monitor
	}
	r = fcntl(udev_monitor_fd, F_SETFL, r | O_NONBLOCK)
	if r != 0 {
		// usbi_err(nil, "setting udev monitor fd flags (%d)", errno)
		goto err_free_monitor
	}

	r = usbi_pipe(udev_control_pipe)
	if r != 0 {
		// usbi_err(nil, "could not create udev control pipe")
		goto err_free_monitor
	}

	// Go todo: use a goroutine
	r = pthread_create(&linux_event_thread, nil, linux_udev_event_thread_main, nil)
	if r != 0 {
		// usbi_err(nil, "creating hotplug event thread (%d)", r)
		goto err_close_pipe
	}

	return LIBUSB_SUCCESS

err_close_pipe:
	close(udev_control_pipe[0])
	close(udev_control_pipe[1])
err_free_monitor:
	udev_monitor_unref(_udev_monitor)
	_udev_monitor = nil
	udev_monitor_fd = -1
err_free_ctx:
	udev_unref(udev_ctx)
err:
	udev_ctx = nil
	return LIBUSB_ERROR_OTHER
}

func linux_udev_stop_event_monitor() libusb_error {
	if udev_ctx == nil {
		panic("assert(udev_ctx != nil)")
	}
	if _udev_monitor == nil {
		panic("assert(udev_monitor != nil")
	}
	if udev_monitor_fd == -1 {
		panic("assert(udev_monitor_fd != -1")
	}

	/* Write some dummy data to the control pipe and
	 * wait for the thread to exit */
	var dummy rune	
	usbi_write(udev_control_pipe[1], &dummy, sizeof(dummy))
	// if (r <= 0) {
	// 	// usbi_warn(nil, "udev control pipe signal failed")
	// }
	pthread_join(linux_event_thread, nil)

	/* Release the udev monitor */
	udev_monitor_unref(_udev_monitor)
	_udev_monitor = nil
	udev_monitor_fd = -1

	/* Clean up the udev context */
	udev_unref(udev_ctx)
	udev_ctx = nil

	/* close and reset control pipe */
	close(udev_control_pipe[0])
	close(udev_control_pipe[1])
	udev_control_pipe[0] = -1
	udev_control_pipe[1] = -1

	return LIBUSB_SUCCESS
}

// Go change: this always returns NULL, so instead it's just 
// not returning anything
func linux_udev_event_thread_main() {
	var dummy rune
	fds := [2]pollfd{
		{	
			fd: udev_control_pipe[0],
			events: POLLIN,
		},
		{
			fd: udev_monitor_fd,
			 events: POLLIN,
		},
	}

	// usbi_dbg("udev event thread entering.")

	for poll(fds, 2, -1) >= 0 {
		if fds[0].revents & POLLIN != 0 {
			/* activity on control pipe, read the byte and exit */
			r := usbi_read(udev_control_pipe[0], &dummy, 1)
			// if (r <= 0) {
			// 	// usbi_warn(nil, "udev control pipe read failed")
			// }
			break
		}
		if fds[1].revents & POLLIN != 0 {
			linux_hotplug_lock.Lock()
			udev_dev := udev_monitor_receive_device(_udev_monitor)
			if udev_dev != nil {
				udev_hotplug_event(udev_dev)
			}
			linux_hotplug_lock.Unlock()
		}
	}

	// usbi_dbg("udev event thread exiting")
}

func udev_device_info(ctx *libusb_context, detached bool,
			    udev_dev *udev_device, busnum *uint8,
			    devaddr *uint8, sys_name *string) libusb_error {

	dev_node := udev_device_get_devnode(udev_dev)
	if dev_node == "" {
		return LIBUSB_ERROR_OTHER
	}

	*sys_name = udev_device_get_sysname(udev_dev)
	if *sys_name = "" {
		return LIBUSB_ERROR_OTHER
	}

	return linux_get_device_address(ctx, detached, busnum, devaddr, dev_node, *sys_name)
}

func linux_udev_scan_devices(ctx *libusb_context) libusb_error {
	var entry *udev_list_entry
	var udev_dev *udev_device 
	var sys_name string

	if udev_ctx == nil {
		panic("assert(udev_ctx != nil)")
	}

	enumerator := udev_enumerate_new(udev_ctx)
	if enumerator == nil {
		// usbi_err(ctx, "error creating udev enumerator")
		return LIBUSB_ERROR_OTHER
	}

	udev_enumerate_add_match_subsystem(enumerator, "usb")
	udev_enumerate_add_match_property(enumerator, "DEVTYPE", "usb_device")
	udev_enumerate_scan_devices(enumerator)
	devices := udev_enumerate_get_list_entry(enumerator)

	// Why does every lib have this garbage defined iterator 
	udev_list_entry_foreach(entry, devices) {
		path := udev_list_entry_get_name(entry)
		var busnum, devaddr uint8

		udev_dev = udev_device_new_from_syspath(udev_ctx, path)

		r := udev_device_info(ctx, 0, udev_dev, &busnum, &devaddr, &sys_name)
		if r != 0 {
			udev_device_unref(udev_dev)
			continue
		}

		linux_enumerate_device(ctx, busnum, devaddr, sys_name)
		udev_device_unref(udev_dev)
	}

	udev_enumerate_unref(enumerator)

	return LIBUSB_SUCCESS
}

func udev_hotplug_event(udev_dev *udev_device) {
	var sys_name string
	var busnum, devaddr uint8

	udev_action := udev_device_get_action(udev_dev)
	if (!udev_action) {
		udev_device_unref(udev_dev)
		return	 
	}

	detached := udev_action[:6] == "remove"

	r := udev_device_info(nil, detached, udev_dev, &busnum, &devaddr, &sys_name)
	if LIBUSB_SUCCESS != r {
		udev_device_unref(udev_dev)
		return
	}

	// usbi_dbg("udev hotplug event. action: %s.", udev_action)

	if udev_action[:3] == "add" {
		linux_hotplug_enumerate(busnum, devaddr, sys_name)
	} else if detached {
		linux_device_disconnected(busnum, devaddr)
	}// else {
		// usbi_err(nil, "ignoring udev action %s", udev_action)
	// }

	udev_device_unref(udev_dev)
}

func linux_udev_hotplug_poll() {
	linux_hotplug_lock.Lock()
	udev_dev := udev_monitor_receive_device(_udev_monitor)
	for udev_dev != nil {
		// usbi_dbg("Handling hotplug event from hotplug_poll")
		udev_hotplug_event(udev_dev)
		udev_dev = udev_monitor_receive_device(_udev_monitor)		
	}
	linux_hotplug_lock.Unlock()
}
