

#if !defined(ARRAYSIZE)
#define ARRAYSIZE(array) (sizeof(array) / sizeof(array[0]))
#endif

static  void *usbi_reallocf(void *ptr, int size)
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

#define TRANSFER_CTX(transfer)	((transfer)->dev_handle).dev.ctx
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
	((struct libusb_transfer *)(((uint8 *)(transfer))	\
		+ sizeof(struct usbi_transfer)))
#define LIBUSB_TRANSFER_TO_USBI_TRANSFER(transfer)			\
	((struct usbi_transfer *)(((uint8 *)(transfer))		\
		- sizeof(struct usbi_transfer)))

static  void *usbi_transfer_get_os_priv(struct usbi_transfer *transfer)
{
	return ((uint8 *)transfer) + sizeof(struct usbi_transfer)
		+ sizeof(struct libusb_transfer)
		+ (transfer->num_iso_packets
			* sizeof(struct libusb_iso_packet_descriptor));
}