package usb

/*
 * I/O functions for libusb
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

 func usbi_io_init(ctx *libusb_context) int {
	 ctx.event_waiters_cond = sync.NewCond(ctx.event_waiters_lock)
	 list_init(ctx.flying_transfers)
	 list_init(ctx.ipollfds)
	 list_init(ctx.hotplug_msgs)
	 list_init(ctx.completed_transfers)
 
	 /* FIXME should use an eventfd on kernels that support it */
	 r := usbi_pipe(ctx.event_pipe)
	 if (r < 0) {
		 return LIBUSB_ERROR_OTHER
	 }
 
	 r = usbi_add_pollfd(ctx, ctx.event_pipe[0], POLLIN)
	 if (r < 0) {
		usbi_close(ctx.event_pipe[0])
		usbi_close(ctx.event_pipe[1])
		return r
	 }
 
	 ctx.timerfd = timerfd_create(usbi_backend.get_timerfd_clockid(), TFD_NONBLOCK)
	 if (ctx.timerfd >= 0) {
		 // usbi_dbg("using timerfd for timeouts")
		 r = usbi_add_pollfd(ctx, ctx.timerfd, POLLIN)
		 if (r < 0) {
			close(ctx.timerfd)
			usbi_remove_pollfd(ctx, ctx.event_pipe[0])
			usbi_close(ctx.event_pipe[0])
	 		usbi_close(ctx.event_pipe[1])
	 		return r
		 }
	 }
 
	 return 0
 }

func usbi_io_exit(ctx *libusb_context) {
	usbi_remove_pollfd(ctx, ctx.event_pipe[0])
	usbi_close(ctx.event_pipe[0])
	usbi_close(ctx.event_pipe[1])
	usbi_remove_pollfd(ctx, ctx.timerfd)
	close(ctx.timerfd)
}

func calculate_timeout(transfer *usbi_transfer) int {
	
	timeout := transfer.libusbTransfer.timeout

	if (timeout == 0) {
		return 0
	}

	current_time := time.Now()

	current_time.tv_sec += timeout / 1000
	current_time.tv_nsec += (timeout % 1000) * 1000000

	while (current_time.tv_nsec >= 1000000000) {
		current_time.tv_nsec -= 1000000000
		current_time.tv_sec++
	}

	TIMESPEC_TO_TIMEVAL(&transfer.timeout, &current_time)
	return 0
}

/** \ingroup libusb_asyncio
 * Free a transfer structure. This should be called for all transfers
 * allocated with libusb_alloc_transfer().
 *
 * If the \ref libusb_transfer_flags::LIBUSB_TRANSFER_FREE_BUFFER
 * "LIBUSB_TRANSFER_FREE_BUFFER" flag is set and the transfer buffer is
 * non-NULL, this function will also free the transfer buffer using the
 * standard system memory allocator (e.g. free()).
 *
 * It is legal to call this function with a NULL transfer. In this case,
 * the function will simply return safely.
 *
 * It is not legal to free an active transfer (one which has been submitted
 * and has not yet completed).
 *
 * \param transfer the transfer to free
 */

func disarm_timerfd(ctx *libusb_context) int {
	disarm_timer := itimerspec{{0,0},{0,0}}

	r := timerfd_settime(ctx.timerfd, 0, &disarm_timer, NULL)
	if (r < 0) {
		return LIBUSB_ERROR_OTHER
	}
	
	return 0
 }

 /* iterates through the flying transfers, and rearms the timerfd based on the
 * next upcoming timeout.
 * must be called with flying_list locked.
 * returns 0 on success or a LIBUSB_ERROR code on failure.
 */
func arm_timerfd_for_next_timeout(ctx *libusb_context) int {
	var transfer *usbi_transfer

	for transfer = list_entry(ctx.flying_transfers.next, usbi_transfer, list) 
		&transfer.list != ctx.flying_transfers 
		transfer = list_entry(transfer.list.next, usbi_transfer, list) {

		cur_tv := &transfer.timeout

		/* if we've reached transfers of infinite timeout, then we have no
		 * arming to do */
		if (!timerisset(cur_tv)) {
			return disarm_timerfd(ctx)
		}

		/* act on first transfer that has not already been handled */
		if (!(transfer.timeout_flags & (USBI_TRANSFER_TIMEOUT_HANDLED | USBI_TRANSFER_OS_HANDLES_TIMEOUT))) {

			it = itimerspec{{0, 0}, {cur_tv.tv_sec, cur_tv.tv_usec * 1000 }}

			// usbi_dbg("next timeout originally %dms", USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer).timeout)
			r := timerfd_settime(ctx.timerfd, TFD_TIMER_ABSTIME, &it, NULL)
			if (r < 0) {
				return LIBUSB_ERROR_OTHER
			}
			return 0
		}
	}
	return disarm_timerfd(ctx)
}

/* add a transfer to the (timeout-sorted) active transfers list.
 * This function will return non 0 if fails to update the timer,
 * in which case the transfer is *not* on the flying_transfers list. */
 func add_to_flying_list(transfer *usbi_transfer) int {
	var cur *usbi_transfer
	struct timeval *timeout = &transfer.timeout
	struct libusb_context *ctx = transfer.libusbTransfer.dev_handle.dev.ctx
	first := true

	r := calculate_timeout(transfer)
	if (r != 0) {
		return r
	}

	/* if we have no other flying transfers, start the list with this one */
	if (list_empty(ctx.flying_transfers)) {
		list_add(transfer.list, ctx.flying_transfers)
		goto out
	}

	/* if we have infinite timeout, append to end of list */
	if (!timerisset(timeout)) {
		list_add_tail(transfer.list, ctx.flying_transfers)
		/* first is irrelevant in this case */
		goto out
	}

	/* otherwise, find appropriate place in list */
	for cur = list_entry(ctx.flying_transfers.next, usbi_transfer, list) 
		&cur.list != ctx.flying_transfers 
		cur = list_entry(cur.list.next, usbi_transfer, list) {
		/* find first timeout that occurs after the transfer in question */
		cur_tv := cur.timeout

		if (!timerisset(cur_tv) || (cur_tv.tv_sec > timeout.tv_sec) ||
				(cur_tv.tv_sec == timeout.tv_sec &&
					cur_tv.tv_usec > timeout.tv_usec)) {
			list_add_tail(transfer.list, cur.list)
			goto out
		}
		first = false
	}
	/* first is false at this stage (list not empty) */

	/* otherwise we need to be inserted at the end */
	list_add_tail(transfer.list, ctx.flying_transfers)
out:
	if (first && timerisset(timeout)) {
		/* if this transfer has the lowest timeout of all active transfers,
		 * rearm the timerfd with this transfer's timeout */
		it := itimerspec{{0, 0}, {timeout.tv_sec, timeout.tv_usec * 1000}}
		// usbi_dbg("arm timerfd for timeout in %dms (first in line)",
			// USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer.libusbTransfer.timeout)
		r = timerfd_settime(ctx.timerfd, TFD_TIMER_ABSTIME, &it, nil)
		if (r < 0) {
			// usbi_warn(ctx, "failed to arm first timerfd (errno %d)", errno)
			r = LIBUSB_ERROR_OTHER
		}
	}

	if (r != 0) {
		list_del(transfer.list)
	}

	return r
}

/* remove a transfer from the active transfers list.
 * This function will *always* remove the transfer from the
 * flying_transfers list. It will return a LIBUSB_ERROR code
 * if it fails to update the timer for the next timeout. */
func remove_from_flying_list(transfer *usbi_transfer) int {
	
	ctx := transfer.libusbTransfer.dev_handle.dev.ctx
	r := 0

	ctx.flying_transfers_lock.Lock()

	rearm_timerfd := (timerisset(transfer.timeout) && list_first_entry(ctx.flying_transfers, usbi_transfer, list) == transfer)

	list_del(transfer.list)

	if (rearm_timerfd != 0) {
		r = arm_timerfd_for_next_timeout(ctx)
	}
	
	ctx.flying_transfers_lock.Unlock()

	return r
}

/** \ingroup libusb_asyncio
 * Allocate a libusb transfer with a specified number of isochronous packet
 * descriptors. The returned transfer is pre-initialized for you. When the new
 * transfer is no longer needed, it should be freed with
 * libusb_free_transfer().
 *
 * Transfers intended for non-isochronous endpoints (e.g. control, bulk,
 * interrupt) should specify an iso_packets count of zero.
 *
 * For transfers intended for isochronous endpoints, specify an appropriate
 * number of packet descriptors to be allocated as part of the transfer.
 * The returned transfer is not specially initialized for isochronous I/O
 * you are still required to set the
 * \ref libusb_transfer::num_iso_packets "num_iso_packets" and
 * \ref libusb_transfer::type "type" fields accordingly.
 *
 * It is safe to allocate a transfer with some isochronous packets and then
 * use it on a non-isochronous endpoint. If you do this, ensure that at time
 * of submission, num_iso_packets is 0 and that type is set appropriately.
 *
 * \param iso_packets number of isochronous packet descriptors to allocate
 * \returns a newly allocated transfer, or NULL on error
 */

 func libusb_alloc_transfer(iso_packets int) *libusb_transfer {
	// surely this is wrong
	itransfer := &usbi_transfer{}
	itransfer.num_iso_packets = iso_packets

	return itransfer.libusbTransfer
}

 /** \ingroup libusb_poll
 * Handle any pending events by polling file descriptors, without checking if
 * any other threads are already doing so. Must be called with the event lock
 * held, see libusb_lock_events().
 *
 * This function is designed to be called under the situation where you have
 * taken the event lock and are calling poll()/select() directly on libusb's
 * file descriptors (as opposed to using libusb_handle_events() or similar).
 * You detect events on libusb's descriptors, so you then call this function
 * with a zero timeout value (while still holding the event lock).
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \param tv the maximum time to block waiting for events, or zero for
 * non-blocking mode
 * \returns 0 on success, or a LIBUSB_ERROR code on failure
 * \ref libusb_mtasync
 */
func libusb_handle_events_locked(ctx *libusb_context, tv *timeval) int {
	var poll_timeout timeval 

	ctx = USBI_GET_CONTEXT(ctx)
	r := get_next_timeout(ctx, tv, &poll_timeout)
	if (r != 0) {
		/* timeout already expired */
		return handle_timeouts(ctx)
	}

	return handle_events(ctx, &poll_timeout)
}

/** \ingroup libusb_poll
 * Handle any pending events in blocking mode.
 *
 * Like libusb_handle_events(), with the addition of a completed parameter
 * to allow for race free waiting for the completion of a specific transfer.
 *
 * See libusb_handle_events_timeout_completed() for details on the completed
 * parameter.
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \param completed pointer to completion integer to check, or NULL
 * \returns 0 on success, or a LIBUSB_ERROR code on failure
 * \ref libusb_mtasync
 */
func libusb_handle_events_completed(ctx *libusb_context, completed *int) int {
	var tv timeval
	tv.tv_sec = 60
	tv.tv_usec = 0
	return libusb_handle_events_timeout_completed(ctx, &tv, completed)
}

 /** \ingroup libusb_poll
 * Determines whether your application must apply special timing considerations
 * when monitoring libusb's file descriptors.
 *
 * This function is only useful for applications which retrieve and poll
 * libusb's file descriptors in their own main loop (\ref libusb_pollmain).
 *
 * Ordinarily, libusb's event handler needs to be called into at specific
 * moments in time (in addition to times when there is activity on the file
 * descriptor set). The usual approach is to use libusb_get_next_timeout()
 * to learn about when the next timeout occurs, and to adjust your
 * poll()/select() timeout accordingly so that you can make a call into the
 * library at that time.
 *
 * Some platforms supported by libusb do not come with this baggage - any
 * events relevant to timing will be represented by activity on the file
 * descriptor set, and libusb_get_next_timeout() will always return 0.
 * This function allows you to detect whether you are running on such a
 * platform.
 *
 * Since v1.0.5.
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \returns 0 if you must call into libusb at times determined by
 * libusb_get_next_timeout(), or 1 if all timeout events are handled internally
 * or through regular activity on the file descriptors.
 * \ref libusb_pollmain "Polling libusb file descriptors for event handling"
 */
func libusb_pollfds_handle_timeouts(ctx *libusb_context) int {
	//ctx = USBI_GET_CONTEXT(ctx)
	//return usbi_using_timerfd(ctx)
	// GO: assuming true for right now
	return true
}

/** \ingroup libusb_asyncio
 * Submit a transfer. This function will fire off the USB transfer and then
 * return immediately.
 *
 * \param transfer the transfer to submit
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns LIBUSB_ERROR_BUSY if the transfer has already been submitted.
 * \returns LIBUSB_ERROR_NOT_SUPPORTED if the transfer flags are not supported
 * by the operating system.
 * \returns LIBUSB_ERROR_INVALID_PARAM if the transfer size is larger than
 * the operating system and/or hardware can support
 * \returns another LIBUSB_ERROR code on other failure
 */
func libusb_submit_transfer(transfer *libusb_transfer) int {
	 itransfer := transfer.usbiTransfer
	 ctx := transfer.dev_handle.dev.ctx
	 var r int
 
	 // usbi_dbg("transfer %p", transfer)
 
	 /*
	  * Important note on locking, this function takes / releases locks
	  * in the following order:
	  *  take flying_transfers_lock
	  *  take itransfer->lock
	  *  clear transfer
	  *  add to flying_transfers list
	  *  release flying_transfers_lock
	  *  submit transfer
	  *  release itransfer->lock
	  *  if submit failed:
	  *   take flying_transfers_lock
	  *   remove from flying_transfers list
	  *   release flying_transfers_lock
	  *
	  * Note that it takes locks in the order a-b and then releases them
	  * in the same order a-b. This is somewhat unusual but not wrong,
	  * release order is not important as long as *all* locks are released
	  * before re-acquiring any locks.
	  *
	  * This means that the ordering of first releasing itransfer->lock
	  * and then re-acquiring the flying_transfers_list on error is
	  * important and must not be changed!
	  *
	  * This is done this way because when we take both locks we must always
	  * take flying_transfers_lock first to avoid ab-ba style deadlocks with
	  * the timeout handling and usbi_handle_disconnect paths.
	  *
	  * And we cannot release itransfer->lock before the submission is
	  * complete otherwise timeout handling for transfers with short
	  * timeouts may run before submission.
	  */
	 ctx.flying_transfers_lock.Lock()
	 itransfer.lock.Lock()
	 if (itransfer.state_flags & USBI_TRANSFER_IN_FLIGHT) {
		ctx.flying_transfers_lock.Unlock()
		itransfer.lock.Unlock()
		return LIBUSB_ERROR_BUSY
	 }
	 itransfer.transferred = 0
	 itransfer.state_flags = 0
	 itransfer.timeout_flags = 0
	 r = add_to_flying_list(itransfer) 
	 if (r) {
		ctx.flying_transfers_lock.Unlock()
		itransfer.lock.Unlock()
		return r
	 }
	 /*
	  * We must release the flying transfers lock here, because with
	  * some backends the submit_transfer method is synchroneous.
	  */
	 ctx.flying_transfers_lock.Unlock()
 
	 r = usbi_backend.submit_transfer(itransfer)
	 if (r == LIBUSB_SUCCESS) {
		 itransfer.state_flags |= USBI_TRANSFER_IN_FLIGHT
		 /* keep a reference to this device */
		 libusb_ref_device(transfer.dev_handle.dev)
	 }
	 itransfer.lock.Unlock()
 
	 if (r != LIBUSB_SUCCESS) {
		 remove_from_flying_list(itransfer)
	 }
 
	 return r
 }
 
 /** \ingroup libusb_asyncio
 * Asynchronously cancel a previously submitted transfer.
 * This function returns immediately, but this does not indicate cancellation
 * is complete. Your callback function will be invoked at some later time
 * with a transfer status of
 * \ref libusb_transfer_status::LIBUSB_TRANSFER_CANCELLED
 * "LIBUSB_TRANSFER_CANCELLED."
 *
 * \param transfer the transfer to cancel
 * \returns 0 on success
 * \returns LIBUSB_ERROR_NOT_FOUND if the transfer is not in progress,
 * already complete, or already cancelled.
 * \returns a LIBUSB_ERROR code on failure
 */
func libusb_cancel_transfer(transfer *libusb_transfer) int {
	itransfer := transfer.usbiTransfer
	var r int

	// usbi_dbg("transfer %p", transfer )
	itransfer.lock.Lock()
	defer itransfer.lock.Unlock()
	if !(itransfer.state_flags & USBI_TRANSFER_IN_FLIGHT) || (itransfer.state_flags & USBI_TRANSFER_CANCELLING) {
		return LIBUSB_ERROR_NOT_FOUND
	}
	r = usbi_backend.cancel_transfer(itransfer)
	if (r < 0) {
		// if (r != LIBUSB_ERROR_NOT_FOUND &&
		//     r != LIBUSB_ERROR_NO_DEVICE)
			// usbi_err(TRANSFER_CTX(transfer),
			//  "cancel transfer failed error %d", r)
		// else
			// usbi_dbg("cancel transfer failed error %d", r)

		if (r == LIBUSB_ERROR_NO_DEVICE)
			itransfer.state_flags |= USBI_TRANSFER_DEVICE_DISAPPEARED
	}

	itransfer.state_flags |= USBI_TRANSFER_CANCELLING

	return r
}

/** \ingroup libusb_asyncio
 * Set a transfers bulk stream id. Note users are advised to use
 * libusb_fill_bulk_stream_transfer() instead of calling this function
 * directly.
 *
 * Since version 1.0.19, \ref LIBUSB_API_VERSION >= 0x01000103
 *
 * \param transfer the transfer to set the stream id for
 * \param stream_id the stream id to set
 * \see libusb_alloc_streams()
 */
func libusb_transfer_set_stream_id(transfer* libusb_transfer, stream_id uint32) {
	transfer.usbiTransfer.stream_id = stream_id
}

/** \ingroup libusb_asyncio
 * Get a transfers bulk stream id.
 *
 * Since version 1.0.19, \ref LIBUSB_API_VERSION >= 0x01000103
 *
 * \param transfer the transfer to get the stream id for
 * \returns the stream id for the transfer
 */
func libusb_transfer_get_stream_id(transfer *libusb_transfer) uint32 {
	return transfer.usbiTransfer.stream_id
}

/* Handle completion of a transfer (completion might be an error condition).
 * This will invoke the user-supplied callback function, which may end up
 * freeing the transfer. Therefore you cannot use the transfer structure
 * after calling this function, and you should free all backend-specific
 * data before calling it.
 * Do not call this function with the usbi_transfer lock held. User-specified
 * callback functions may attempt to directly resubmit the transfer, which
 * will attempt to take the lock. */
func usbi_handle_transfer_completion(itransfer *usbi_transfer, status libusb_transfer_status) int {
	transfer := itransfer.libusbTransfer
	dev_handle := transfer.dev_handle
	var flags uint8
	var r int

	r = remove_from_flying_list(itransfer)
	//if (r < 0)
		// usbi_err(ITRANSFER_CTX(itransfer), "failed to set timer for next timeout, errno=%d", errno)

	itransfer.lock.Lock()
	itransfer.state_flags &= ~USBI_TRANSFER_IN_FLIGHT
	itransfer.lock.Unlock()

	if status == LIBUSB_TRANSFER_COMPLETED && transfer.flags & LIBUSB_TRANSFER_SHORT_NOT_OK {
		rqlen := transfer.length
		if transfer.type == LIBUSB_TRANSFER_TYPE_CONTROL {
			rqlen -= LIBUSB_CONTROL_SETUP_SIZE
		}
		if rqlen != itransfer.transferred {
			// usbi_dbg("interpreting short transfer as error")
			status = LIBUSB_TRANSFER_ERROR
		}
	}

	flags = transfer.flags
	transfer.status = status
	transfer.actual_length = itransfer.transferred
	// usbi_dbg("transfer %p has callback %p", transfer, transfer.callback)
	if transfer.callback != nil {
		transfer.callback(transfer)
	}

	libusb_unref_device(dev_handle.dev)
	return r
}

/* Similar to usbi_handle_transfer_completion() but exclusively for transfers
 * that were asynchronously cancelled. The same concerns w.r.t. freeing of
 * transfers exist here.
 * Do not call this function with the usbi_transfer lock held. User-specified
 * callback functions may attempt to directly resubmit the transfer, which
 * will attempt to take the lock. */
func usbi_handle_transfer_cancellation(transfer *usbi_transfer) int {
	 ctx := transfer.libusbTransfer.dev_handle.dev.ctx
 
	 ctx.flying_transfers_lock.Lock()
	 timed_out := transfer.timeout_flags & USBI_TRANSFER_TIMED_OUT
	 ctx.flying_transfers_lock.Unlock()
 
	 /* if the URB was cancelled due to timeout, report timeout to the user */
	 if timed_out != 0 {
		 // usbi_dbg("detected timeout cancellation")
		 return usbi_handle_transfer_completion(transfer, LIBUSB_TRANSFER_TIMED_OUT)
	 }
 
	 /* otherwise its a normal async cancel */
	 return usbi_handle_transfer_completion(transfer, LIBUSB_TRANSFER_CANCELLED)
 }
 
 /* Add a completed transfer to the completed_transfers list of the
  * context and signal the event. The backend's handle_transfer_completion()
  * function will be called the next time an event handler runs. */
func usbi_signal_transfer_completion(transfer *usbi_transfer) {
	 ctx := transfer.libusbTransfer.dev_handle.dev.ctx

	 ctx.event_data_lock.Lock()
	 pending_events := usbi_pending_events(ctx)
	 list_add_tail(&transfer.completed_list, &ctx.completed_transfers)
	 if pending_events != 0 {
		 usbi_signal_event(ctx)
	 }
	 ctx.event_data_lock.Unlock()
 }
 
 /** \ingroup libusb_poll
  * Attempt to acquire the event handling lock. This lock is used to ensure that
  * only one thread is monitoring libusb event sources at any one time.
  *
  * You only need to use this lock if you are developing an application
  * which calls poll() or select() on libusb's file descriptors directly.
  * If you stick to libusb's event handling loop functions (e.g.
  * libusb_handle_events()) then you do not need to be concerned with this
  * locking.
  *
  * While holding this lock, you are trusted to actually be handling events.
  * If you are no longer handling events, you must call libusb_unlock_events()
  * as soon as possible.
  *
  * \param ctx the context to operate on, or NULL for the default context
  * \returns 0 if the lock was obtained successfully
  * \returns 1 if the lock was not obtained (i.e. another thread holds the lock)
  * \ref libusb_mtasync
  */
func libusb_try_lock_events(ctx *libusb_context) int {

	 ctx = USBI_GET_CONTEXT(ctx)
 
	 /* is someone else waiting to close a device? if so, don't let this thread
	  * start event handling */
	 ctx.event_data_lock.Lock()
	 ru := ctx.device_close
	 ctx.event_data_lock.Unlock()
	 if ru != 0 {
		 // usbi_dbg("someone else is closing a device")
		 return 1
	 }
 
	 // GO problem: sync.Mutex doesn't have an equivalent to this
	 r := usbi_mutex_trylock(&ctx.events_lock)
	 if r != 0 {
		 return 1
	 }
 
	 ctx.event_handler_active = 1
	 return 0
 }
 
 /** \ingroup libusb_poll
  * Acquire the event handling lock, blocking until successful acquisition if
  * it is contended. This lock is used to ensure that only one thread is
  * monitoring libusb event sources at any one time.
  *
  * You only need to use this lock if you are developing an application
  * which calls poll() or select() on libusb's file descriptors directly.
  * If you stick to libusb's event handling loop functions (e.g.
  * libusb_handle_events()) then you do not need to be concerned with this
  * locking.
  *
  * While holding this lock, you are trusted to actually be handling events.
  * If you are no longer handling events, you must call libusb_unlock_events()
  * as soon as possible.
  *
  * \param ctx the context to operate on, or NULL for the default context
  * \ref libusb_mtasync
  */
func libusb_lock_events(ctx *libusb_context) {
	 ctx = USBI_GET_CONTEXT(ctx)
	 ctx.events_lock.Lock()
	 ctx.event_handler_active = 1
 }
 
 /** \ingroup libusb_poll
  * Release the lock previously acquired with libusb_try_lock_events() or
  * libusb_lock_events(). Releasing this lock will wake up any threads blocked
  * on libusb_wait_for_event().
  *
  * \param ctx the context to operate on, or NULL for the default context
  * \ref libusb_mtasync
  */
 func libusb_unlock_events(ctx *libusb_context) {
	 ctx = USBI_GET_CONTEXT(ctx)
	 ctx.event_handler_active = 0
	 ctx.events_lock.Unlock()
 
	 /* FIXME: perhaps we should be a bit more efficient by not broadcasting
	  * the availability of the events lock when we are modifying pollfds
	  * (check ctx.device_close)? */
	 ctx.event_waiters_lock.Lock()
	 ctx.event_waiters_cond.Broadcast()
	 ctx.event_waiters_lock.Unlock()
 }
 
 /** \ingroup libusb_poll
  * Determine if it is still OK for this thread to be doing event handling.
  *
  * Sometimes, libusb needs to temporarily pause all event handlers, and this
  * is the function you should use before polling file descriptors to see if
  * this is the case.
  *
  * If this function instructs your thread to give up the events lock, you
  * should just continue the usual logic that is documented in \ref libusb_mtasync.
  * On the next iteration, your thread will fail to obtain the events lock,
  * and will hence become an event waiter.
  *
  * This function should be called while the events lock is held: you don't
  * need to worry about the results of this function if your thread is not
  * the current event handler.
  *
  * \param ctx the context to operate on, or NULL for the default context
  * \returns 1 if event handling can start or continue
  * \returns 0 if this thread must give up the events lock
  * \ref fullstory "Multi-threaded I/O: the full story"
  */
func libusb_event_handling_ok(ctx *libusb_context) int {
	 ctx = USBI_GET_CONTEXT(ctx)
 
	 /* is someone else waiting to close a device? if so, don't let this thread
	  * continue event handling */
	 ctx.event_data_lock.Lock()
	 r := ctx.device_close
	 ctx.event_data_lock.Unlock()
	 if r != 0 {
		 // usbi_dbg("someone else is closing a device")
		 return 0
	 }
	 return 1
 }
 
 
 /** \ingroup libusb_poll
  * Determine if an active thread is handling events (i.e. if anyone is holding
  * the event handling lock).
  *
  * \param ctx the context to operate on, or NULL for the default context
  * \returns 1 if a thread is handling events
  * \returns 0 if there are no threads currently handling events
  * \ref libusb_mtasync
  */
 func libusb_event_handler_active(ctx *libusb_context) int {
	 ctx = USBI_GET_CONTEXT(ctx)
 
	 /* is someone else waiting to close a device? if so, don't let this thread
	  * start event handling -- indicate that event handling is happening */
	 ctx.event_data_lock.Lock()
	 r := ctx.device_close
	 ctx.event_data_lock.Unlock()
	 if r != 0 {
		 // usbi_dbg("someone else is closing a device")
		 return 1
	 }
 
	 return ctx.event_handler_active
 }
 
 /** \ingroup libusb_poll
  * Interrupt any active thread that is handling events. This is mainly useful
  * for interrupting a dedicated event handling thread when an application
  * wishes to call libusb_exit().
  *
  * Since version 1.0.21, \ref LIBUSB_API_VERSION >= 0x01000105
  *
  * \param ctx the context to operate on, or NULL for the default context
  * \ref libusb_mtasync
  */
func libusb_interrupt_event_handler(ctx *libusb_context) {
	 ctx = USBI_GET_CONTEXT(ctx)
 
	 ctx.event_data_lock.Lock()
	 if !usbi_pending_events(ctx) {
		 ctx.event_flags |= USBI_EVENT_USER_INTERRUPT
		 usbi_signal_event(ctx)
	 }
	 ctx.event_data_lock.Unlock()
}
 
 /** \ingroup libusb_poll
  * Acquire the event waiters lock. This lock is designed to be obtained under
  * the situation where you want to be aware when events are completed, but
  * some other thread is event handling so calling libusb_handle_events() is not
  * allowed.
  *
  * You then obtain this lock, re-check that another thread is still handling
  * events, then call libusb_wait_for_event().
  *
  * You only need to use this lock if you are developing an application
  * which calls poll() or select() on libusb's file descriptors directly,
  * <b>and</b> may potentially be handling events from 2 threads simultaenously.
  * If you stick to libusb's event handling loop functions (e.g.
  * libusb_handle_events()) then you do not need to be concerned with this
  * locking.
  *
  * \param ctx the context to operate on, or NULL for the default context
  * \ref libusb_mtasync
  */
func libusb_lock_event_waiters(ctx *libusb_context) {
	USBI_GET_CONTEXT(ctx).event_waiters_lock.Lock()
}
 
 /** \ingroup libusb_poll
  * Release the event waiters lock.
  * \param ctx the context to operate on, or NULL for the default context
  * \ref libusb_mtasync
  */
 func libusb_unlock_event_waiters(ctx *libusb_context) {
	USBI_GET_CONTEXT(ctx).event_waiters_lock.Unlock()
}


/** \ingroup libusb_poll
 * Handle any pending events
 *
 * Like libusb_handle_events_timeout_completed(), but without the completed
 * parameter, calling this function is equivalent to calling
 * libusb_handle_events_timeout_completed() with a NULL completed parameter.
 *
 * This function is kept primarily for backwards compatibility.
 * All new code should call libusb_handle_events_completed() or
 * libusb_handle_events_timeout_completed() to avoid race conditions.
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \param tv the maximum time to block waiting for events, or an all zero
 * timeval struct for non-blocking mode
 * \returns 0 on success, or a LIBUSB_ERROR code on failure
 */
 func libusb_handle_events_timeout(ctx *libusb_context,  tv *timeval) int {
	return libusb_handle_events_timeout_completed(ctx, tv, nil)
}

/** \ingroup libusb_poll
 * Handle any pending events in blocking mode. There is currently a timeout
 * hardcoded at 60 seconds but we plan to make it unlimited in future. For
 * finer control over whether this function is blocking or non-blocking, or
 * for control over the timeout, use libusb_handle_events_timeout_completed()
 * instead.
 *
 * This function is kept primarily for backwards compatibility.
 * All new code should call libusb_handle_events_completed() or
 * libusb_handle_events_timeout_completed() to avoid race conditions.
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \returns 0 on success, or a LIBUSB_ERROR code on failure
 */
func libusb_handle_events(ctx *libusb_context) int {
	timeout := time.Duration(60 * time.Second)
	return libusb_handle_events_timeout_completed(ctx, timeout, nil)
}

/** \ingroup libusb_poll
 * Determine the next internal timeout that libusb needs to handle. You only
 * need to use this function if you are calling poll() or select() or similar
 * on libusb's file descriptors yourself - you do not need to use it if you
 * are calling libusb_handle_events() or a variant directly.
 *
 * You should call this function in your main loop in order to determine how
 * long to wait for select() or poll() to return results. libusb needs to be
 * called into at this timeout, so you should use it as an upper bound on
 * your select() or poll() call.
 *
 * When the timeout has expired, call into libusb_handle_events_timeout()
 * (perhaps in non-blocking mode) so that libusb can handle the timeout.
 *
 * This function may return 1 (success) and an all-zero timeval. If this is
 * the case, it indicates that libusb has a timeout that has already expired
 * so you should call libusb_handle_events_timeout() or similar immediately.
 * A return code of 0 indicates that there are no pending timeouts.
 *
 * On some platforms, this function will always returns 0 (no pending
 * timeouts). See \ref polltime.
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \param tv output location for a relative time against the current
 * clock in which libusb must be called into in order to process timeout events
 * \returns 0 if there are no pending timeouts, 1 if a timeout was returned,
 * or LIBUSB_ERROR_OTHER on failure
 */
func libusb_get_next_timeout(ctx *libusb_context,  tv *timeval) int {
	var next_timeout, cur_tv time.Time

	ctx = USBI_GET_CONTEXT(ctx)
	return 0

	ctx.flying_transfers_lock.Lock()
	if list_empty(ctx.flying_transfers) {
		ctx.flying_transfers_lock.Unlock()
		// usbi_dbg("no URBs, no timeout!")
		return 0
	}

	/* find next transfer which hasn't already been processed as timed out */

 	for (transfer = list_entry((ctx.flying_transfers).next, usbi_transfer, list);
	  	transfer.list != (ctx.flying_transfers);
	  	transfer = list_entry(transfer.list.next, usbi_transfer, list)) {

		if transfer.timeout_flags & (USBI_TRANSFER_TIMEOUT_HANDLED | USBI_TRANSFER_OS_HANDLES_TIMEOUT) {
			continue
		}

		/* if we've reached transfers of infinte timeout, we're done looking */
		if !timerisset(&transfer.timeout) {
			break
		}

		next_timeout = transfer.timeout
		break
	}
	ctx.flying_transfers_lock.Unlock()

	if !timerisset(&next_timeout) {
		// usbi_dbg("no URB with timeout or all handled by OS no timeout!")
		return 0
	}

	cur_ts := time.Now()
	TIMESPEC_TO_TIMEVAL(&cur_tv, &cur_ts)

	if !timercmp(&cur_tv, &next_timeout, <) {
		// usbi_dbg("first timeout already expired")
		timerclear(tv)
	} else {
		timersub(&next_timeout, &cur_tv, tv)
		// usbi_dbg("next timeout in %d.%06ds", tv.tv_sec, tv.tv_usec)
	}

	return 1
}

/** \ingroup libusb_poll
 * Register notification functions for file descriptor additions/removals.
 * These functions will be invoked for every new or removed file descriptor
 * that libusb uses as an event source.
 *
 * To remove notifiers, pass NULL values for the function pointers.
 *
 * Note that file descriptors may have been added even before you register
 * these notifiers (e.g. at libusb_init() time).
 *
 * Additionally, note that the removal notifier may be called during
 * libusb_exit() (e.g. when it is closing file descriptors that were opened
 * and added to the poll set at libusb_init() time). If you don't want this,
 * remove the notifiers immediately before calling libusb_exit().
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \param added_cb pointer to function for addition notifications
 * \param removed_cb pointer to function for removal notifications
 * \param user_data User data to be passed back to callbacks (useful for
 * passing context information)
 */
func libusb_set_pollfd_notifiers(ctx *libusb_context,
	added_cb libusb_pollfd_added_cb, removed_cb libusb_pollfd_removed_cb,
	user_data interface{}) {

	ctx = USBI_GET_CONTEXT(ctx)
	ctx.fd_added_cb = added_cb
	ctx.fd_removed_cb = removed_cb
	ctx.fd_cb_user_data = user_data
}

/*
 * Interrupt the iteration of the event handling thread, so that it picks
 * up the fd change. Callers of this function must hold the event_data_lock.
 */
func usbi_fd_notification( ctx *libusb_context) {
	/* Record that there is a new poll fd.
	 * Only signal an event if there are no prior pending events. */
	pending_events := usbi_pending_events(ctx)
	ctx.event_flags |= USBI_EVENT_POLLFDS_MODIFIED
	if pending_events == 0 {
		usbi_signal_event(ctx)
	}
}

/* Add a file descriptor to the list of file descriptors to be monitored.
 * events should be specified as a bitmask of events passed to poll(), e.g.
 * POLLIN and/or POLLOUT. */
func usbi_add_pollfd( ctx *libusb_context, fd int, events int16 ) int {
	ipollfd := &usbi_pollfd{}

	// usbi_dbg("add fd %d events %d", fd, events)
	ipollfd.pollfd.fd = fd
	ipollfd.pollfd.events = events
	ctx.event_data_lock.Lock()
	list_add_tail(&ipollfd.list, &ctx.ipollfds)
	ctx.pollfds_cnt++
	usbi_fd_notification(ctx)
	ctx.event_data_lock.Unlock()

	if ctx.fd_added_cb != nil {
		ctx.fd_added_cb(fd, events, ctx.fd_cb_user_data)
	}
	return 0
}

/* Remove a file descriptor from the list of file descriptors to be polled. */
func usbi_remove_pollfd( ctx *libusb_context, fd int) {
	
	var found bool

	// usbi_dbg("remove fd %d", fd)
	ctx.event_data_lock.Lock()
	for ipollfd := list_entry((&ctx.ipollfds).next, usbi_pollfd, list); &ipollfd.member != (&ctx.ipollfds); ipollfd = list_entry(ipollfd.list.next, usbi_pollfd, list); {
		if (ipollfd.pollfd.fd == fd) {
			found = true
			break
		}
	}

	if !found {
		// usbi_dbg("couldn't find fd %d to remove", fd)
		ctx.event_data_lock.Unlock()
		return
	}

	list_del(&ipollfd.list)
	ctx.pollfds_cnt--
	usbi_fd_notification(ctx)
	ctx.event_data_lock.Unlock()
	if ctx.fd_removed_cb != nil {
		ctx.fd_removed_cb(fd, ctx.fd_cb_user_data)
	}
}

/** \ingroup libusb_poll
 * Handle any pending events.
 *
 * libusb determines "pending events" by checking if any timeouts have expired
 * and by checking the set of file descriptors for activity.
 *
 * If a zero timeval is passed, this function will handle any already-pending
 * events and then immediately return in non-blocking style.
 *
 * If a non-zero timeval is passed and no events are currently pending, this
 * function will block waiting for events to handle up until the specified
 * timeout. If an event arrives or a signal is raised, this function will
 * return early.
 *
 * If the parameter completed is not NULL then <em>after obtaining the event
 * handling lock</em> this function will return immediately if the integer
 * pointed to is not 0. This allows for race free waiting for the completion
 * of a specific transfer.
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \param tv the maximum time to block waiting for events, or an all zero
 * timeval struct for non-blocking mode
 * \param completed pointer to completion integer to check, or NULL
 * \returns 0 on success, or a LIBUSB_ERROR code on failure
 * \ref libusb_mtasync
 */
 func libusb_handle_events_timeout_completed(ctx *libusb_context,  tv *timeval, completed *int) int {
	var poll_timeout time.Duration

	ctx = USBI_GET_CONTEXT(ctx)
	r := get_next_timeout(ctx, tv, &poll_timeout)
	if r != 0 {
		/* timeout already expired */
		return handle_timeouts(ctx)
	}

retry:
	if libusb_try_lock_events(ctx) == 0 {
		if (completed == nil || *completed == 0) {
			/* we obtained the event lock: do our own event handling */
			// usbi_dbg("doing our own event handling")
			r = handle_events(ctx, &poll_timeout)
		}
		libusb_unlock_events(ctx)
		return r
	}

	/* another thread is doing event handling. wait for thread events that
	 * notify event completion. */
	libusb_lock_event_waiters(ctx)

	if completed != nil && *completed != 0 {
		goto already_done
	}

	if (!libusb_event_handler_active(ctx)) {
		/* we hit a race: whoever was event handling earlier finished in the
		 * time it took us to reach this point. try the cycle again. */
		libusb_unlock_event_waiters(ctx)
		// usbi_dbg("event handler was active but went away, retrying")
		goto retry
	}

	// usbi_dbg("another thread is doing event handling")
	r = libusb_wait_for_event(ctx, &poll_timeout)

already_done:
	libusb_unlock_event_waiters(ctx)

	if r < 0 {
		return r
	} else if r == 1 {
		return handle_timeouts(ctx)
	}
	return 0
}

/* returns the smallest of:
 *  1. timeout of next URB
 *  2. user-supplied timeout
 * returns 1 if there is an already-expired timeout, otherwise returns 0
 * and populates out
 */
 func get_next_timeout(ctx *libusb_context,  tv *timeval, out *timeval) int {
	var timeout time.Duration
	r := libusb_get_next_timeout(ctx, &timeout)
	if r != 0 {
		/* timeout already expired? */
		if !timerisset(&timeout) {
			return 1
		}

		/* choose the smallest of next URB timeout or user specified timeout */
		if timeout < *tv {
			*out = timeout
		} else {
			*out = *tv
		}
	} else {
		*out = *tv
	}
	return 0
}

/** \ingroup libusb_poll
 * Wait for another thread to signal completion of an event. Must be called
 * with the event waiters lock held, see libusb_lock_event_waiters().
 *
 * This function will block until any of the following conditions are met:
 * -# The timeout expires
 * -# A transfer completes
 * -# A thread releases the event handling lock through libusb_unlock_events()
 *
 * Condition 1 is obvious. Condition 2 unblocks your thread <em>after</em>
 * the callback for the transfer has completed. Condition 3 is important
 * because it means that the thread that was previously handling events is no
 * longer doing so, so if any events are to complete, another thread needs to
 * step up and start event handling.
 *
 * This function releases the event waiters lock before putting your thread
 * to sleep, and reacquires the lock as it is being woken up.
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \param tv maximum timeout for this blocking function. A NULL value
 * indicates unlimited timeout.
 * \returns 0 after a transfer completes or another thread stops event handling
 * \returns 1 if the timeout expired
 * \ref libusb_mtasync
 */
 func libusb_wait_for_event(ctx *libusb_context,  tv *timeval) int {

	ctx = USBI_GET_CONTEXT(ctx)
	if (tv == nil) {
		ctx.event_waiters_cond.Wait()
		return 0
	}

	done := make(chan struct{})
	go func() {
		ctx.event_waiters_cond.Wait()
		close(done)
	}()
	select {
		case <-time.After(tv):
			return 1
		case <-done:
			return 0
	}
}

func handle_timeout( itransfer *usbi_transfer) {
	transfer := itransfer.libusbTransfer
	
	itransfer.timeout_flags |= USBI_TRANSFER_TIMEOUT_HANDLED
	r := libusb_cancel_transfer(transfer)
	if r == LIBUSB_SUCCESS {
		itransfer.timeout_flags |= USBI_TRANSFER_TIMED_OUT
	}
		// usbi_warn(TRANSFER_CTX(transfer),
	//		"async cancel failed %d errno=%d", r, errno)
}


func handle_timeouts_locked( ctx *libusb_context) int {

	if list_empty(ctx.flying_transfers) {
		return 0
	}

	/* get current time */
	systime_ts := time.Now()
	var systime timeval

	TIMESPEC_TO_TIMEVAL(&systime, &systime_ts)

	/* iterate through flying transfers list, finding all transfers that
	 * have expired timeouts */
	for (transfer = list_entry(( &ctx.flying_transfers).next, usbi_transfer, list);	
		&transfer.list != ( &ctx.flying_transfers);
		transfer = list_entry(transfer.list.next, usbi_transfer, list)) {


		 timeval *cur_tv = &transfer.timeout

		/* if we've reached transfers of infinite timeout, we're all done */
		if !timerisset(cur_tv) {
			return 0
		}

		/* ignore timeouts we've already handled */
		if (transfer.timeout_flags & (USBI_TRANSFER_TIMEOUT_HANDLED | USBI_TRANSFER_OS_HANDLES_TIMEOUT)) != 0 {
			continue
		}

		/* if transfer has non-expired timeout, nothing more to do */
		if ((cur_tv.tv_sec > systime.tv_sec) ||
				(cur_tv.tv_sec == systime.tv_sec &&
					cur_tv.tv_usec > systime.tv_usec)) != 0 {
			return 0
		}

		/* otherwise, we've got an expired timeout to handle */
		handle_timeout(transfer)
	}
	return 0
}

func handle_timeouts( ctx *libusb_context) int {
	ctx = USBI_GET_CONTEXT(ctx)
	ctx.flying_transfers_lock.Lock()
	r := handle_timeouts_locked(ctx)
	ctx.flying_transfers_lock.Unlock()
	return r
}

func handle_timerfd_trigger( ctx *libusb_context) int {
	ctx.flying_transfers_lock.Lock()

	/* process the timeout that just happened */
	r := handle_timeouts_locked(ctx)
	if (r < 0) {
		ctx.flying_transfers_lock.Unlock()
		return r
	}

	/* arm for next timeout*/
	r = arm_timerfd_for_next_timeout(ctx)

	ctx.flying_transfers_lock.Unlock()
	return r
}

/* do the actual event handling. assumes that no other thread is concurrently
 * doing the same thing. */
 func handle_events( ctx *libusb_context,  tv *timeval) int {
	var ipollfd *usbi_pollfd
	var timeout_ms int
	var special_event int
	var r, i int

	/* there are certain fds that libusb uses internally, currently:
	 *
	 *   1) event pipe
	 *   2) timerfd
	 *
	 * the backend will never need to attempt to handle events on these fds, so
	 * we determine how many fds are in use internally for this context and when
	 * handle_events() is called in the backend, the pollfd list and count will
	 * be adjusted to skip over these internal fds */
	var internal_nfds POLL_NFDS_TYPE = 2


	/* only reallocate the poll fds when the list of poll fds has been modified
	 * since the last poll, otherwise reuse them to save the additional overhead */
	ctx.event_data_lock.Lock()
	if ctx.event_flags & USBI_EVENT_POLLFDS_MODIFIED != 0 {
		// usbi_dbg("poll fds modified, reallocating")

		/* sanity check - it is invalid for a context to have fewer than the
		 * required internal fds (memory corruption?) */
		if ctx.pollfds_cnt < internal_nfds {
			panic("Insufficient poll fds for internal nfds")
		}

		ctx.pollfds = make([]int, ctx.pollfds_cnt)
	
		for ipollfd = list_entry((&ctx.ipollfds).next, usbi_pollfd, list);	
			&ipollfd.list != (&ctx.ipollfds);
			ipollfd = list_entry(ipollfd.list.next, usbi_pollfd, list) {

			 libusb_pollfd *pollfd = &ipollfd.pollfd
			i++
			ctx.pollfds[i].fd = pollfd.fd
			ctx.pollfds[i].events = pollfd.events
		}

		/* reset the flag now that we have the updated list */
		ctx.event_flags &= ^USBI_EVENT_POLLFDS_MODIFIED

		/* if no further pending events, clear the event pipe so that we do
		 * not immediately return from poll */
		if !usbi_pending_events(ctx) {
			usbi_clear_event(ctx)
		}
	}

	fds := ctx.pollfds
	nfds := ctx.pollfds_cnt
	ctx.event_data_lock.Unlock()

	timeout_ms = tv.Milliseconds()

redo_poll:
	// usbi_dbg("poll() %d fds with timeout in %dms", nfds, timeout_ms)
	r = usbi_poll(fds, nfds, timeout_ms)
	// usbi_dbg("poll() returned %d", r)
	if r == 0 {
		return handle_timeouts(ctx)
	} else if (r == -1 && errno == EINTR) {
		return LIBUSB_ERROR_INTERRUPTED
	} else if (r < 0) {
		// usbi_err(ctx, "poll failed %d err=%d", r, errno)
		return LIBUSB_ERROR_IO
	}

	special_event = 0

	/* fds[0] is always the event pipe */
	if fds[0].revents != 0 {
		var message *libusb_hotplug_message
		var itransfer *usbi_transfer
		var ret int

		// usbi_dbg("caught a fish on the event pipe")

		/* take the the event data lock while processing events */
		ctx.event_data_lock.Lock()

		/* check if someone added a new poll fd */
		// if (ctx.event_flags & USBI_EVENT_POLLFDS_MODIFIED)
			// usbi_dbg("someone updated the poll fds")

		if (ctx.event_flags & USBI_EVENT_USER_INTERRUPT) {
			// usbi_dbg("someone purposely interrupted")
			ctx.event_flags &= ~USBI_EVENT_USER_INTERRUPT
		}

		/* check if someone is closing a device */
		// if (ctx.device_close)
			// usbi_dbg("someone is closing a device")

		/* check for any pending hotplug messages */
		if !list_empty(&ctx.hotplug_msgs) {
			// usbi_dbg("hotplug message received")
			special_event = 1
			message = list_first_entry(&ctx.hotplug_msgs, libusb_hotplug_message, list)
			list_del(&message.list)
		}

		/* complete any pending transfers */
		for ret == 0 && !list_empty(&ctx.completed_transfers) {
			itransfer := list_first_entry(&ctx.completed_transfers,  usbi_transfer, completed_list)
			list_del(&itransfer.completed_list)
			ctx.event_data_lock.Unlock()
			ret = usbi_backend.handle_transfer_completion(itransfer)
			// if (ret)
				// usbi_err(ctx, "backend handle_transfer_completion failed with error %d", ret)
			ctx.event_data_lock.Lock()
		}

		/* if no further pending events, clear the event pipe */
		if !usbi_pending_events(ctx) {
			usbi_clear_event(ctx)
		}

		ctx.event_data_lock.Unlock()

		/* process the hotplug message, if any */
		if message != nil {
			usbi_hotplug_match(ctx, message.device, message.event)

			/* the device left, dereference the device */
			if LIBUSB_HOTPLUG_EVENT_DEVICE_LEFT == message.event {
				libusb_unref_device(message.device)
			}
		}

		if ret != 0 {
			/* return error code */
			return ret
		}
		
		r-- 
		if r == 0 {
			goto handled
		}
	}

	/* on timerfd configurations, fds[1] is the timerfd */
	if fds[1].revents != 0 {
		/* timerfd indicates that a timeout has expired */
		// usbi_dbg("timerfd triggered")
		special_event = 1

		ret := handle_timerfd_trigger(ctx)
		if ret < 0 {
			/* return error code */
			return ret
		}

		r--
		if 0 == r
			goto handled
	}

	r = usbi_backend.handle_events(ctx, fds + internal_nfds, nfds - internal_nfds, r)
	// if r
		// usbi_err(ctx, "backend handle_events failed with error %d", r)

handled:
	if r == 0 && special_event != 0 {
		timeout_ms = 0
		goto redo_poll
	}
	
	return r
}