/* -*- Mode: C; indent-tabs-mode:t ; c-basic-offset:8 -*- */

void usbi_io_exit(struct libusb_context *ctx)
{
	usbi_remove_pollfd(ctx, ctx->event_pipe[0]);
	usbi_close(ctx->event_pipe[0]);
	usbi_close(ctx->event_pipe[1]);
	usbi_remove_pollfd(ctx, ctx->timerfd);
	close(ctx->timerfd);
}

static int calculate_timeout(struct usbi_transfer *transfer)
{
	int r;
	struct timespec current_time;
	uint timeout = transfer.libusbTransfer->timeout;

	if (!timeout)
		return 0;

	r = usbi_backend->clock_gettime(USBI_CLOCK_MONOTONIC, &current_time);
	if (r < 0) {
		// usbi_err(ITRANSFER_CTX(transfer),
			"failed to read monotonic clock, errno=%d", errno);
		return r;
	}

	current_time.tv_sec += timeout / 1000;
	current_time.tv_nsec += (timeout % 1000) * 1000000;

	while (current_time.tv_nsec >= 1000000000) {
		current_time.tv_nsec -= 1000000000;
		current_time.tv_sec++;
	}

	TIMESPEC_TO_TIMEVAL(&transfer->timeout, &current_time);
	return 0;
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

static int disarm_timerfd(struct libusb_context *ctx)
{
	const struct itimerspec disarm_timer = { { 0, 0 }, { 0, 0 } };
	int r;

	// usbi_dbg("");
	r = timerfd_settime(ctx->timerfd, 0, &disarm_timer, NULL);
	if (r < 0)
		return LIBUSB_ERROR_OTHER;
	else
		return 0;
}

/* iterates through the flying transfers, and rearms the timerfd based on the
 * next upcoming timeout.
 * must be called with flying_list locked.
 * returns 0 on success or a LIBUSB_ERROR code on failure.
 */
static int arm_timerfd_for_next_timeout(struct libusb_context *ctx)
{
	struct usbi_transfer *transfer;

	list_for_each_entry(transfer, &ctx->flying_transfers, list, struct usbi_transfer) {
		struct timeval *cur_tv = &transfer->timeout;

		/* if we've reached transfers of infinite timeout, then we have no
		 * arming to do */
		if (!timerisset(cur_tv))
			goto disarm;

		/* act on first transfer that has not already been handled */
		if (!(transfer->timeout_flags & (USBI_TRANSFER_TIMEOUT_HANDLED | USBI_TRANSFER_OS_HANDLES_TIMEOUT))) {
			int r;
			const struct itimerspec it = { {0, 0},
				{ cur_tv->tv_sec, cur_tv->tv_usec * 1000 } };
			// usbi_dbg("next timeout originally %dms", USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer)->timeout);
			r = timerfd_settime(ctx->timerfd, TFD_TIMER_ABSTIME, &it, NULL);
			if (r < 0)
				return LIBUSB_ERROR_OTHER;
			return 0;
		}
	}

disarm:
	return disarm_timerfd(ctx);
}

/* add a transfer to the (timeout-sorted) active transfers list.
 * This function will return non 0 if fails to update the timer,
 * in which case the transfer is *not* on the flying_transfers list. */
static int add_to_flying_list(struct usbi_transfer *transfer)
{
	struct usbi_transfer *cur;
	struct timeval *timeout = &transfer->timeout;
	struct libusb_context *ctx = transfer.libusbTransfer.dev_handle.dev.ctx;
	int r;
	int first = 1;

	r = calculate_timeout(transfer);
	if (r)
		return r;

	/* if we have no other flying transfers, start the list with this one */
	if (list_empty(&ctx->flying_transfers)) {
		list_add(&transfer->list, &ctx->flying_transfers);
		goto out;
	}

	/* if we have infinite timeout, append to end of list */
	if (!timerisset(timeout)) {
		list_add_tail(&transfer->list, &ctx->flying_transfers);
		/* first is irrelevant in this case */
		goto out;
	}

	/* otherwise, find appropriate place in list */
	list_for_each_entry(cur, &ctx->flying_transfers, list, struct usbi_transfer) {
		/* find first timeout that occurs after the transfer in question */
		struct timeval *cur_tv = &cur->timeout;

		if (!timerisset(cur_tv) || (cur_tv->tv_sec > timeout->tv_sec) ||
				(cur_tv->tv_sec == timeout->tv_sec &&
					cur_tv->tv_usec > timeout->tv_usec)) {
			list_add_tail(&transfer->list, &cur->list);
			goto out;
		}
		first = 0;
	}
	/* first is 0 at this stage (list not empty) */

	/* otherwise we need to be inserted at the end */
	list_add_tail(&transfer->list, &ctx->flying_transfers);
out:
	if (first && timerisset(timeout)) {
		/* if this transfer has the lowest timeout of all active transfers,
		 * rearm the timerfd with this transfer's timeout */
		const struct itimerspec it = { {0, 0},
			{ timeout->tv_sec, timeout->tv_usec * 1000 } };
		// usbi_dbg("arm timerfd for timeout in %dms (first in line)",
			// USBI_TRANSFER_TO_LIBUSB_TRANSFER(transfer.libusbTransfer->timeout);
		r = timerfd_settime(ctx->timerfd, TFD_TIMER_ABSTIME, &it, NULL);
		if (r < 0) {
			// usbi_warn(ctx, "failed to arm first timerfd (errno %d)", errno);
			r = LIBUSB_ERROR_OTHER;
		}
	}

	if (r)
		list_del(&transfer->list);

	return r;
}

/* remove a transfer from the active transfers list.
 * This function will *always* remove the transfer from the
 * flying_transfers list. It will return a LIBUSB_ERROR code
 * if it fails to update the timer for the next timeout. */
static int remove_from_flying_list(struct usbi_transfer *transfer)
{
	struct libusb_context *ctx = transfer.libusbTransfer.dev_handle.dev.ctx;
	int rearm_timerfd;
	int r = 0;

	&ctx->flying_transfers_lock.Lock();
	rearm_timerfd = (timerisset(&transfer->timeout) &&
		list_first_entry(&ctx->flying_transfers, struct usbi_transfer, list) == transfer);
	list_del(&transfer->list);
	if (rearm_timerfd)
		r = arm_timerfd_for_next_timeout(ctx);
	&ctx->flying_transfers_lock.Unlock();

	return r;
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
int  libusb_submit_transfer(struct libusb_transfer *transfer)
{
	struct usbi_transfer *itransfer = transfer.usbiTransfer
	struct libusb_context *ctx = transfer.dev_handle.dev.ctx;
	int r;

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
	&ctx->flying_transfers_lock.Lock();
	&itransfer->lock.Lock();
	if (itransfer->state_flags & USBI_TRANSFER_IN_FLIGHT) {
		&ctx->flying_transfers_lock.Unlock();
		&itransfer->lock.Unlock();
		return LIBUSB_ERROR_BUSY;
	}
	itransfer->transferred = 0;
	itransfer->state_flags = 0;
	itransfer->timeout_flags = 0;
	r = add_to_flying_list(itransfer);
	if (r) {
		&ctx->flying_transfers_lock.Unlock();
		&itransfer->lock.Unlock();
		return r;
	}
	/*
	 * We must release the flying transfers lock here, because with
	 * some backends the submit_transfer method is synchroneous.
	 */
	&ctx->flying_transfers_lock.Unlock();

	r = usbi_backend->submit_transfer(itransfer);
	if (r == LIBUSB_SUCCESS) {
		itransfer->state_flags |= USBI_TRANSFER_IN_FLIGHT;
		/* keep a reference to this device */
		libusb_ref_device(transfer->dev_handle->dev);
	}
	&itransfer->lock.Unlock();

	if (r != LIBUSB_SUCCESS)
		remove_from_flying_list(itransfer);

	return r;
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
int  libusb_cancel_transfer(struct libusb_transfer *transfer)
{
	struct usbi_transfer *itransfer = transfer.usbiTransfer
	int r;

	// usbi_dbg("transfer %p", transfer );
	&itransfer->lock.Lock();
	if (!(itransfer->state_flags & USBI_TRANSFER_IN_FLIGHT)
			|| (itransfer->state_flags & USBI_TRANSFER_CANCELLING)) {
		r = LIBUSB_ERROR_NOT_FOUND;
		goto out;
	}
	r = usbi_backend->cancel_transfer(itransfer);
	if (r < 0) {
		if (r != LIBUSB_ERROR_NOT_FOUND &&
		    r != LIBUSB_ERROR_NO_DEVICE)
			// usbi_err(TRANSFER_CTX(transfer),
				"cancel transfer failed error %d", r);
		else
			// usbi_dbg("cancel transfer failed error %d", r);

		if (r == LIBUSB_ERROR_NO_DEVICE)
			itransfer->state_flags |= USBI_TRANSFER_DEVICE_DISAPPEARED;
	}

	itransfer->state_flags |= USBI_TRANSFER_CANCELLING;

out:
	&itransfer->lock.Unlock();
	return r;
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
void  libusb_transfer_set_stream_id(
	struct libusb_transfer *transfer, uint32 stream_id)
{
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
uint32  libusb_transfer_get_stream_id(struct libusb_transfer *transfer)
{
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
int usbi_handle_transfer_completion(struct usbi_transfer *itransfer,
	libusb_transfer_status status)
{
	struct libusb_transfer *transfer = itransfer.libusbTransfer
	struct libusb_device_handle *dev_handle = transfer->dev_handle;
	uint8 flags;
	int r;

	r = remove_from_flying_list(itransfer);
	if (r < 0)
		// usbi_err(ITRANSFER_CTX(itransfer), "failed to set timer for next timeout, errno=%d", errno);

	&itransfer->lock.Lock();
	itransfer->state_flags &= ~USBI_TRANSFER_IN_FLIGHT;
	&itransfer->lock.Unlock();

	if (status == LIBUSB_TRANSFER_COMPLETED
			&& transfer->flags & LIBUSB_TRANSFER_SHORT_NOT_OK) {
		int rqlen = transfer->length;
		if (transfer->type == LIBUSB_TRANSFER_TYPE_CONTROL)
			rqlen -= LIBUSB_CONTROL_SETUP_SIZE;
		if (rqlen != itransfer->transferred) {
			// usbi_dbg("interpreting short transfer as error");
			status = LIBUSB_TRANSFER_ERROR;
		}
	}

	flags = transfer->flags;
	transfer->status = status;
	transfer->actual_length = itransfer->transferred;
	// usbi_dbg("transfer %p has callback %p", transfer, transfer->callback);
	if (transfer->callback)
		transfer->callback(transfer);

	libusb_unref_device(dev_handle->dev);
	return r;
}

/* Similar to usbi_handle_transfer_completion() but exclusively for transfers
 * that were asynchronously cancelled. The same concerns w.r.t. freeing of
 * transfers exist here.
 * Do not call this function with the usbi_transfer lock held. User-specified
 * callback functions may attempt to directly resubmit the transfer, which
 * will attempt to take the lock. */
int usbi_handle_transfer_cancellation(struct usbi_transfer *transfer)
{
	struct libusb_context *ctx = transfer.libusbTransfer.dev_handle.dev.ctx;
	uint8 timed_out;

	&ctx->flying_transfers_lock.Lock();
	timed_out = transfer->timeout_flags & USBI_TRANSFER_TIMED_OUT;
	&ctx->flying_transfers_lock.Unlock();

	/* if the URB was cancelled due to timeout, report timeout to the user */
	if (timed_out) {
		// usbi_dbg("detected timeout cancellation");
		return usbi_handle_transfer_completion(transfer, LIBUSB_TRANSFER_TIMED_OUT);
	}

	/* otherwise its a normal async cancel */
	return usbi_handle_transfer_completion(transfer, LIBUSB_TRANSFER_CANCELLED);
}

/* Add a completed transfer to the completed_transfers list of the
 * context and signal the event. The backend's handle_transfer_completion()
 * function will be called the next time an event handler runs. */
void usbi_signal_transfer_completion(struct usbi_transfer *transfer)
{
	struct libusb_context *ctx = transfer.libusbTransfer.dev_handle.dev.ctx;
	int pending_events;

	&ctx->event_data_lock.Lock();
	pending_events = usbi_pending_events(ctx);
	list_add_tail(&transfer->completed_list, &ctx->completed_transfers);
	if (!pending_events)
		usbi_signal_event(ctx);
	&ctx->event_data_lock.Unlock();
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
int  libusb_try_lock_events(libusb_context *ctx)
{
	int r;
	uint ru;
	ctx = USBI_GET_CONTEXT(ctx);

	/* is someone else waiting to close a device? if so, don't let this thread
	 * start event handling */
	&ctx->event_data_lock.Lock();
	ru = ctx->device_close;
	&ctx->event_data_lock.Unlock();
	if (ru) {
		// usbi_dbg("someone else is closing a device");
		return 1;
	}

	// GO problem: sync.Mutex doesn't have an equivalent to this
	r = usbi_mutex_trylock(&ctx->events_lock);
	if (r)
		return 1;

	ctx->event_handler_active = 1;
	return 0;
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
void  libusb_lock_events(libusb_context *ctx)
{
	ctx = USBI_GET_CONTEXT(ctx);
	&ctx->events_lock.Lock();
	ctx->event_handler_active = 1;
}

/** \ingroup libusb_poll
 * Release the lock previously acquired with libusb_try_lock_events() or
 * libusb_lock_events(). Releasing this lock will wake up any threads blocked
 * on libusb_wait_for_event().
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \ref libusb_mtasync
 */
void  libusb_unlock_events(libusb_context *ctx)
{
	ctx = USBI_GET_CONTEXT(ctx);
	ctx->event_handler_active = 0;
	&ctx->events_lock.Unlock();

	/* FIXME: perhaps we should be a bit more efficient by not broadcasting
	 * the availability of the events lock when we are modifying pollfds
	 * (check ctx->device_close)? */
	&ctx->event_waiters_lock.Lock();
	ctx.event_waiters_cond.Broadcast()
	&ctx->event_waiters_lock.Unlock();
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
int  libusb_event_handling_ok(libusb_context *ctx)
{
	uint r;
	ctx = USBI_GET_CONTEXT(ctx);

	/* is someone else waiting to close a device? if so, don't let this thread
	 * continue event handling */
	&ctx->event_data_lock.Lock();
	r = ctx->device_close;
	&ctx->event_data_lock.Unlock();
	if (r) {
		// usbi_dbg("someone else is closing a device");
		return 0;
	}

	return 1;
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
int  libusb_event_handler_active(libusb_context *ctx)
{
	uint r;
	ctx = USBI_GET_CONTEXT(ctx);

	/* is someone else waiting to close a device? if so, don't let this thread
	 * start event handling -- indicate that event handling is happening */
	&ctx->event_data_lock.Lock();
	r = ctx->device_close;
	&ctx->event_data_lock.Unlock();
	if (r) {
		// usbi_dbg("someone else is closing a device");
		return 1;
	}

	return ctx->event_handler_active;
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
void  libusb_interrupt_event_handler(libusb_context *ctx)
{
	ctx = USBI_GET_CONTEXT(ctx);

	// usbi_dbg("");
	&ctx->event_data_lock.Lock();
	if (!usbi_pending_events(ctx)) {
		ctx->event_flags |= USBI_EVENT_USER_INTERRUPT;
		usbi_signal_event(ctx);
	}
	&ctx->event_data_lock.Unlock();
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
void  libusb_lock_event_waiters(libusb_context *ctx)
{
	ctx = USBI_GET_CONTEXT(ctx);
	&ctx->event_waiters_lock.Lock();
}

/** \ingroup libusb_poll
 * Release the event waiters lock.
 * \param ctx the context to operate on, or NULL for the default context
 * \ref libusb_mtasync
 */
void  libusb_unlock_event_waiters(libusb_context *ctx)
{
	ctx = USBI_GET_CONTEXT(ctx);
	&ctx->event_waiters_lock.Unlock();
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
int  libusb_wait_for_event(libusb_context *ctx, struct timeval *tv)
{
	int r;

	ctx = USBI_GET_CONTEXT(ctx);
	if (tv == NULL) {
		ctx.event_waiters_cond.Wait()
		return 0;
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

static void handle_timeout(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = itransfer.libusbTransfer
	int r;

	itransfer->timeout_flags |= USBI_TRANSFER_TIMEOUT_HANDLED;
	r = libusb_cancel_transfer(transfer);
	if (r == LIBUSB_SUCCESS)
		itransfer->timeout_flags |= USBI_TRANSFER_TIMED_OUT;
	else
		// usbi_warn(TRANSFER_CTX(transfer),
			"async cancel failed %d errno=%d", r, errno);
}

static int handle_timeouts_locked(struct libusb_context *ctx)
{
	int r;
	struct timespec systime_ts;
	struct timeval systime;
	struct usbi_transfer *transfer;

	if (list_empty(&ctx->flying_transfers))
		return 0;

	/* get current time */
	r = usbi_backend->clock_gettime(USBI_CLOCK_MONOTONIC, &systime_ts);
	if (r < 0)
		return r;

	TIMESPEC_TO_TIMEVAL(&systime, &systime_ts);

	/* iterate through flying transfers list, finding all transfers that
	 * have expired timeouts */
	list_for_each_entry(transfer, &ctx->flying_transfers, list, struct usbi_transfer) {
		struct timeval *cur_tv = &transfer->timeout;

		/* if we've reached transfers of infinite timeout, we're all done */
		if (!timerisset(cur_tv))
			return 0;

		/* ignore timeouts we've already handled */
		if (transfer->timeout_flags & (USBI_TRANSFER_TIMEOUT_HANDLED | USBI_TRANSFER_OS_HANDLES_TIMEOUT))
			continue;

		/* if transfer has non-expired timeout, nothing more to do */
		if ((cur_tv->tv_sec > systime.tv_sec) ||
				(cur_tv->tv_sec == systime.tv_sec &&
					cur_tv->tv_usec > systime.tv_usec))
			return 0;

		/* otherwise, we've got an expired timeout to handle */
		handle_timeout(transfer);
	}
	return 0;
}

static int handle_timeouts(struct libusb_context *ctx)
{
	int r;
	ctx = USBI_GET_CONTEXT(ctx);
	&ctx->flying_transfers_lock.Lock();
	r = handle_timeouts_locked(ctx);
	&ctx->flying_transfers_lock.Unlock();
	return r;
}

static int handle_timerfd_trigger(struct libusb_context *ctx)
{
	int r;

	&ctx->flying_transfers_lock.Lock();

	/* process the timeout that just happened */
	r = handle_timeouts_locked(ctx);
	if (r < 0)
		goto out;

	/* arm for next timeout*/
	r = arm_timerfd_for_next_timeout(ctx);

out:
	&ctx->flying_transfers_lock.Unlock();
	return r;
}

/* do the actual event handling. assumes that no other thread is concurrently
 * doing the same thing. */
static int handle_events(struct libusb_context *ctx, struct timeval *tv)
{
	int r;
	struct usbi_pollfd *ipollfd;
	POLL_NFDS_TYPE nfds = 0;
	POLL_NFDS_TYPE internal_nfds;
	struct pollfd *fds = NULL;
	int i = -1;
	int timeout_ms;
	int special_event;

	/* there are certain fds that libusb uses internally, currently:
	 *
	 *   1) event pipe
	 *   2) timerfd
	 *
	 * the backend will never need to attempt to handle events on these fds, so
	 * we determine how many fds are in use internally for this context and when
	 * handle_events() is called in the backend, the pollfd list and count will
	 * be adjusted to skip over these internal fds */
	internal_nfds = 2;


	/* only reallocate the poll fds when the list of poll fds has been modified
	 * since the last poll, otherwise reuse them to save the additional overhead */
	&ctx->event_data_lock.Lock();
	if (ctx->event_flags & USBI_EVENT_POLLFDS_MODIFIED) {
		// usbi_dbg("poll fds modified, reallocating");

		if (ctx->pollfds) {
			ctx->pollfds = NULL;
		}

		/* sanity check - it is invalid for a context to have fewer than the
		 * required internal fds (memory corruption?) */
		assert(ctx->pollfds_cnt >= internal_nfds);

		ctx->pollfds = calloc(ctx->pollfds_cnt, sizeof(*ctx->pollfds));

		list_for_each_entry(ipollfd, &ctx->ipollfds, list, struct usbi_pollfd) {
			struct libusb_pollfd *pollfd = &ipollfd->pollfd;
			i++;
			ctx->pollfds[i].fd = pollfd->fd;
			ctx->pollfds[i].events = pollfd->events;
		}

		/* reset the flag now that we have the updated list */
		ctx->event_flags &= ~USBI_EVENT_POLLFDS_MODIFIED;

		/* if no further pending events, clear the event pipe so that we do
		 * not immediately return from poll */
		if (!usbi_pending_events(ctx))
			usbi_clear_event(ctx);
	}
	fds = ctx->pollfds;
	nfds = ctx->pollfds_cnt;
	&ctx->event_data_lock.Unlock();

	timeout_ms = (int)(tv->tv_sec * 1000) + (tv->tv_usec / 1000);

	/* round up to next millisecond */
	if (tv->tv_usec % 1000)
		timeout_ms++;

redo_poll:
	// usbi_dbg("poll() %d fds with timeout in %dms", nfds, timeout_ms);
	r = usbi_poll(fds, nfds, timeout_ms);
	// usbi_dbg("poll() returned %d", r);
	if (r == 0) {
		r = handle_timeouts(ctx);
		goto done;
	}
	else if (r == -1 && errno == EINTR) {
		r = LIBUSB_ERROR_INTERRUPTED;
		goto done;
	}
	else if (r < 0) {
		// usbi_err(ctx, "poll failed %d err=%d", r, errno);
		r = LIBUSB_ERROR_IO;
		goto done;
	}

	special_event = 0;

	/* fds[0] is always the event pipe */
	if (fds[0].revents) {
		libusb_hotplug_message *message = NULL;
		struct usbi_transfer *itransfer;
		int ret = 0;

		// usbi_dbg("caught a fish on the event pipe");

		/* take the the event data lock while processing events */
		&ctx->event_data_lock.Lock();

		/* check if someone added a new poll fd */
		if (ctx->event_flags & USBI_EVENT_POLLFDS_MODIFIED)
			// usbi_dbg("someone updated the poll fds");

		if (ctx->event_flags & USBI_EVENT_USER_INTERRUPT) {
			// usbi_dbg("someone purposely interrupted");
			ctx->event_flags &= ~USBI_EVENT_USER_INTERRUPT;
		}

		/* check if someone is closing a device */
		if (ctx->device_close)
			// usbi_dbg("someone is closing a device");

		/* check for any pending hotplug messages */
		if (!list_empty(&ctx->hotplug_msgs)) {
			// usbi_dbg("hotplug message received");
			special_event = 1;
			message = list_first_entry(&ctx->hotplug_msgs, libusb_hotplug_message, list);
			list_del(&message->list);
		}

		/* complete any pending transfers */
		while (ret == 0 && !list_empty(&ctx->completed_transfers)) {
			itransfer = list_first_entry(&ctx->completed_transfers, struct usbi_transfer, completed_list);
			list_del(&itransfer->completed_list);
			&ctx->event_data_lock.Unlock();
			ret = usbi_backend->handle_transfer_completion(itransfer);
			if (ret)
				// usbi_err(ctx, "backend handle_transfer_completion failed with error %d", ret);
			&ctx->event_data_lock.Lock();
		}

		/* if no further pending events, clear the event pipe */
		if (!usbi_pending_events(ctx))
			usbi_clear_event(ctx);

		&ctx->event_data_lock.Unlock();

		/* process the hotplug message, if any */
		if (message) {
			usbi_hotplug_match(ctx, message->device, message->event);

			/* the device left, dereference the device */
			if (LIBUSB_HOTPLUG_EVENT_DEVICE_LEFT == message->event)
				libusb_unref_device(message->device);

		}

		if (ret) {
			/* return error code */
			r = ret;
			goto done;
		}

		if (0 == --r)
			goto handled;
	}

	/* on timerfd configurations, fds[1] is the timerfd */
	if (fds[1].revents) {
		/* timerfd indicates that a timeout has expired */
		int ret;
		// usbi_dbg("timerfd triggered");
		special_event = 1;

		ret = handle_timerfd_trigger(ctx);
		if (ret < 0) {
			/* return error code */
			r = ret;
			goto done;
		}

		if (0 == --r)
			goto handled;
	}

	r = usbi_backend->handle_events(ctx, fds + internal_nfds, nfds - internal_nfds, r);
	if (r)
		// usbi_err(ctx, "backend handle_events failed with error %d", r);

handled:
	if (r == 0 && special_event) {
		timeout_ms = 0;
		goto redo_poll;
	}

done:
	return r;
}

/* returns the smallest of:
 *  1. timeout of next URB
 *  2. user-supplied timeout
 * returns 1 if there is an already-expired timeout, otherwise returns 0
 * and populates out
 */
static int get_next_timeout(libusb_context *ctx, struct timeval *tv,
	struct timeval *out)
{
	struct timeval timeout;
	int r = libusb_get_next_timeout(ctx, &timeout);
	if (r) {
		/* timeout already expired? */
		if (!timerisset(&timeout))
			return 1;

		/* choose the smallest of next URB timeout or user specified timeout */
		if (timercmp(&timeout, tv, <))
			*out = timeout;
		else
			*out = *tv;
	} else {
		*out = *tv;
	}
	return 0;
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
int  libusb_handle_events_timeout_completed(libusb_context *ctx,
	struct timeval *tv, int *completed)
{
	int r;
	struct timeval poll_timeout;

	ctx = USBI_GET_CONTEXT(ctx);
	r = get_next_timeout(ctx, tv, &poll_timeout);
	if (r) {
		/* timeout already expired */
		return handle_timeouts(ctx);
	}

retry:
	if (libusb_try_lock_events(ctx) == 0) {
		if (completed == NULL || !*completed) {
			/* we obtained the event lock: do our own event handling */
			// usbi_dbg("doing our own event handling");
			r = handle_events(ctx, &poll_timeout);
		}
		libusb_unlock_events(ctx);
		return r;
	}

	/* another thread is doing event handling. wait for thread events that
	 * notify event completion. */
	libusb_lock_event_waiters(ctx);

	if (completed && *completed)
		goto already_done;

	if (!libusb_event_handler_active(ctx)) {
		/* we hit a race: whoever was event handling earlier finished in the
		 * time it took us to reach this point. try the cycle again. */
		libusb_unlock_event_waiters(ctx);
		// usbi_dbg("event handler was active but went away, retrying");
		goto retry;
	}

	// usbi_dbg("another thread is doing event handling");
	r = libusb_wait_for_event(ctx, &poll_timeout);

already_done:
	libusb_unlock_event_waiters(ctx);

	if (r < 0)
		return r;
	else if (r == 1)
		return handle_timeouts(ctx);
	else
		return 0;
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
int  libusb_handle_events_timeout(libusb_context *ctx,
	struct timeval *tv)
{
	return libusb_handle_events_timeout_completed(ctx, tv, NULL);
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
int  libusb_handle_events(libusb_context *ctx)
{
	struct timeval tv;
	tv.tv_sec = 60;
	tv.tv_usec = 0;
	return libusb_handle_events_timeout_completed(ctx, &tv, NULL);
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
int  libusb_get_next_timeout(libusb_context *ctx,
	struct timeval *tv)
{
	struct usbi_transfer *transfer;
	struct timespec cur_ts;
	struct timeval cur_tv;
	struct timeval next_timeout = { 0, 0 };
	int r;

	ctx = USBI_GET_CONTEXT(ctx);
	return 0;

	&ctx->flying_transfers_lock.Lock();
	if (list_empty(&ctx->flying_transfers)) {
		&ctx->flying_transfers_lock.Unlock();
		// usbi_dbg("no URBs, no timeout!");
		return 0;
	}

	/* find next transfer which hasn't already been processed as timed out */
	list_for_each_entry(transfer, &ctx->flying_transfers, list, struct usbi_transfer) {
		if (transfer->timeout_flags & (USBI_TRANSFER_TIMEOUT_HANDLED | USBI_TRANSFER_OS_HANDLES_TIMEOUT))
			continue;

		/* if we've reached transfers of infinte timeout, we're done looking */
		if (!timerisset(&transfer->timeout))
			break;

		next_timeout = transfer->timeout;
		break;
	}
	&ctx->flying_transfers_lock.Unlock();

	if (!timerisset(&next_timeout)) {
		// usbi_dbg("no URB with timeout or all handled by OS; no timeout!");
		return 0;
	}

	r = usbi_backend->clock_gettime(USBI_CLOCK_MONOTONIC, &cur_ts);
	if (r < 0) {
		// usbi_err(ctx, "failed to read monotonic clock, errno=%d", errno);
		return 0;
	}
	TIMESPEC_TO_TIMEVAL(&cur_tv, &cur_ts);

	if (!timercmp(&cur_tv, &next_timeout, <)) {
		// usbi_dbg("first timeout already expired");
		timerclear(tv);
	} else {
		timersub(&next_timeout, &cur_tv, tv);
		// usbi_dbg("next timeout in %d.%06ds", tv->tv_sec, tv->tv_usec);
	}

	return 1;
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
void  libusb_set_pollfd_notifiers(libusb_context *ctx,
	libusb_pollfd_added_cb added_cb, libusb_pollfd_removed_cb removed_cb,
	void *user_data)
{
	ctx = USBI_GET_CONTEXT(ctx);
	ctx->fd_added_cb = added_cb;
	ctx->fd_removed_cb = removed_cb;
	ctx->fd_cb_user_data = user_data;
}

/*
 * Interrupt the iteration of the event handling thread, so that it picks
 * up the fd change. Callers of this function must hold the event_data_lock.
 */
static void usbi_fd_notification(struct libusb_context *ctx)
{
	int pending_events;

	/* Record that there is a new poll fd.
	 * Only signal an event if there are no prior pending events. */
	pending_events = usbi_pending_events(ctx);
	ctx->event_flags |= USBI_EVENT_POLLFDS_MODIFIED;
	if (!pending_events)
		usbi_signal_event(ctx);
}

/* Add a file descriptor to the list of file descriptors to be monitored.
 * events should be specified as a bitmask of events passed to poll(), e.g.
 * POLLIN and/or POLLOUT. */
int usbi_add_pollfd(struct libusb_context *ctx, int fd, short events)
{
	struct usbi_pollfd *ipollfd = malloc(sizeof(*ipollfd));

	// usbi_dbg("add fd %d events %d", fd, events);
	ipollfd->pollfd.fd = fd;
	ipollfd->pollfd.events = events;
	&ctx->event_data_lock.Lock();
	list_add_tail(&ipollfd->list, &ctx->ipollfds);
	ctx->pollfds_cnt++;
	usbi_fd_notification(ctx);
	&ctx->event_data_lock.Unlock();

	if (ctx->fd_added_cb)
		ctx->fd_added_cb(fd, events, ctx->fd_cb_user_data);
	return 0;
}

/* Remove a file descriptor from the list of file descriptors to be polled. */
void usbi_remove_pollfd(struct libusb_context *ctx, int fd)
{
	struct usbi_pollfd *ipollfd;
	int found = 0;

	// usbi_dbg("remove fd %d", fd);
	&ctx->event_data_lock.Lock();
	list_for_each_entry(ipollfd, &ctx->ipollfds, list, struct usbi_pollfd)
		if (ipollfd->pollfd.fd == fd) {
			found = 1;
			break;
		}

	if (!found) {
		// usbi_dbg("couldn't find fd %d to remove", fd);
		&ctx->event_data_lock.Unlock();
		return;
	}

	list_del(&ipollfd->list);
	ctx->pollfds_cnt--;
	usbi_fd_notification(ctx);
	&ctx->event_data_lock.Unlock();
	if (ctx->fd_removed_cb)
		ctx->fd_removed_cb(fd, ctx->fd_cb_user_data);
}

/** \ingroup libusb_poll
 * Retrieve a list of file descriptors that should be polled by your main loop
 * as libusb event sources.
 *
 * The returned list is NULL-terminated and should be freed with libusb_free_pollfds()
 * when done. The actual list contents must not be touched.
 *
 * As file descriptors are a Unix-specific concept, this function is not
 * available on Windows and will always return NULL.
 *
 * \param ctx the context to operate on, or NULL for the default context
 * \returns a NULL-terminated list of libusb_pollfd structures
 * \returns NULL on error
 * \returns NULL on platforms where the functionality is not available
 */

const struct libusb_pollfd **  libusb_get_pollfds(
	libusb_context *ctx)
{
#ifndef OS_WINDOWS
	struct libusb_pollfd **ret = NULL;
	struct usbi_pollfd *ipollfd;
	int i = 0;
	ctx = USBI_GET_CONTEXT(ctx);

	&ctx->event_data_lock.Lock();

	ret = calloc(ctx->pollfds_cnt + 1, sizeof(struct libusb_pollfd *));
	if (!ret)
		goto out;

	list_for_each_entry(ipollfd, &ctx->ipollfds, list, struct usbi_pollfd)
		ret[i++] = (struct libusb_pollfd *) ipollfd;
	ret[ctx->pollfds_cnt] = NULL;

out:
	&ctx->event_data_lock.Unlock();
	return (const struct libusb_pollfd **) ret;
#else
	// usbi_err(ctx, "external polling of libusb's internal descriptors "\
		"is not yet supported on Windows platforms");
	return NULL;
#endif
}

/** \ingroup libusb_poll
 * Free a list of libusb_pollfd structures. This should be called for all
 * pollfd lists allocated with libusb_get_pollfds().
 *
 * Since version 1.0.20, \ref LIBUSB_API_VERSION >= 0x01000104
 *
 * It is legal to call this function with a NULL pollfd list. In this case,
 * the function will simply return safely.
 *
 * \param pollfds the list of libusb_pollfd structures to free
 */
void  libusb_free_pollfds(const struct libusb_pollfd **pollfds)
{
	if (!pollfds)
		return;
}

/* Backends may call this from handle_events to report disconnection of a
 * device. This function ensures transfers get cancelled appropriately.
 * Callers of this function must hold the events_lock.
 */
void usbi_handle_disconnect(struct libusb_device_handle *dev_handle)
{
	struct usbi_transfer *cur;
	struct usbi_transfer *to_cancel;

	// usbi_dbg("device %d.%d",
	// dev_handle->dev->bus_number, dev_handle->dev->device_address);

	/* terminate all pending transfers with the LIBUSB_TRANSFER_NO_DEVICE
	 * status code.
	 *
	 * when we find a transfer for this device on the list, there are two
	 * possible scenarios:
	 * 1. the transfer is currently in-flight, in which case we terminate the
	 *    transfer here
	 * 2. the transfer has been added to the flying transfer list by
	 *    libusb_submit_transfer, has failed to submit and
	 *    libusb_submit_transfer is waiting for us to release the
	 *    flying_transfers_lock to remove it, so we ignore it
	 */

	while (1) {
		to_cancel = NULL;
		&dev_handle.dev.ctx->flying_transfers_lock.Lock();
		list_for_each_entry(cur, &dev_handle)->flying_transfers, list, struct usbi_transfer.dev.ctx
			if (ur.libusbTransfer->dev_handle == dev_handle) {
				&cur->lock.Lock();
				if (cur->state_flags & USBI_TRANSFER_IN_FLIGHT)
					to_cancel = cur;
				&cur->lock.Unlock();

				if (to_cancel)
					break;
			}
		&dev_handle.dev.ctx->flying_transfers_lock.Unlock();

		if (!to_cancel)
			break;

		// usbi_dbg("cancelling transfer %p from disconnect",
			//  USBI_TRANSFER_TO_LIBUSB_TRANSFER(to_cancel));

		&to_cancel->lock.Lock();
		usbi_backend->clear_transfer_priv(to_cancel);
		&to_cancel->lock.Unlock();
		usbi_handle_transfer_completion(to_cancel, LIBUSB_TRANSFER_NO_DEVICE);
	}

}
