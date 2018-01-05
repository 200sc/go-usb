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
 
	 ctx.timerfd = timerfd_create(usbi_backend->get_timerfd_clockid(), TFD_NONBLOCK)
	 if (ctx.timerfd >= 0) {
		 // usbi_dbg("using timerfd for timeouts")
		 r = usbi_add_pollfd(ctx, ctx.timerfd, POLLIN)
		 if (r < 0) {
			close(ctx->timerfd)
			usbi_remove_pollfd(ctx, ctx.event_pipe[0])
			usbi_close(ctx.event_pipe[0])
	 		usbi_close(ctx.event_pipe[1])
	 		return r
		 }
	 }
 
	 return 0
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