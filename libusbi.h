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

/* The following is used to silence warnings for unused variables */
#define UNUSED(var)		do { (void)(var); } while(0)

#if !defined(ARRAYSIZE)
#define ARRAYSIZE(array) (sizeof(array) / sizeof(array[0]))
#endif

struct list_head {
	struct list_head *prev, *next;
};

/* Get an entry from the list
 *  ptr - the address of this list_head element in "type"
 *  type - the data type that contains "member"
 *  member - the list_head element in "type"
 */
#define list_entry(ptr, type, member) \
	((type *)((uintptr_t)(ptr) - (uintptr_t)offsetof(type, member)))

#define list_first_entry(ptr, type, member) \
	list_entry((ptr)->next, type, member)

/* Get each entry from a list
 *  pos - A structure pointer has a "member" element
 *  head - list head
 *  member - the list_head element in "pos"
 *  type - the type of the first parameter
 */
#define list_for_each_entry(pos, head, member, type)			\
	for (pos = list_entry((head)->next, type, member);		\
		 &pos->member != (head);				\
		 pos = list_entry(pos->member.next, type, member))

#define list_for_each_entry_safe(pos, n, head, member, type)		\
	for (pos = list_entry((head)->next, type, member),		\
		 n = list_entry(pos->member.next, type, member);	\
		 &pos->member != (head);				\
		 pos = n, n = list_entry(n->member.next, type, member))

#define list_empty(entry) ((entry)->next == (entry))

static  void list_init(struct list_head *entry)
{
	entry->prev = entry->next = entry;
}

static  void list_add(struct list_head *entry, struct list_head *head)
{
	entry->next = head->next;
	entry->prev = head;

	head->next->prev = entry;
	head->next = entry;
}

static  void list_add_tail(struct list_head *entry,
	struct list_head *head)
{
	entry->next = head;
	entry->prev = head->prev;

	head->prev->next = entry;
	head->prev = entry;
}

static  void list_del(struct list_head *entry)
{
	entry->next->prev = entry->prev;
	entry->prev->next = entry->next;
	entry->next = entry->prev = NULL;
}

static  void *usbi_reallocf(void *ptr, size_t size)
{
	void *ret = realloc(ptr, size);
	return ret;
}

#define container_of(ptr, type, member) ({			\
	const typeof( ((type *)0)->member ) *mptr = (ptr);	\
	(type *)( (char *)mptr - offsetof(type,member) );})

#ifndef MIN
#define MIN(a, b)	((a) < (b) ? (a) : (b))
#endif
#ifndef MAX
#define MAX(a, b)	((a) > (b) ? (a) : (b))
#endif

#define TIMESPEC_IS_SET(ts) ((ts)->tv_sec != 0 || (ts)->tv_nsec != 0)

#if defined(_WIN32) || defined(__CYGWIN__) || defined(_WIN32_WCE)
#define TIMEVAL_TV_SEC_TYPE	long
#else
#define TIMEVAL_TV_SEC_TYPE	time_t
#endif

/* Some platforms don't have this define */
#ifndef TIMESPEC_TO_TIMEVAL
#define TIMESPEC_TO_TIMEVAL(tv, ts)					\
	do {								\
		(tv)->tv_sec = (TIMEVAL_TV_SEC_TYPE) (ts)->tv_sec;	\
		(tv)->tv_usec = (ts)->tv_nsec / 1000;			\
	} while (0)
#endif

void usbi_log(struct libusb_context *ctx, libusb_log_level level,
	const char *function, const char *format, ...);

void usbi_log_v(struct libusb_context *ctx, libusb_log_level level,
	const char *function, const char *format, va_list args);

#if !defined(_MSC_VER) || _MSC_VER >= 1400

#define _usbi_log(ctx, level, ...) do { (void)(ctx); } while(0)
#define usbi_dbg(...) do {} while(0)

#define usbi_info(ctx, ...) _usbi_log(ctx, LIBUSB_LOG_LEVEL_INFO, __VA_ARGS__)
#define usbi_warn(ctx, ...) _usbi_log(ctx, LIBUSB_LOG_LEVEL_WARNING, __VA_ARGS__)
#define usbi_err(ctx, ...) _usbi_log(ctx, LIBUSB_LOG_LEVEL_ERROR, __VA_ARGS__)

#else /* !defined(_MSC_VER) || _MSC_VER >= 1400 */


#define LOG_BODY(ctxt, level)				\
{							\
	(void)(ctxt);					\
}

static  void usbi_info(struct libusb_context *ctx, const char *format, ...)
	LOG_BODY(ctx, LIBUSB_LOG_LEVEL_INFO)
static  void usbi_warn(struct libusb_context *ctx, const char *format, ...)
	LOG_BODY(ctx, LIBUSB_LOG_LEVEL_WARNING)
static  void usbi_err(struct libusb_context *ctx, const char *format, ...)
	LOG_BODY(ctx, LIBUSB_LOG_LEVEL_ERROR)

static  void usbi_dbg(const char *format, ...)
	LOG_BODY(NULL, LIBUSB_LOG_LEVEL_DEBUG)

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

/* Internal abstraction for thread synchronization */
#if defined(THREADS_POSIX)
#include "os/threads_posix.h"
#elif defined(OS_WINDOWS) || defined(OS_WINCE)
#include "os/threads_windows.h"
#endif

extern struct libusb_context *usbi_default_context;

/* Forward declaration for use in context (fully defined inside poll abstraction) */
struct pollfd;

struct libusb_context {
	int debug;
	int debug_fixed;

	/* internal event pipe, used for signalling occurrence of an internal event. */
	int event_pipe[2];

	struct list_head usb_devs;
	usbi_mutex_t usb_devs_lock;

	/* A list of open handles. Backends are free to traverse this if required.
	 */
	struct list_head open_devs;
	usbi_mutex_t open_devs_lock;

	/* A list of registered hotplug callbacks */
	struct list_head hotplug_cbs;
	usbi_mutex_t hotplug_cbs_lock;

	/* this is a list of in-flight transfer handles, sorted by timeout
	 * expiration. URBs to timeout the soonest are placed at the beginning of
	 * the list, URBs that will time out later are placed after, and urbs with
	 * infinite timeout are always placed at the very end. */
	struct list_head flying_transfers;
	/* Note paths taking both this and usbi_transfer->lock must always
	 * take this lock first */
	usbi_mutex_t flying_transfers_lock;

	/* user callbacks for pollfd changes */
	libusb_pollfd_added_cb fd_added_cb;
	libusb_pollfd_removed_cb fd_removed_cb;
	void *fd_cb_user_data;

	/* ensures that only one thread is handling events at any one time */
	usbi_mutex_t events_lock;

	/* used to see if there is an active thread doing event handling */
	int event_handler_active;

	/* A thread-local storage key to track which thread is performing event
	 * handling */
	usbi_tls_key_t event_handling_key;

	/* used to wait for event completion in threads other than the one that is
	 * event handling */
	usbi_mutex_t event_waiters_lock;
	usbi_cond_t event_waiters_cond;

	/* A lock to protect internal context event data. */
	usbi_mutex_t event_data_lock;

	/* A bitmask of flags that are set to indicate specific events that need to
	 * be handled. Protected by event_data_lock. */
	unsigned int event_flags;

	/* A counter that is set when we want to interrupt and prevent event handling,
	 * in order to safely close a device. Protected by event_data_lock. */
	unsigned int device_close;

	/* list and count of poll fds and an array of poll fd structures that is
	 * (re)allocated as necessary prior to polling. Protected by event_data_lock. */
	struct list_head ipollfds;
	struct pollfd *pollfds;
	POLL_NFDS_TYPE pollfds_cnt;

	/* A list of pending hotplug messages. Protected by event_data_lock. */
	struct list_head hotplug_msgs;

	/* A list of pending completed transfers. Protected by event_data_lock. */
	struct list_head completed_transfers;

#ifdef USBI_TIMERFD_AVAILABLE
	/* used for timeout handling, if supported by OS.
	 * this timerfd is maintained to trigger on the next pending timeout */
	int timerfd;
#endif

	struct list_head list;
};

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

struct libusb_device {
	/* lock protects refcnt, everything else is finalized at initialization
	 * time */
	usbi_mutex_t lock;
	int refcnt;

	struct libusb_context *ctx;

	uint8_t bus_number;
	uint8_t port_number;
	struct libusb_device* parent_dev;
	uint8_t device_address;
	uint8_t num_configurations;
	libusb_speed speed;

	struct list_head list;
	unsigned long session_data;

	struct libusb_device_descriptor device_descriptor;
	int attached;

	unsigned char os_priv
#if defined(__STDC_VERSION__) && (__STDC_VERSION__ >= 199901L)
	[] /* valid C99 code */
#else
	[0] /* non-standard, but usually working code */
#endif
	;
};

struct libusb_device_handle {
	/* lock protects claimed_interfaces */
	usbi_mutex_t lock;
	unsigned long claimed_interfaces;

	struct list_head list;
	struct libusb_device *dev;
	int auto_detach_kernel_driver;
	unsigned char os_priv
#if defined(__STDC_VERSION__) && (__STDC_VERSION__ >= 199901L)
	[] /* valid C99 code */
#else
	[0] /* non-standard, but usually working code */
#endif
	;
};

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

struct usbi_transfer {
	int num_iso_packets;
	struct list_head list;
	struct list_head completed_list;
	struct timeval timeout;
	int transferred;
	uint32_t stream_id;
	uint8_t state_flags;   /* Protected by usbi_transfer->lock */
	uint8_t timeout_flags; /* Protected by the flying_stransfers_lock */

	/* this lock is held during libusb_submit_transfer() and
	 * libusb_cancel_transfer() (allowing the OS backend to prevent duplicate
	 * cancellation, submission-during-cancellation, etc). the OS backend
	 * should also take this lock in the handle_events path, to prevent the user
	 * cancelling the transfer from another thread while you are processing
	 * its completion (presumably there would be races within your OS backend
	 * if this were possible).
	 * Note paths taking both this and the flying_transfers_lock must
	 * always take the flying_transfers_lock first */
	usbi_mutex_t lock;
};

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

/* All standard descriptors have these 2 fields in common */
struct usb_descriptor_header {
	uint8_t bLength;
	uint8_t bDescriptorType;
};

/* shared data and functions */

/* Internal abstraction for poll (needs struct usbi_transfer on Windows) */
#if defined(OS_LINUX) || defined(OS_DARWIN)
#include <unistd.h>
#include "os/poll_posix.h"
#elif defined(OS_WINDOWS) || defined(OS_WINCE)
#include "os/poll_windows.h"
#endif

#if (defined(OS_WINDOWS) || defined(OS_WINCE)) && !defined(__GNUC__)
#define snprintf _snprintf
#define vsnprintf _vsnprintf
int usbi_gettimeofday(struct timeval *tp, void *tzp);
#define LIBUSB_GETTIMEOFDAY_WIN32
#define HAVE_USBI_GETTIMEOFDAY
#else
#endif

struct usbi_pollfd {
	/* must come first */
	struct libusb_pollfd pollfd;

	struct list_head list;
};

int usbi_add_pollfd(struct libusb_context *ctx, int fd, short events);
void usbi_remove_pollfd(struct libusb_context *ctx, int fd);

/* device discovery */

/* we traverse usbfs without knowing how many devices we are going to find.
 * so we create this discovered_devs model which is similar to a linked-list
 * which grows when required. it can be freed once discovery has completed,
 * eliminating the need for a list node in the libusb_device structure
 * itself. */
struct discovered_devs {
	size_t len;
	size_t capacity;
	struct libusb_device *devices
#if defined(__STDC_VERSION__) && (__STDC_VERSION__ >= 199901L)
	[] /* valid C99 code */
#else
	[0] /* non-standard, but usually working code */
#endif
	;
};

struct discovered_devs *discovered_devs_append(
	struct discovered_devs *discdevs, struct libusb_device *dev);