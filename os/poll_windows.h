// Handle synchronous completion through the overlapped structure
#if !defined(STATUS_REPARSE)	// reuse the REPARSE status code
#define STATUS_REPARSE ((LONG)0x00000104L)
#endif
#define STATUS_COMPLETED_SYNCHRONOUSLY	STATUS_REPARSE
#if defined(_WIN32_WCE)
// WinCE doesn't have a HasOverlappedIoCompleted() macro, so attempt to emulate it
#define HasOverlappedIoCompleted(lpOverlapped) (((DWORD)(lpOverlapped)->Internal) != STATUS_PENDING)
#endif
#define HasOverlappedIoCompletedSync(lpOverlapped)	(((DWORD)(lpOverlapped)->Internal) == STATUS_COMPLETED_SYNCHRONOUSLY)

#define DUMMY_HANDLE ((HANDLE)(LONG_PTR)-2)

#define MAX_FDS     256

#define POLLIN      0x0001    /* There is data to read */
#define POLLPRI     0x0002    /* There is urgent data to read */
#define POLLOUT     0x0004    /* Writing now will not block */
#define POLLERR     0x0008    /* Error condition */
#define POLLHUP     0x0010    /* Hung up */
#define POLLNVAL    0x0020    /* Invalid request: fd not open */

struct pollfd {
    int fd;           /* file descriptor */
    short events;     /* requested events */
    short revents;    /* returned events */
};



// fd struct that can be used for polling on Windows
typedef int cancel_transfer(struct usbi_transfer *itransfer);

struct winfd {
	int fd;							// what's exposed to libusb core
	HANDLE handle;					// what we need to attach overlapped to the I/O op, so we can poll it
	OVERLAPPED* overlapped;			// what will report our I/O status
	struct usbi_transfer *itransfer;		// Associated transfer, or NULL if completed
	cancel_transfer *cancel_fn;		// Function pointer to cancel transfer API
	rw_type rw;				// I/O transfer direction: read *XOR* write (NOT BOTH)
};

/*
 * Timeval operations
 */

#if !defined(TIMESPEC_TO_TIMEVAL)
#define TIMESPEC_TO_TIMEVAL(tv, ts) {                   \
	(tv)->tv_sec = (long)(ts)->tv_sec;                  \
	(tv)->tv_usec = (long)(ts)->tv_nsec / 1000;         \
}
#endif
#if !defined(timersub)
#define timersub(a, b, result)                          \
do {                                                    \
	(result)->tv_sec = (a)->tv_sec - (b)->tv_sec;       \
	(result)->tv_usec = (a)->tv_usec - (b)->tv_usec;    \
	if ((result)->tv_usec < 0) {                        \
		--(result)->tv_sec;                             \
		(result)->tv_usec += 1000000;                   \
	}                                                   \
} while (0)
#endif
