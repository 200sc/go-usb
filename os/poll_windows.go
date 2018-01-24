package os

/*
 * Windows compat: POSIX compatibility wrapper
 * Copyright © 2012-2013 RealVNC Ltd.
 * Copyright © 2009-2010 Pete Batard <pete@akeo.ie>
 * With contributions from Michael Plante, Orin Eman et al.
 * Parts of poll implementation from libusb-win32, by Stephan Meyer et al.
 *
 * poll_windows: poll compatibility wrapper for Windows
 * Copyright © 2012-2013 RealVNC Ltd.
 * Copyright © 2009-2010 Pete Batard <pete@akeo.ie>
 * With contributions from Michael Plante, Orin Eman et al.
 * Parts of poll implementation from libusb-win32, by Stephan Meyer et al.
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
 *
 */


type windows_version int
const (
	WINDOWS_CE windows_version = -2,
	WINDOWS_UNDEFINED windows_version = -1,
	WINDOWS_UNSUPPORTED windows_version = 0,
	WINDOWS_XP windows_version = 0x51,
	WINDOWS_2003 windows_version = 0x52,	// Also XP x64
	WINDOWS_VISTA windows_version = 0x60,
	WINDOWS_7 windows_version = 0x61,
	WINDOWS_8 windows_version = 0x62,
	WINDOWS_8_1_OR_LATER windows_version = 0x63,
	WINDOWS_MAX windows_version = 0x64
)

// access modes
type rw_type uint8 
const (
	RW_NONE rw_type = iota
	RW_READ rw_type
	RW_WRITE rw_type
)

// Handle synchronous completion through the overlapped structure
const (
	STATUS_REPARSE = 0x00000104L
	STATUS_COMPLETED_SYNCHRONOUSLY = STATUS_REPARSE
	MAX_FDS = 256
	POLLIN =     0x0001    /* There is data to read */
	POLLPRI =    0x0002    /* There is urgent data to read */
	POLLOUT =    0x0004    /* Writing now will not block */
	POLLERR =    0x0008    /* Error condition */
	POLLHUP =    0x0010    /* Hung up */
	POLLNVAL =   0x0020    /* Invalid request: fd not open */
	DUMMY_HANDLE HANDLE = -2
)

type cancel_transfer func(*usbi_transfer) int

type winfd struct {
	fd int					// what's exposed to libusb core
	handle HANDLE					// what we need to attach overlapped to the I/O op, so we can poll it
	overlapped *OVERLAPPED			// what will report our I/O status
	itransfer *usbi_transfer		// Associated transfer, or NULL if completed
	cancel_fn cancel_transfer		// Function pointer to cancel transfer API
	rw rw_type				// I/O transfer direction: read *XOR* write (NOT BOTH)
}