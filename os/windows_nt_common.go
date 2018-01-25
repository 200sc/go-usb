package os

/*
 * Windows backend common header for libusb 1.0
 *
 * This file brings together header code common between
 * the desktop Windows backends.
 * Copyright © 2012-2013 RealVNC Ltd.
 * Copyright © 2009-2012 Pete Batard <pete@akeo.ie>
 * With contributions from Michael Plante, Orin Eman et al.
 * Parts of this code adapted from libusb-win32-v1 by Stephan Meyer
 * Major code testing contribution by Xiaofan Chen
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

// Missing from MinGW

type USB_CONFIGURATION_DESCRIPTIOR struct {
	bLength             uint8
	bDescriptorType     uint8
	wTotalLength        uint16
	bNumInterfaces      uint8
	bConfigurationValue uint8
	iConfiguration      uint8
	bmAttributes        uint8
	MaxPower            uint8
}

// Global variables
const (
 TIMER_REQUEST_RETRY_MS	= 100
 WM_TIMER_REQUEST = (WM_USER + 1)
 WM_TIMER_EXIT = (WM_USER + 2)
)

// Timer thread
var (
	timer_thread HANDLE
	timer_thread_id uint16
)

// func windows_init_dlls(void) libusb_error {
	
	// Use w32 instead 
	// DLL_LOAD_FUNC_PREFIXED(User32, p, GetMessageA, TRUE);
	// DLL_LOAD_FUNC_PREFIXED(User32, p, PeekMessageA, TRUE);
	// DLL_LOAD_FUNC_PREFIXED(User32, p, PostThreadMessageA, TRUE);

// 	return LIBUSB_SUCCESS
// }

int windows_common_init(ctx *libusb_context) {
	if !windows_init_clock(ctx) {
		windows_common_exit()
		return LIBUSB_ERROR_NO_MEM
	}

	if !htab_create(ctx, HTAB_SIZE) {
		windows_common_exit()
		return LIBUSB_ERROR_NO_MEM
	}

	return LIBUSB_SUCCESS
}

func windows_common_exit() {
	htab_destroy()
	windows_destroy_clock()
}

func windows_handle_callback(itransfer *usbi_transfer, io_result uint32, io_size uint32) {
	transfer := itransfer.libusbTransfer

	switch (transfer.type) {
	case LIBUSB_TRANSFER_TYPE_CONTROL:
		fallthrough
	case LIBUSB_TRANSFER_TYPE_BULK:
		fallthrough
	case LIBUSB_TRANSFER_TYPE_INTERRUPT:
		fallthrough
	case LIBUSB_TRANSFER_TYPE_ISOCHRONOUS:
		windows_transfer_callback(itransfer, io_result, io_size)
	case LIBUSB_TRANSFER_TYPE_BULK_STREAM:
		// usbi_warn(ITRANSFER_CTX(itransfer), "bulk stream transfers are not yet supported on this platform")
	default:
		// usbi_err(ITRANSFER_CTX(itransfer), "unknown endpoint type %d", transfer.type)
	}
}

func windows_transfer_callback(itransfer *usbi_transfer, io_result, io_size uint32) {
	var status int

	// usbi_dbg("handling I/O completion with errcode %u, size %u", io_result, io_size)

	switch io_result {
	case NO_ERROR:
		status = windows_copy_transfer_data(itransfer, io_size)
	case ERROR_GEN_FAILURE:
		// usbi_dbg("detected endpoint stall")
		status = LIBUSB_TRANSFER_STALL
	case ERROR_SEM_TIMEOUT:
		// usbi_dbg("detected semaphore timeout")
		status = LIBUSB_TRANSFER_TIMED_OUT
	case ERROR_OPERATION_ABORTED:
		windows_copy_transfer_data(itransfer, io_size)
		// if (istatus != LIBUSB_TRANSFER_COMPLETED)
			// usbi_dbg("Failed to copy partial data in aborted operation: %d", istatus)

		// usbi_dbg("detected operation aborted")
		status = LIBUSB_TRANSFER_CANCELLED
	default:
		// usbi_err(ITRANSFER_CTX(itransfer), "detected I/O error %u: %s", io_result, windows_error_str(io_result))
		status = LIBUSB_TRANSFER_ERROR
	}
	windows_clear_transfer_priv(itransfer)	// Cancel polling
	if status == LIBUSB_TRANSFER_CANCELLED {
		usbi_handle_transfer_cancellation(itransfer)
	} else {
		usbi_handle_transfer_completion(itransfer, (libusb_transfer_status)status)
	}
}

func windows_handle_events(ctx *libusb_context, fds []pollfd, nfds POLL_NFDS_TYPE, num_ready int) int {
	var pollable_fd *winfd
	var io_size, io_result uint16

	r := LIBUSB_SUCCESS
	found := false
	
	ctx.open_devs_lock.Lock()
	for i := POLL_NFDS_TYPE(0); i < nfds && num_ready > 0; i++ {

		// usbi_dbg("checking fd %d with revents = %04x", fds[i].fd, fds[i].revents)

		if (!fds[i].revents) {
			continue
		}

		num_ready--

		// Because a Windows OVERLAPPED is used for poll emulation,
		// a pollable fd is created and stored with each transfer
		ctx.flying_transfers_lock.Lock()
		found = false	
		for transfer := list_entry((&ctx.flying_transfers).next, usbi_transfer, list);
			&transfer.list != (&ctx.flying_transfers);
	  		transfer = list_entry(transfer.list.next, usbi_transfer, list) {
			pollable_fd = windows_get_fd(transfer)
			if (pollable_fd.fd == fds[i].fd) {
				found = true
				break
			}
		}
		ctx.flying_transfers_lock.Unlock()

		if found {
			windows_get_overlapped_result(transfer, pollable_fd, &io_result, &io_size)

			usbi_remove_pollfd(ctx, pollable_fd.fd)
			// let handle_callback free the event using the transfer wfd
			// If you don't use the transfer wfd, you run a risk of trying to free a
			// newly allocated wfd that took the place of the one from the transfer.
			windows_handle_callback(transfer, io_result, io_size)
		} else {
			// usbi_err(ctx, "could not find a matching transfer for fd %d", fds[i])
			r = LIBUSB_ERROR_NOT_FOUND
			break
		}
	}
	ctx.open_devs_lock.Unlock()

	return r
}

func windows_destroy_clock() {
	if timer_thread != nil {
		// actually the signal to quit the thread.
		if !pPostThreadMessageA(timer_thread_id, WM_TIMER_EXIT, 0, 0) || (WaitForSingleObject(timer_thread, INFINITE) != WAIT_OBJECT_0) {
			// usbi_dbg("could not wait for timer thread to quit")
			TerminateThread(timer_thread, 1)
			// shouldn't happen, but we're destroying
			// all objects it might have held anyway.
		}
		CloseHandle(timer_thread)
		timer_thread = nil
		timer_thread_id = 0
	}
}

func windows_init_clock(ctx *libusb_context) bool {
	var affinity, dummy *uint16
	var event HANDLE
	var li_frequency int64
	var i int

	if (QueryPerformanceFrequency(&li_frequency)) {
		// The hires frequency can go as high as 4 GHz, so we'll use a conversion
		// to picoseconds to compute the tv_nsecs part in clock_gettime
		hires_frequency := li_frequency.QuadPart
		hires_ticks_to_ps := 1000000000000 / hires_frequency
		// usbi_dbg("hires timer available (Frequency: %"PRIu64" Hz)", hires_frequency)

		// Because QueryPerformanceCounter might report different values when
		// running on different cores, we create a separate thread for the timer
		// calls, which we glue to the first available core always to prevent timing discrepancies.
		if !GetProcessAffinityMask(GetCurrentProcess(), &affinity, &dummy) || (affinity == 0) {
			// usbi_err(ctx, "could not get process affinity: %s", windows_error_str(0))
			return false
		}

		// The process affinity mask is a bitmask where each set bit represents a core on
		// which this process is allowed to run, so we find the first set bit
		// Go todo: I know what this is doing and I know how to replicate it (unsafe.Pointer) but is there another approach?
		for (i = 0; !(affinity & (DWORD_PTR)(1 << i)); i++); {
			affinity = (DWORD_PTR)(1 << i)
		}

		// usbi_dbg("timer thread will run on core #%d", i)

		event = CreateEvent(nil, FALSE, FALSE, nil)
		if event == nil {
			// usbi_err(ctx, "could not create event: %s", windows_error_str(0))
			return false
		}

		timer_thread = HANDLE(_beginthreadex(nil, 0, windows_clock_gettime_threaded, event, 0, &timer_thread_id))
		if timer_thread == nil {
			// usbi_err(ctx, "unable to create timer thread - aborting")
			CloseHandle(event)
			return false
		}

		!SetThreadAffinityMask(timer_thread, affinity)
		// if <above line> 
			// usbi_warn(ctx, "unable to set timer thread affinity, timer discrepancies may arise")

		// Wait for timer thread to init before continuing.
		if WaitForSingleObject(event, INFINITE) != WAIT_OBJECT_0 {
			// usbi_err(ctx, "failed to wait for timer thread to become ready - aborting")
			CloseHandle(event)
			return false
		}

		CloseHandle(event)
	} else {
		// usbi_dbg("no hires timer available on this platform")
		hires_frequency = 0
		hires_ticks_to_ps = UINT64_C(0)
	}

	return true
}