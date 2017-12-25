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

#define DEVICE_DESC_LENGTH	18

#define USB_MAXENDPOINTS	32
#define USB_MAXINTERFACES	32
#define USB_MAXCONFIG		8

/* Backend specific capabilities */
#define USBI_CAP_HAS_HID_ACCESS			0x00010000
#define USBI_CAP_SUPPORTS_DETACH_KERNEL_DRIVER	0x00020000

/* Maximum number of bytes in a log line */
#define USBI_MAX_LOG_LEN	1024
/* Terminator for log lines */
#define USBI_LOG_LINE_END	"\n"

#if !defined(ARRAYSIZE)
#define ARRAYSIZE(array) (sizeof(array) / sizeof(array[0]))
#endif

static  void *usbi_reallocf(void *ptr, size_t size)
{
	void *ret = realloc(ptr, size);
	return ret;
}

#define container_of(ptr, type, member) ({			\
	const typeof( ((type *)0)->member ) *mptr = (ptr);	\
	(type *)( (char *)mptr - offsetof(type,member) );})

#define TIMESPEC_IS_SET(ts) ((ts)->tv_sec != 0 || (ts)->tv_nsec != 0)

/* Some platforms don't have this define */
#ifndef TIMESPEC_TO_TIMEVAL
#define TIMESPEC_TO_TIMEVAL(tv, ts)					\
	do {								\
		(tv)->tv_sec = (int64) (ts)->tv_sec;	\
		(tv)->tv_usec = (ts)->tv_nsec / 1000;			\
	} while (0)
#endif

#if !defined(_MSC_VER) || _MSC_VER >= 1400

#else /* !defined(_MSC_VER) || _MSC_VER >= 1400 */
#endif /* !defined(_MSC_VER) || _MSC_VER >= 1400 */

#define USBI_GET_CONTEXT(ctx)				\
	do {						\
		if (!(ctx))				\
			(ctx) = usbi_default_context;	\
	} while(0)

#define DEVICE_CTX(dev)		((dev)->ctx)
#define HANDLE_CTX(handle)	(DEVICE_CTX((handle)->dev))
#define TRANSFER_CTX(transfer)	(HANDLE_CTX((transfer)->dev_handle))
#define ITRANSFER_CTX(transfer) \
	(TRANSFER_CTX(USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer)))

#define IS_EPIN(ep)		(0 != ((ep) & LIBUSB_ENDPOINT_IN))
#define IS_EPOUT(ep)		(!IS_EPIN(ep))
#define IS_XFERIN(xfer)		(0 != ((xfer)->endpoint & LIBUSB_ENDPOINT_IN))
#define IS_XFEROUT(xfer)	(!IS_XFERIN(xfer))

/* Forward declaration for use in context (fully defined inside poll abstraction) */
/* Macros for managing event handling state */
#define usbi_handling_events(ctx) \
	(usbi_tls_key_get((ctx)->event_handling_key) != NULL)

#define usbi_start_event_handling(ctx) \
	usbi_tls_key_set((ctx)->event_handling_key, ctx)

#define usbi_end_event_handling(ctx) \
	usbi_tls_key_set((ctx)->event_handling_key, NULL)

/* Update the following macro if new event sources are added */
#define usbi_pending_events(ctx) \
	((ctx)->event_flags || (ctx)->device_close \
	 || !list_empty(&(ctx)->hotplug_msgs) || !list_empty(&(ctx)->completed_transfers))

#ifdef USBI_TIMERFD_AVAILABLE
#define usbi_using_timerfd(ctx) ((ctx)->timerfd >= 0)
#else
#define usbi_using_timerfd(ctx) (0)
#endif

#define USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer)			\
	((struct libusb_transfer *)(((unsigned char *)(transfer))	\
		+ sizeof(struct usbi_transfer)))
#define LIBUSB_TRANSFER_TO_USBI_TRANSFER(transfer)			\
	((struct usbi_transfer *)(((unsigned char *)(transfer))		\
		- sizeof(struct usbi_transfer)))

static  void *usbi_transfer_get_os_priv(struct usbi_transfer *transfer)
{
	return ((unsigned char *)transfer) + sizeof(struct usbi_transfer)
		+ sizeof(struct libusb_transfer)
		+ (transfer->num_iso_packets
			* sizeof(struct libusb_iso_packet_descriptor));
}

/* bus structures */

/* shared data and functions */

#if (defined(OS_WINDOWS) || defined(OS_WINCE)) && !defined(__GNUC__)
#define snprintf _snprintf
#define vsnprintf _vsnprintf
#define LIBUSB_GETTIMEOFDAY_WIN32
#define HAVE_USBI_GETTIMEOFDAY
#else
#endif