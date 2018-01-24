package usb

import (
	"sync"
	"time"
)

/*
 * Internal header for libusb
 * Copyright © 2007-2009 Daniel Drake <dsd@gentoo.org>
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

const (
	USB_MAXENDPOINTS  = 32
	USB_MAXINTERFACES = 32
	USB_MAXCONFIG     = 8

	/* Backend specific capabilities */
	USBI_CAP_HAS_HID_ACCESS                = 0x00010000
	USBI_CAP_SUPPORTS_DETACH_KERNEL_DRIVER = 0x00020000

	/* Maximum number of bytes in a log line */
	USBI_MAX_LOG_LEN = 1024
)

type usbi_event_flags uint8

type pollfd struct {
	fd      int    /* file descriptor */
	events  uint16 /* requested events */
	revents uint16 /* returned events */
}
type POLL_NFDS_TYPE uint32

const (
	/* The list of pollfds has been modified */
	USBI_EVENT_POLLFDS_MODIFIED usbi_event_flags = 1 << 0

	/* The user has interrupted the event handler */
	USBI_EVENT_USER_INTERRUPT usbi_event_flags = 1 << 1
)

type usbi_clock uint8

const (
	USBI_CLOCK_MONOTONIC usbi_clock = iota
	USBI_CLOCK_REALTIME  usbi_clock = iota
)

type usbi_transfer_state_flags uint8

const (
	/* Transfer successfully submitted by backend */
	USBI_TRANSFER_IN_FLIGHT usbi_transfer_state_flags = 1 << 0

	/* Cancellation was requested via libusb_cancel_transfer() */
	USBI_TRANSFER_CANCELLING usbi_transfer_state_flags = 1 << 1

	/* Operation on the transfer failed because the device disappeared */
	USBI_TRANSFER_DEVICE_DISAPPEARED usbi_transfer_state_flags = 1 << 2
)

type usbi_transfer_timeout_flags uint8

const (
	/* Set by backend submit_transfer() if the OS handles timeout */
	USBI_TRANSFER_OS_HANDLES_TIMEOUT usbi_transfer_timeout_flags = 1 << 0

	/* The transfer timeout has been handled */
	USBI_TRANSFER_TIMEOUT_HANDLED usbi_transfer_timeout_flags = 1 << 1

	/* The transfer timeout was successfully processed */
	USBI_TRANSFER_TIMED_OUT usbi_transfer_timeout_flags = 1 << 2
)

type libusb_context struct {
	debug       int
	debug_fixed bool

	/* internal event pipe, used for signalling occurrence of an internal event. */
	event_pipe [2]int

	usb_devs      *LinkedList
	usb_devs_lock sync.Mutex

	/* A list of open handles. Backends are free to traverse this if required.
	 */
	open_devs      *LinkedList
	open_devs_lock sync.Mutex

	/* A list of registered hotplug callbacks */
	hotplug_cbs      *LinkedList
	hotplug_cbs_lock sync.Mutex

	/* this is a list of in-flight transfer handles, sorted by timeout
	 * expiration. URBs to timeout the soonest are placed at the beginning of
	 * the list, URBs that will time out later are placed after, and urbs with
	 * infinite timeout are always placed at the very end. */
	flying_transfers *LinkedList
	/* Note paths taking both this and usbi_transfer->lock must always
	 * take this lock first */
	flying_transfers_lock sync.Mutex

	/* user callbacks for pollfd changes */
	fd_added_cb     libusb_pollfd_added_cb
	fd_removed_cb   libusb_pollfd_removed_cb
	fd_cb_user_data interface{}

	/* ensures that only one thread is handling events at any one time */
	events_lock sync.Mutex

	/* used to see if there is an active thread doing event handling */
	event_handler_active int

	/* A thread-local storage key to track which thread is performing event
	 * handling */
	//event_handling_key usbi_tls_key_t

	/* used to wait for event completion in threads other than the one that is
	 * event handling */
	event_waiters_lock sync.Mutex
	event_waiters_cond sync.Cond

	/* A lock to protect internal context event data. */
	event_data_lock sync.Mutex

	/* A bitmask of flags that are set to indicate specific events that need to
	 * be handled. Protected by event_data_lock. */
	event_flags uint

	/* A counter that is set when we want to interrupt and prevent event handling,
	 * in order to safely close a device. Protected by event_data_lock. */
	device_close uint

	/* list and count of poll fds and an array of poll fd structures that is
	 * (re)allocated as necessary prior to polling. Protected by event_data_lock. */
	ipollfds    *LinkedList
	pollfds     []pollfd
	pollfds_cnt POLL_NFDS_TYPE

	/* A list of pending hotplug messages. Protected by event_data_lock. */
	hotplug_msgs *LinkedList

	/* A list of pending completed transfers. Protected by event_data_lock. */
	completed_transfers *LinkedList

	/* used for timeout handling, if supported by OS.
	 * this timerfd is maintained to trigger on the next pending timeout */
	timerfd int

	list *LinkedList
}

type libusb_device struct {
	/* lock protects refcnt, everything else is finalized at initialization
	 * time */
	lock   sync.Mutex
	refcnt int

	ctx *libusb_context

	bus_number         uint8
	port_number        uint8
	parent_dev         *libusb_device
	device_address     uint8
	num_configurations uint8
	speed              libusb_speed

	list         *LinkedList
	session_data uint64

	device_descriptor libusb_device_descriptor
	attached          bool

	os_priv uint8
}

type libusb_device_handle struct {
	/* lock protects claimed_interfaces */
	lock               sync.Mutex
	claimed_interfaces uint64

	list                      *LinkedList
	dev                       *libusb_device
	auto_detach_kernel_driver int
	os_priv                   []uint8
}

/* in-memory transfer layout:
 *
 * 1. struct usbi_transfer
 * 2. struct libusb_transfer (which includes iso packets) [variable size]
 * 3. os private data [variable size]
 *
 * from a libusb_transfer, you can get the usbi_transfer by rewinding the
 * appropriate number of bytes.
 * the usbi_transfer includes the number of allocated packets, so you can
 * determine the size of the transfer and hence the start and length of the
 * OS-private data.
 */

type usbi_transfer struct {
	libusbTransfer  *libusb_transfer
	num_iso_packets int
	list            *LinkedList
	completed_list  *LinkedList
	timeout         time.Time
	transferred     int
	stream_id       uint32
	state_flags     uint8 /* Protected by usbi_transfer->lock */
	timeout_flags   uint8 /* Protected by the flying_stransfers_lock */

	/* this lock is held during libusb_submit_transfer() and
	 * libusb_cancel_transfer() (allowing the OS backend to prevent duplicate
	 * cancellation, submission-during-cancellation, etc). the OS backend
	 * should also take this lock in the handle_events path, to prevent the user
	 * cancelling the transfer from another thread while you are processing
	 * its completion (presumably there would be races within your OS backend
	 * if this were possible).
	 * Note paths taking both this and the flying_transfers_lock must
	 * always take the flying_transfers_lock first */
	lock  sync.Mutex
	tpriv interface{}
}

// this might need to be os-abstracted
func (usbt *usbi_transfer) usbi_transfer_get_os_priv() interface{} {
	return usbt.tpriv
}

/* All standard descriptors have these 2 fields in common */
type usb_descriptor_header struct {
	bLength         uint8
	bDescriptorType uint8
}

type usbi_pollfd struct {
	/* must come first */
	pollfd libusb_pollfd
	list   *LinkedList
}

/* device discovery */

/* we traverse usbfs without knowing how many devices we are going to find.
 * so we create this discovered_devs model which is similar to a linked-list
 * which grows when required. it can be freed once discovery has completed,
 * eliminating the need for a list node in the libusb_device structure
 * itself. */
type discovered_devs []*libusb_device

func IS_EPIN(ep uint8) bool {
	return ep&LIBUSB_ENDPOINT_IN != 0
}

func IS_XFERIN(xfer *libusb_transfer) bool {
	return xfer.endpoint&LIBUSB_ENDPOINT_IN != 0
}

func USBI_GET_CONTEXT(ctx *libusb_context) *libusb_context {
	if ctx == nil {
		return usbi_default_context
	}
	return ctx
}

/* Update the following macro if new event sources are added */
func usbi_pending_events(ctx *libusb_context) bool {
	return ctx.event_flags != nil ||
		ctx.device_close != nil ||
		!list_empty(ctx.hotplug_msgs) ||
		!list_empty(ctx.completed_transfers)
}
