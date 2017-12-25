/*
 * usbfs header structures
 * Copyright © 2007 Daniel Drake <dsd@gentoo.org>
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

#define SYSFS_DEVICE_PATH "/sys/bus/usb/devices"

struct usbfs_ctrltransfer {
	/* keep in sync with usbdevice_fs.h:usbdevfs_ctrltransfer */
	uint8  bmRequestType;
	uint8  bRequest;
	uint16_t wValue;
	uint16_t wIndex;
	uint16_t wLength;

	uint32 timeout;	/* in milliseconds */

	/* pointer to data */
	void *data;
};

struct usbfs_bulktransfer {
	/* keep in sync with usbdevice_fs.h:usbdevfs_bulktransfer */
	unsigned int ep;
	unsigned int len;
	unsigned int timeout;	/* in milliseconds */

	/* pointer to data */
	void *data;
};

struct usbfs_setinterface {
	/* keep in sync with usbdevice_fs.h:usbdevfs_setinterface */
	unsigned int interface;
	unsigned int altsetting;
};

#define USBFS_MAXDRIVERNAME 255

struct usbfs_getdriver {
	unsigned int interface;
	char driver[USBFS_MAXDRIVERNAME + 1];
};

#define USBFS_URB_SHORT_NOT_OK		0x01
#define USBFS_URB_ISO_ASAP			0x02
#define USBFS_URB_BULK_CONTINUATION	0x04
#define USBFS_URB_QUEUE_BULK		0x10
#define USBFS_URB_ZERO_PACKET		0x40

struct usbfs_iso_packet_desc {
	unsigned int length;
	unsigned int actual_length;
	unsigned int status;
};

#define MAX_ISO_BUFFER_LENGTH		49152 * 128
#define MAX_BULK_BUFFER_LENGTH		16384
#define MAX_CTRL_BUFFER_LENGTH		4096

struct usbfs_urb {
	uint8 type;
	uint8 endpoint;
	int status;
	unsigned int flags;
	void *buffer;
	int buffer_length;
	int actual_length;
	int start_frame;
	union {
		int number_of_packets;	/* Only used for isoc urbs */
		unsigned int stream_id;	/* Only used with bulk streams */
	};
	int error_count;
	unsigned int signr;
	void *usercontext;
	struct usbfs_iso_packet_desc iso_frame_desc[0];
};

struct usbfs_connectinfo {
	unsigned int devnum;
	uint8 slow;
};

struct usbfs_ioctl {
	int ifno;	/* interface 0..N ; negative numbers reserved */
	int ioctl_code;	/* MUST encode size + direction of data so the
			 * macros in <asm/ioctl.h> give correct values */
	void *data;	/* param buffer (in, or out) */
};

struct usbfs_hub_portinfo {
	uint8 numports;
	uint8 port[127];	/* port to device num mapping */
};

#define USBFS_CAP_ZERO_PACKET		0x01
#define USBFS_CAP_BULK_CONTINUATION	0x02
#define USBFS_CAP_NO_PACKET_SIZE_LIM	0x04
#define USBFS_CAP_BULK_SCATTER_GATHER	0x08
#define USBFS_CAP_REAP_AFTER_DISCONNECT	0x10

#define USBFS_DISCONNECT_CLAIM_IF_DRIVER	0x01
#define USBFS_DISCONNECT_CLAIM_EXCEPT_DRIVER	0x02

struct usbfs_disconnect_claim {
	unsigned int interface;
	unsigned int flags;
	char driver[USBFS_MAXDRIVERNAME + 1];
};

struct usbfs_streams {
	unsigned int num_streams; /* Not used by USBDEVFS_FREE_STREAMS */
	unsigned int num_eps;
	uint8 eps[0];
};

#define IOCTL_USBFS_CONTROL	_IOWR('U', 0, struct usbfs_ctrltransfer)
#define IOCTL_USBFS_BULK		_IOWR('U', 2, struct usbfs_bulktransfer)
#define IOCTL_USBFS_RESETEP	_IOR('U', 3, unsigned int)
#define IOCTL_USBFS_SETINTF	_IOR('U', 4, struct usbfs_setinterface)
#define IOCTL_USBFS_SETCONFIG	_IOR('U', 5, unsigned int)
#define IOCTL_USBFS_GETDRIVER	_IOW('U', 8, struct usbfs_getdriver)
#define IOCTL_USBFS_SUBMITURB	_IOR('U', 10, struct usbfs_urb)
#define IOCTL_USBFS_DISCARDURB	_IO('U', 11)
#define IOCTL_USBFS_REAPURB	_IOW('U', 12, void *)
#define IOCTL_USBFS_REAPURBNDELAY	_IOW('U', 13, void *)
#define IOCTL_USBFS_CLAIMINTF	_IOR('U', 15, unsigned int)
#define IOCTL_USBFS_RELEASEINTF	_IOR('U', 16, unsigned int)
#define IOCTL_USBFS_CONNECTINFO	_IOW('U', 17, struct usbfs_connectinfo)
#define IOCTL_USBFS_IOCTL         _IOWR('U', 18, struct usbfs_ioctl)
#define IOCTL_USBFS_HUB_PORTINFO	_IOR('U', 19, struct usbfs_hub_portinfo)
#define IOCTL_USBFS_RESET		_IO('U', 20)
#define IOCTL_USBFS_CLEAR_HALT	_IOR('U', 21, unsigned int)
#define IOCTL_USBFS_DISCONNECT	_IO('U', 22)
#define IOCTL_USBFS_CONNECT	_IO('U', 23)
#define IOCTL_USBFS_CLAIM_PORT	_IOR('U', 24, unsigned int)
#define IOCTL_USBFS_RELEASE_PORT	_IOR('U', 25, unsigned int)
#define IOCTL_USBFS_GET_CAPABILITIES	_IOR('U', 26, __u32)
#define IOCTL_USBFS_DISCONNECT_CLAIM	_IOR('U', 27, struct usbfs_disconnect_claim)
#define IOCTL_USBFS_ALLOC_STREAMS	_IOR('U', 28, struct usbfs_streams)
#define IOCTL_USBFS_FREE_STREAMS	_IOR('U', 29, struct usbfs_streams)
