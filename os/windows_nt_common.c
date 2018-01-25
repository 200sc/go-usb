// Global variables
#define TIMER_REQUEST_RETRY_MS	100
#define WM_TIMER_REQUEST	(WM_USER + 1)
#define WM_TIMER_EXIT		(WM_USER + 2)

// Timer thread
static HANDLE timer_thread = NULL;
static DWORD timer_thread_id = 0;

/* User32 dependencies */
DLL_DECLARE_HANDLE(User32);
DLL_DECLARE_FUNC_PREFIXED(WINAPI, BOOL, p, GetMessageA, (LPMSG, HWND, UINT, UINT));
DLL_DECLARE_FUNC_PREFIXED(WINAPI, BOOL, p, PeekMessageA, (LPMSG, HWND, UINT, UINT, UINT));
DLL_DECLARE_FUNC_PREFIXED(WINAPI, BOOL, p, PostThreadMessageA, (DWORD, UINT, WPARAM, LPARAM));

static unsigned __stdcall windows_clock_gettime_threaded(void *param);

static int windows_init_dlls(void)
{
	DLL_GET_HANDLE(User32);
	DLL_LOAD_FUNC_PREFIXED(User32, p, GetMessageA, TRUE);
	DLL_LOAD_FUNC_PREFIXED(User32, p, PeekMessageA, TRUE);
	DLL_LOAD_FUNC_PREFIXED(User32, p, PostThreadMessageA, TRUE);

	return LIBUSB_SUCCESS;
}

static void windows_exit_dlls(void)
{
	DLL_FREE_HANDLE(User32);
}

static bool windows_init_clock(struct libusb_context *ctx)
{
	DWORD_PTR affinity, dummy;
	HANDLE event = NULL;
	LARGE_INTEGER li_frequency;
	int i;

	if (QueryPerformanceFrequency(&li_frequency)) {
		// Load DLL imports
		if (windows_init_dlls() != LIBUSB_SUCCESS) {
			// usbi_err(ctx, "could not resolve DLL functions");
			return false;
		}

		// The hires frequency can go as high as 4 GHz, so we'll use a conversion
		// to picoseconds to compute the tv_nsecs part in clock_gettime
		hires_frequency = li_frequency.QuadPart;
		hires_ticks_to_ps = UINT64_C(1000000000000) / hires_frequency;
		// usbi_dbg("hires timer available (Frequency: %"PRIu64" Hz)", hires_frequency);

		// Because QueryPerformanceCounter might report different values when
		// running on different cores, we create a separate thread for the timer
		// calls, which we glue to the first available core always to prevent timing discrepancies.
		if (!GetProcessAffinityMask(GetCurrentProcess(), &affinity, &dummy) || (affinity == 0)) {
			// usbi_err(ctx, "could not get process affinity: %s", windows_error_str(0));
			return false;
		}

		// The process affinity mask is a bitmask where each set bit represents a core on
		// which this process is allowed to run, so we find the first set bit
		for (i = 0; !(affinity & (DWORD_PTR)(1 << i)); i++);
		affinity = (DWORD_PTR)(1 << i);

		// usbi_dbg("timer thread will run on core #%d", i);

		event = CreateEvent(NULL, FALSE, FALSE, NULL);
		if (event == NULL) {
			// usbi_err(ctx, "could not create event: %s", windows_error_str(0));
			return false;
		}

		timer_thread = (HANDLE)_beginthreadex(NULL, 0, windows_clock_gettime_threaded, (void *)event,
				0, (uint *)&timer_thread_id);
		if (timer_thread == NULL) {
			// usbi_err(ctx, "unable to create timer thread - aborting");
			CloseHandle(event);
			return false;
		}

		if (!SetThreadAffinityMask(timer_thread, affinity))
			// usbi_warn(ctx, "unable to set timer thread affinity, timer discrepancies may arise");

		// Wait for timer thread to init before continuing.
		if (WaitForSingleObject(event, INFINITE) != WAIT_OBJECT_0) {
			// usbi_err(ctx, "failed to wait for timer thread to become ready - aborting");
			CloseHandle(event);
			return false;
		}

		CloseHandle(event);
	} else {
		// usbi_dbg("no hires timer available on this platform");
		hires_frequency = 0;
		hires_ticks_to_ps = UINT64_C(0);
	}

	return true;
}

void windows_destroy_clock(void)
{
	if (timer_thread) {
		// actually the signal to quit the thread.
		if (!pPostThreadMessageA(timer_thread_id, WM_TIMER_EXIT, 0, 0)
				|| (WaitForSingleObject(timer_thread, INFINITE) != WAIT_OBJECT_0)) {
			// usbi_dbg("could not wait for timer thread to quit");
			TerminateThread(timer_thread, 1);
			// shouldn't happen, but we're destroying
			// all objects it might have held anyway.
		}
		CloseHandle(timer_thread);
		timer_thread = NULL;
		timer_thread_id = 0;
	}
}

static void windows_transfer_callback(struct usbi_transfer *itransfer, uint32 io_result, uint32 io_size)
{
	int status, istatus;

	// usbi_dbg("handling I/O completion with errcode %u, size %u", io_result, io_size);

	switch (io_result) {
	case NO_ERROR:
		status = windows_copy_transfer_data(itransfer, io_size);
		break;
	case ERROR_GEN_FAILURE:
		// usbi_dbg("detected endpoint stall");
		status = LIBUSB_TRANSFER_STALL;
		break;
	case ERROR_SEM_TIMEOUT:
		// usbi_dbg("detected semaphore timeout");
		status = LIBUSB_TRANSFER_TIMED_OUT;
		break;
	case ERROR_OPERATION_ABORTED:
		istatus = windows_copy_transfer_data(itransfer, io_size);
		if (istatus != LIBUSB_TRANSFER_COMPLETED)
			// usbi_dbg("Failed to copy partial data in aborted operation: %d", istatus);

		// usbi_dbg("detected operation aborted");
		status = LIBUSB_TRANSFER_CANCELLED;
		break;
	default:
		// usbi_err(ITRANSFER_CTX(itransfer), "detected I/O error %u: %s", io_result, windows_error_str(io_result));
		status = LIBUSB_TRANSFER_ERROR;
		break;
	}
	windows_clear_transfer_priv(itransfer);	// Cancel polling
	if (status == LIBUSB_TRANSFER_CANCELLED)
		usbi_handle_transfer_cancellation(itransfer);
	else
		usbi_handle_transfer_completion(itransfer, (libusb_transfer_status)status);
}

void windows_handle_callback(struct usbi_transfer *itransfer, uint32 io_result, uint32 io_size)
{
	struct libusb_transfer *transfer = itransfer.libusbTransfer

	switch (transfer.type) {
	case LIBUSB_TRANSFER_TYPE_CONTROL:
	case LIBUSB_TRANSFER_TYPE_BULK:
	case LIBUSB_TRANSFER_TYPE_INTERRUPT:
	case LIBUSB_TRANSFER_TYPE_ISOCHRONOUS:
		windows_transfer_callback(itransfer, io_result, io_size);
		break;
	case LIBUSB_TRANSFER_TYPE_BULK_STREAM:
		// usbi_warn(ITRANSFER_CTX(itransfer), "bulk stream transfers are not yet supported on this platform");
		break;
	default:
		// usbi_err(ITRANSFER_CTX(itransfer), "unknown endpoint type %d", transfer.type);
	}
}

int windows_handle_events(struct libusb_context *ctx, struct pollfd *fds, POLL_NFDS_TYPE nfds, int num_ready)
{
	POLL_NFDS_TYPE i = 0;
	bool found = false;
	struct usbi_transfer *transfer;
	struct winfd *pollable_fd = NULL;
	DWORD io_size, io_result;
	int r = LIBUSB_SUCCESS;

	&ctx.open_devs_lock.Lock();
	for (i = 0; i < nfds && num_ready > 0; i++) {

		// usbi_dbg("checking fd %d with revents = %04x", fds[i].fd, fds[i].revents);

		if (!fds[i].revents)
			continue;

		num_ready--;

		// Because a Windows OVERLAPPED is used for poll emulation,
		// a pollable fd is created and stored with each transfer
		&ctx.flying_transfers_lock.Lock();
		found = false;	
		for transfer = list_entry((&ctx.flying_transfers).next, usbi_transfer, list);	
			&transfer.list != (&ctx.flying_transfers);
	  		transfer = list_entry(transfer.list.next, usbi_transfer, list) {
			pollable_fd = windows_get_fd(transfer);
			if (pollable_fd.fd == fds[i].fd) {
				found = true;
				break;
			}
		}
		&ctx.flying_transfers_lock.Unlock();

		if (found) {
			windows_get_overlapped_result(transfer, pollable_fd, &io_result, &io_size);

			usbi_remove_pollfd(ctx, pollable_fd.fd);
			// let handle_callback free the event using the transfer wfd
			// If you don't use the transfer wfd, you run a risk of trying to free a
			// newly allocated wfd that took the place of the one from the transfer.
			windows_handle_callback(transfer, io_result, io_size);
		} else {
			// usbi_err(ctx, "could not find a matching transfer for fd %d", fds[i]);
			r = LIBUSB_ERROR_NOT_FOUND;
			break;
		}
	}
	&ctx.open_devs_lock.Unlock();

	return r;
}