// +build windows

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
	// usbi_err(ctx, "external polling of libusb's internal descriptors "\
	//	"is not yet supported on Windows platforms");
	return nil
}
