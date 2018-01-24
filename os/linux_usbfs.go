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


const (
	SYSFS_DEVICE_PATH "/sys/bus/usb/devices"
	USBFS_MAXDRIVERNAME 255
	USBFS_URB_SHORT_NOT_OK		0x01
	USBFS_URB_ISO_ASAP			0x02
	USBFS_URB_BULK_CONTINUATION	0x04
	USBFS_URB_QUEUE_BULK		0x10
	USBFS_URB_ZERO_PACKET		0x40
	MAX_ISO_BUFFER_LENGTH		49152 * 128
	MAX_BULK_BUFFER_LENGTH		16384
	MAX_CTRL_BUFFER_LENGTH		4096
	USBFS_CAP_ZERO_PACKET		0x01
	USBFS_CAP_BULK_CONTINUATION	0x02
	USBFS_CAP_NO_PACKET_SIZE_LIM	0x04
	USBFS_CAP_BULK_SCATTER_GATHER	0x08
	USBFS_CAP_REAP_AFTER_DISCONNECT	0x10
	USBFS_DISCONNECT_CLAIM_IF_DRIVER	0x01
	USBFS_DISCONNECT_CLAIM_EXCEPT_DRIVER	0x02
)
var (
	IOCTL_USBFS_CONTROL	= _IOWR('U', 0, struct usbfs_ctrltransfer)
	IOCTL_USBFS_BULK		= _IOWR('U', 2, struct usbfs_bulktransfer)
	IOCTL_USBFS_RESETEP	= _IOR('U', 3, uint)
	IOCTL_USBFS_SETINTF	= _IOR('U', 4, struct usbfs_setinterface)
	IOCTL_USBFS_SETCONFIG	= _IOR('U', 5, uint)
	IOCTL_USBFS_GETDRIVER	_IOW('U', 8, struct usbfs_getdriver)
	IOCTL_USBFS_SUBMITURB	= _IOR('U', 10, struct usbfs_urb)
	IOCTL_USBFS_DISCARDURB	_IO('U', 11)
	IOCTL_USBFS_REAPURB	_IOW('U', 12, void *)
	IOCTL_USBFS_REAPURBNDELAY	_IOW('U', 13, void *)
	IOCTL_USBFS_CLAIMINTF	= _IOR('U', 15, uint)
	IOCTL_USBFS_RELEASEINTF	= _IOR('U', 16, uint)
	IOCTL_USBFS_CONNECTINFO	_IOW('U', 17, struct usbfs_connectinfo)
	IOCTL_USBFS_IOCTL         = _IOWR('U', 18, struct usbfs_ioctl)
	IOCTL_USBFS_HUB_PORTINFO	= _IOR('U', 19, struct usbfs_hub_portinfo)
	IOCTL_USBFS_RESET		_IO('U', 20)
	IOCTL_USBFS_CLEAR_HALT	= _IOR('U', 21, uint)
	IOCTL_USBFS_DISCONNECT	_IO('U', 22)
	IOCTL_USBFS_CONNECT	_IO('U', 23)
	IOCTL_USBFS_CLAIM_PORT	= _IOR('U', 24, uint)
	IOCTL_USBFS_RELEASE_PORT	= _IOR('U', 25, uint)
	IOCTL_USBFS_GET_CAPABILITIES	= _IOR('U', 26, __u32)
	IOCTL_USBFS_DISCONNECT_CLAIM	= _IOR('U', 27, struct usbfs_disconnect_claim)
	IOCTL_USBFS_ALLOC_STREAMS	= _IOR('U', 28, struct usbfs_streams)
	IOCTL_USBFS_FREE_STREAMS	= _IOR('U', 29, struct usbfs_streams)
)
   
type usbfs_ctrltransfer  struct{
	/* keep in sync with usbdevice_fs.h:usbdevfs_ctrltransfer */
	bmRequestType uint8
	bRequest uint8
	wValue uint16
	wIndex uint16
	wLength uint16

	timeout uint32 /* in milliseconds */

	/* pointer to data */
	data interface{}
}

type usbfs_bulktransfer  struct{
	/* keep in sync with usbdevice_fs.h:usbdevfs_bulktransfer */
	ep uint
	len uint
	timeout uint /* in milliseconds */

	/* pointer to data */
	data interface{}
}

type usbfs_setinterface  struct{
	/* keep in sync with usbdevice_fs.h:usbdevfs_setinterface */
	interface uint
	altsetting uint
}
type usbfs_getdriver  struct{
	interface uint
	driver [USBFS_MAXDRIVERNAME + 1]rune
}

type usbfs_iso_packet_desc  struct{
	length uint
	actual_length uint
	status uint
}

type usbfs_urb  struct{
	type uint8
	endpoint uint8
	status int
	flags uint
	*buffer void
	buffer_length int
	actual_length int
	start_frame int
	packets_or_stream_id int // ???
	error_count int
	signr uint
	*usercontext void
	iso_frame_desc [0]usbfs_iso_packet_desc // ???? 
}

type usbfs_connectinfo  struct{
	devnum uint
	slow uint8
}

type usbfs_ioctl  struct{
	ifno int	/* interface 0..N  negative numbers reserved */
	ioctl_code int	 /* MUST encode size + direction of data so the
	*  macros in <asm/ioctl.h> give correct values */
	data interface{}	 /* param buffer (in, or out) */
}

type usbfs_hub_portinfo  struct{
	numports uint8
	port [127]uint8	 /* port to device num mapping */
}

type usbfs_disconnect_claim  struct{
	interface uint
	flags uint
	driver [USBFS_MAXDRIVERNAME + 1]rune
}

type usbfs_streams  struct{
	num_streams uint /* Not used by USBDEVFS_FREE_STREAMS */
	num_eps uint
	eps [0]uint8 // ????????
}
