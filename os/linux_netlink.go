package os

/*
 * Linux usbfs backend for libusb
 * Copyright (C) 2007-2009 Daniel Drake <dsd@gentoo.org>
 * Copyright (c) 2001 Johannes Erdfelt <johannes@erdfelt.com>
 * Copyright (c) 2013 Nathan Hjelm <hjelmn@mac.com>
 * Copyright (c) 2016 Chris Dickens <christopher.a.dickens@gmail.com>
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

const NL_GROUP_KERNEL = 1

var linux_netlink_socket int  = -1
var netlink_control_pipe [2]int = { -1, -1 }

func set_fd_cloexec_nb(fd int) int {
	flags := fcntl(fd, F_GETFL)
	if flags == -1 {
		// usbi_err(NULL, "failed to get netlink fd status flags (%d)", errno);
		return -1
	}

	if flags & O_NONBLOCK == 0 {
		if fcntl(fd, F_SETFL, flags | O_NONBLOCK) == -1 {
			// usbi_err(NULL, "failed to set netlink fd status flags (%d)", errno);
			return -1
		}
	}

	return 0
}

func linux_netlink_start_event_monitor() int {

	sa_nl := sockaddr_nl{
		nl_family: AF_NETLINK,
		nl_groups: NL_GROUP_KERNEL,
	}
	socktype := SOCK_RAW
	opt := 1

	linux_netlink_socket = socket(PF_NETLINK, socktype, NETLINK_KOBJECT_UEVENT)
	if linux_netlink_socket == -1 && errno == EINVAL {
		// usbi_dbg("failed to create netlink socket of type %d, attempting SOCK_RAW", socktype);
		linux_netlink_socket = socket(PF_NETLINK, SOCK_RAW, NETLINK_KOBJECT_UEVENT)
	}

	if linux_netlink_socket == -1 {
		// usbi_err(NULL, "failed to create netlink socket (%d)", errno);
		goto err
	}

	ret := set_fd_cloexec_nb(linux_netlink_socket)
	if ret == -1 {
		goto err_close_socket
	}

	ret = bind(linux_netlink_socket, &sa_nl, sizeof(sa_nl))
	if ret == -1 {
		// usbi_err(NULL, "failed to bind netlink socket (%d)", errno);
		goto err_close_socket
	}

	ret = setsockopt(linux_netlink_socket, SOL_SOCKET, SO_PASSCRED, &opt, sizeof(opt))
	if ret == -1 {
		// usbi_err(NULL, "failed to set netlink socket SO_PASSCRED option (%d)", errno);
		goto err_close_socket
	}

	ret = usbi_pipe(netlink_control_pipe);
	if ret {
		// usbi_err(NULL, "failed to create netlink control pipe");
		goto err_close_socket
	}

	// Go todo: goroutine
	ret = pthread_create(&libusb_linux_event_thread, NULL, linux_netlink_event_thread_main, NULL)
	if ret != 0 {
		// usbi_err(NULL, "failed to create netlink event thread (%d)", ret);
		goto err_close_pipe
	}

	return LIBUSB_SUCCESS

err_close_pipe:
	close(netlink_control_pipe[0])
	close(netlink_control_pipe[1])
	netlink_control_pipe[0] = -1
	netlink_control_pipe[1] = -1
err_close_socket:
	close(linux_netlink_socket)
	linux_netlink_socket = -1
err:
	return LIBUSB_ERROR_OTHER
}

func linux_netlink_stop_event_monitor() int {
	var dummy rune

	if linux_netlink_socket == -1 {
		panic("asset(linux_netlink_socket != -1)")
	}

	/* Write some dummy data to the control pipe and
	 * wait for the thread to exit */
	usbi_write(netlink_control_pipe[1], &dummy, 1)
	// if (r <= 0)
		// usbi_warn(NULL, "netlink control pipe signal failed");

	pthread_join(libusb_linux_event_thread, NULL)

	close(linux_netlink_socket)
	linux_netlink_socket = -1

	/* close and reset control pipe */
	close(netlink_control_pipe[0])
	close(netlink_control_pipe[1])
	netlink_control_pipe[0] = -1
	netlink_control_pipe[1] = -1

	return LIBUSB_SUCCESS
}

func netlink_message_parse(buffer string, ln int, key string) string {
	keylen := len(key)

	for i := 0; i < ln; i++ {
		if buffer[i:i+keylen] == key && buffer[i+keylen] == '=' {
			return buffer[i+keylen+1]
		}
		// What???
		i += len(buffer) + 1 
	}

	return ""
}

/* parse parts of netlink message common to both libudev and the kernel */
func linux_netlink_parse(buffer string, ln int, detached *int,
	sys_name *string, busnum, devaddr *uint8) int {

	errno = 0

	*sys_name = NULL
	*detached = false
	*busnum   = 0
	*devaddr  = 0

	tmp := netlink_message_parse(buffer, ln, "ACTION")
	if tmp == "" {
		return -1
	} else if tmp == "remove" {
		*detached = true
	} else if tmp == "add" {
		// usbi_dbg("unknown device action %s", tmp);
		return -1
	}

	/* check that this is a usb message */
	tmp = netlink_message_parse(buffer, len, "SUBSYSTEM")
	if tmp != "usb" {
		/* not usb. ignore */
		return -1
	}

	/* check that this is an actual usb device */
	tmp = netlink_message_parse(buffer, len, "DEVTYPE")
	if tmp != "usb_device" {
		/* not usb. ignore */
		return -1
	}

	tmp = netlink_message_parse(buffer, len, "BUSNUM")
	if tmp != "" {
		n, err := strconv.Atoi(tmp)
		if err != nil {
			return -1 // I guess that's how we're reporting errors here
		}
		*busnum = uint8(n) & 0xff
		if errno != 0 {
			errno = 0
			return -1
		}

		tmp = netlink_message_parse(buffer, len, "DEVNUM")
		if tmp == "" {
			return -1
		}

		*devaddr = *busnum
	} else {
		/* no bus number. try "DEVICE" */
		tmp = netlink_message_parse(buffer, len, "DEVICE")
		if tmp == "" {
			/* not usb. ignore */
			return -1
		}

		/* Parse a device path such as /dev/bus/usb/003/004 */
		slashIndex := strings.LastIndex(tmp, "/")
		if slashIndex < 0 {
			return -1
		}

		n, err := strconv.Atoi(tmp[slashIndex-3:])
		if err != nil {
			errno = 0
			return -1
		}
		*busnum = uint8(n) & oxff

		n, err = strconv.Atoi(tmp[slashIndex+1:])
		if err != nil {
			errno = 0
			return -1
		}
		*devaddr = uint8(n) & oxff

		return 0
	}

	tmp = netlink_message_parse(buffer, len, "DEVPATH")
	if tmp == nil {
		return -1
	}

	sys_name = tmp[:strings.LastIndex(tmp, "/")]

	/* found a usb device */
	return 0
}

func linux_netlink_read_message() int {
	// todo: what is a ucred
	cred_buffer := make([]rune, sizeof(ucred))
	msg_buffer := make([]rune, 2048)
	var sys_name string
	var busnum, devaddr uint8
	var detached bool

	var sa_nl sockaddr_nl
	iov := iovec{iov_base: msg_buffer, iov_len: 2048 }

	msg := msghdr{
		msg_iov: &iov,
		msg_iovlen: 1,
		msg_control: cred_buffer,
		msg_controllen: = len(cred_buffer),
		msg_name: &sa_nl,
		msg_namelen: = sizeof(sa_nl)
	}

	/* read netlink message */
	ln := recvmsg(linux_netlink_socket, &msg, 0)
	if ln == -1 {
		// if (errno != EAGAIN && errno != EINTR)
			// usbi_err(NULL, "error receiving message from netlink (%d)", errno);
		return -1
	}

	if ln < 32 || (msg.msg_flags & MSG_TRUNC) != 0 {
		// usbi_err(NULL, "invalid netlink message length");
		return -1
	}

	if sa_nl.nl_groups != NL_GROUP_KERNEL || sa_nl.nl_pid != 0 {
		// usbi_dbg("ignoring netlink message from unknown group/PID (%u/%u)",
			//  (uint)sa_nl.nl_groups, (uint)sa_nl.nl_pid)
		return -1
	}

	cmsg := CMSG_FIRSTHDR(&msg)
	if (!cmsg || cmsg.cmsg_type != SCM_CREDENTIALS) {
		// usbi_dbg("ignoring netlink message with no sender credentials");
		return -1;
	}

	// Go todo: this is gonna need work
	cred := CMSG_DATA(cmsg).(*ucred)
	if cred.uid != 0 {
		// usbi_dbg("ignoring netlink message with non-zero sender UID %u", (uint)cred.uid);
		return -1
	}

	r := linux_netlink_parse(msg_buffer, (int)len, &detached, &sys_name, &busnum, &devaddr);
	if r != 0 {
		return r
	}

	// usbi_dbg("netlink hotplug found device busnum: %hhu, devaddr: %hhu, sys_name: %s, removed: %s",
		//  busnum, devaddr, sys_name, detached ? "yes" : "no");

	/* signal device is available (or not) to all contexts */
	if detached {
		linux_device_disconnected(busnum, devaddr)
	} else {
		linux_hotplug_enumerate(busnum, devaddr, sys_name)
	}

	return 0
}

func linux_netlink_event_thread_main() {
	var dummy rune
	fds := [2]pollfd{
		{	
			fd: netlink_control_pipe[0],
			events: POLLIN,
		},
		{
			fd: linux_netlink_socket,
			 events: POLLIN,
		},
	}

	// usbi_dbg("netlink event thread entering");

	for poll(fds, 2, -1) >= 0 {
		if fds[0].revents & POLLIN {
			/* activity on control pipe, read the byte and exit */
			r := usbi_read(netlink_control_pipe[0], &dummy, sizeof(dummy));
			// if (r <= 0)
				// usbi_warn(NULL, "netlink control pipe read failed");
			break
		}
		if fds[1].revents & POLLIN != 0 {
			linux_hotplug_lock.Lock()
			linux_netlink_read_message()
			linux_hotplug_lock.Unlock()
		}
	}
	// usbi_dbg("netlink event thread exiting");
}

func linux_netlink_hotplug_poll() {
	var r libusb_error
	linux_hotplug_lock.Lock()
	for r == 0 {
		r = linux_netlink_read_message()
	} 
	linux_hotplug_lock.Unlock()
}
