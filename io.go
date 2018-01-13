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

	current_time.tv_sec += timeout / 1000;
	current_time.tv_nsec += (timeout % 1000) * 1000000;

	while (current_time.tv_nsec >= 1000000000) {
		current_time.tv_nsec -= 1000000000;
		current_time.tv_sec++;
	}

	TIMESPEC_TO_TIMEVAL(&transfer.timeout, &current_time);
	return 0;
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

	for transfer = list_entry(ctx.flying_transfers.next, usbi_transfer, list); 
		&transfer.list != ctx.flying_transfers; 
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

			// usbi_dbg("next timeout originally %dms", USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer).timeout);
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
	for cur = list_entry(ctx.flying_transfers.next, usbi_transfer, list); 
		&cur.list != ctx.flying_transfers; 
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
			// USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer.libusbTransfer.timeout);
		r = timerfd_settime(ctx.timerfd, TFD_TIMER_ABSTIME, &it, nil)
		if (r < 0) {
			// usbi_warn(ctx, "failed to arm first timerfd (errno %d)", errno);
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
	
	ctx := transfer.libusbTransfer.dev_handle.dev.ctx;
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
 * The returned transfer is not specially initialized for isochronous I/O;
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
	itransfer.num_iso_packets = iso_packets;

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
	//ctx = USBI_GET_CONTEXT(ctx);
	//return usbi_using_timerfd(ctx);
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
 
	 // usbi_dbg("transfer %p", transfer);
 
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

	// usbi_dbg("transfer %p", transfer );
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
		// usbi_err(ITRANSFER_CTX(itransfer), "failed to set timer for next timeout, errno=%d", errno);

	itransfer.lock.Lock()
	itransfer.state_flags &= ~USBI_TRANSFER_IN_FLIGHT
	itransfer.lock.Unlock()

	if status == LIBUSB_TRANSFER_COMPLETED && transfer.flags & LIBUSB_TRANSFER_SHORT_NOT_OK {
		rqlen := transfer.length
		if transfer.type == LIBUSB_TRANSFER_TYPE_CONTROL {
			rqlen -= LIBUSB_CONTROL_SETUP_SIZE
		}
		if rqlen != itransfer.transferred {
			// usbi_dbg("interpreting short transfer as error");
			status = LIBUSB_TRANSFER_ERROR
		}
	}

	flags = transfer.flags
	transfer.status = status
	transfer.actual_length = itransfer.transferred;
	// usbi_dbg("transfer %p has callback %p", transfer, transfer.callback);
	if transfer.callback != nil {
		transfer.callback(transfer)
	}

	libusb_unref_device(dev_handle.dev)
	return r
}