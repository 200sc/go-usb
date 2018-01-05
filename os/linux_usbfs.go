package os

/*
 * usbfs header structures
 * Copyright © 2007 Daniel Drake <dsd@gentoo.org>
 * Copyright © 2001 Johannes Erdfelt <johannes@erdfelt.com>
 *
* Linux usbfs backend for libusb
 * Copyright © 2007-2009 Daniel Drake <dsd@gentoo.org>
 * Copyright © 2001 Johannes Erdfelt <johannes@erdfelt.com>
 * Copyright © 2013 Nathan Hjelm <hjelmn@mac.com>
 * Copyright © 2012-2013 Hans de Goede <hdegoede@redhat.com>
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

type usbfs_urb_type uint8

const (
	USBFS_URB_TYPE_ISO usbfs_urb_type = iota
	USBFS_URB_TYPE_INTERRUPT usbfs_urb_type
	USBFS_URB_TYPE_CONTROL usbfs_urb_type
	USBFS_URB_TYPE_BULK usbfs_urb_type
)

type reap_action uint8
const (
	NORMAL reap_action = iota
	/* submission failed after the first URB, so await cancellation/completion
	 * of all the others */
	SUBMIT_FAILED reap_action

	/* cancelled by user or timeout */
	CANCELLED reap_action

	/* completed multi-URB transfer in non-final URB */
	COMPLETED_EARLY reap_action

	/* one or more urbs encountered a low-level error */
	ERROR reap_action
)