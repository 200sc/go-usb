// +build !windows

package usb

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

func libusb_get_pollfds(ctx *libusb_context) **libusb_pollfd {
	ret := make([]*libusb_pollfd, ctx.pollfds_cnt+1)
 
	i := 0
	ctx = USBI_GET_CONTEXT(ctx)

	ctx.event_data_lock.Lock()
		
	for ipollfd := list_entry((ctx.ipollfds).next, usbi_pollfd, list);
		&ipollfd.list != (ctx.ipollfds);
		ipollfd = list_entry(ipollfd.list.next, usbi_pollfd, list)) {

		ret[i++] = ipollfd
	}
	
	ret[ctx.pollfds_cnt] = nil

	ctx.event_data_lock.Unlock()
	return ret
}