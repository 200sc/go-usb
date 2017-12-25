package usb

type usbi_event_flags uint8
const(
	/* The list of pollfds has been modified */
	USBI_EVENT_POLLFDS_MODIFIED usbi_event_flags = 1 << 0,

	/* The user has interrupted the event handler */
	USBI_EVENT_USER_INTERRUPT usbi_event_flags = 1 << 1,
)

type usbi_clock uint8
const(
	USBI_CLOCK_MONOTONIC usbi_clock = iota
	USBI_CLOCK_REALTIME usbi_clock
)

type usbi_transfer_state_flags uint8
const(
	/* Transfer successfully submitted by backend */
	USBI_TRANSFER_IN_FLIGHT usbi_transfer_state_flags = 1 << 0,

	/* Cancellation was requested via libusb_cancel_transfer() */
	USBI_TRANSFER_CANCELLING usbi_transfer_state_flags = 1 << 1,

	/* Operation on the transfer failed because the device disappeared */
	USBI_TRANSFER_DEVICE_DISAPPEARED usbi_transfer_state_flags = 1 << 2,
)

type usbi_transfer_timeout_flags uint8
const(
	/* Set by backend submit_transfer() if the OS handles timeout */
	USBI_TRANSFER_OS_HANDLES_TIMEOUT usbi_transfer_timeout_flags = 1 << 0,

	/* The transfer timeout has been handled */
	USBI_TRANSFER_TIMEOUT_HANDLED usbi_transfer_timeout_flags = 1 << 1,

	/* The transfer timeout was successfully processed */
	USBI_TRANSFER_TIMED_OUT usbi_transfer_timeout_flags = 1 << 2,
)

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

	struct list_head list;
};

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
};

struct libusb_device_handle {
	/* lock protects claimed_interfaces */
	usbi_mutex_t lock;
	unsigned long claimed_interfaces;

	struct list_head list;
	struct libusb_device *dev;
	int auto_detach_kernel_driver;
	unsigned char os_priv
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

/* All standard descriptors have these 2 fields in common */
struct usb_descriptor_header {
	uint8_t bLength;
	uint8_t bDescriptorType;
};

struct usbi_pollfd {
	/* must come first */
	struct libusb_pollfd pollfd;

	struct list_head list;
};

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
};