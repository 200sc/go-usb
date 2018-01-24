package usb

/*
 * Synchronous I/O functions for libusb
 * Copyright © 2007-2008 Daniel Drake <dsd@gentoo.org>
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

/**
 * @defgroup libusb_syncio Synchronous device I/O
 *
 * This page documents libusb's synchronous (blocking) API for USB device I/O.
 * This interface is easy to use but has some limitations. More advanced users
 * may wish to consider using the \ref libusb_asyncio "asynchronous I/O API" instead.
 */

func sync_transfer_cb(transfer *libusb_transfer) {
	*transfer.user_data = 1
}

func sync_transfer_wait_for_completion(transfer *libusb_transfer) {
	var completed *int = transfer.user_data //user_data is probably an interface{}?

	ctx := transfer.dev_handle.dev.ctx

	for *completed == 0 {
		r := libusb_handle_events_completed(ctx, completed)
		if r < 0 {
			if r == LIBUSB_ERROR_INTERRUPTED {
				continue
			}
			// usbi_err(ctx, "libusb_handle_events failed: %s, cancelling transfer and retrying",
			//  libusb_error_name(r));
			libusb_cancel_transfer(transfer)
			continue
		}
	}
}

/** \ingroup libusb_syncio
 * Perform a USB control transfer.
 *
 * The direction of the transfer is inferred from the bmRequestType field of
 * the setup packet.
 *
 * The wValue, wIndex and wLength fields values should be given in host-endian
 * byte order.
 *
 * \param dev_handle a handle for the device to communicate with
 * \param bmRequestType the request type field for the setup packet
 * \param bRequest the request field for the setup packet
 * \param wValue the value field for the setup packet
 * \param wIndex the index field for the setup packet
 * \param data a suitably-sized data buffer for either input or output
 * (depending on direction bits within bmRequestType)
 * \param wLength the length field for the setup packet. The data buffer should
 * be at least this size.
 * \param timeout timeout (in millseconds) that this function should wait
 * before giving up due to no response being received. For an unlimited
 * timeout, use value 0.
 * \returns on success, the number of bytes actually transferred
 * \returns LIBUSB_ERROR_TIMEOUT if the transfer timed out
 * \returns LIBUSB_ERROR_PIPE if the control request was not supported by the
 * device
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns LIBUSB_ERROR_BUSY if called from event handling context
 * \returns LIBUSB_ERROR_INVALID_PARAM if the transfer size is larger than
 * the operating system and/or hardware can support
 * \returns another LIBUSB_ERROR code on other failures
 */
func ibusb_control_transfer(dev_handle *libusb_device_handle,
	bmRequestType uint8, bRequest uint8, wValue uint16, wIndex uint16,
	data []uint8, wLength uint16, timeout uint) int {

	var completed int

	transfer := libusb_alloc_transfer(0)
	buffer := make([]uint8, LIBUSB_CONTROL_SETUP_SIZE)

	libusb_fill_control_setup(buffer, bmRequestType, bRequest, wValue, wIndex, wLength)

	if (bmRequestType & LIBUSB_ENDPOINT_DIR_MASK) == LIBUSB_ENDPOINT_OUT {
		buffer = append(buffer, data[0:wLength])
	} else {
		buffer = make([]uint8, LIBUSB_CONTROL_SETUP_SIZE+wLength)
	}

	libusb_fill_control_transfer(transfer, dev_handle, buffer,
		sync_transfer_cb, &completed, timeout)

	transfer.flags = LIBUSB_TRANSFER_FREE_BUFFER

	r := libusb_submit_transfer(transfer)
	if r < 0 {
		return r
	}

	sync_transfer_wait_for_completion(transfer)

	if (bmRequestType & LIBUSB_ENDPOINT_DIR_MASK) == LIBUSB_ENDPOINT_IN {
		tdata := libusb_control_transfer_get_data(transfer)
		for i := 0; i < transfer.actual_length; i++ {
			data[i] = tdata[i]
		}
	}

	switch transfer.status {
	case LIBUSB_TRANSFER_COMPLETED:
		r = transfer.actual_length
	case LIBUSB_TRANSFER_TIMED_OUT:
		r = LIBUSB_ERROR_TIMEOUT
	case LIBUSB_TRANSFER_STALL:
		r = LIBUSB_ERROR_PIPE
	case LIBUSB_TRANSFER_NO_DEVICE:
		r = LIBUSB_ERROR_NO_DEVICE
	case LIBUSB_TRANSFER_OVERFLOW:
		r = LIBUSB_ERROR_OVERFLOW
	case LIBUSB_TRANSFER_ERROR:
		fallthrough
	case LIBUSB_TRANSFER_CANCELLED:
		r = LIBUSB_ERROR_IO
	default:
		// usbi_warn(dev_handle.dev.ctx,
		// "unrecognised status code %d", transfer.status);
		r = LIBUSB_ERROR_OTHER
	}

	return r
}

func do_sync_bulk_transfer(dev_handle *libusb_device_handle,
	endpoint uint8, buffer []uint8, length int,
	transferred *int, timeout uint, _type uint8) int {

	completed := 0

	transfer := libusb_alloc_transfer(0)

	libusb_fill_bulk_transfer(transfer, dev_handle, endpoint, buffer, length,
		sync_transfer_cb, &completed, timeout)
	transfer._type = _type

	r := libusb_submit_transfer(transfer)
	if r < 0 {
		return r
	}

	sync_transfer_wait_for_completion(transfer)

	if transferred != nil {
		*transferred = transfer.actual_length
	}

	switch transfer.status {
	case LIBUSB_TRANSFER_COMPLETED:
		r = 0
	case LIBUSB_TRANSFER_TIMED_OUT:
		r = LIBUSB_ERROR_TIMEOUT
	case LIBUSB_TRANSFER_STALL:
		r = LIBUSB_ERROR_PIPE
	case LIBUSB_TRANSFER_OVERFLOW:
		r = LIBUSB_ERROR_OVERFLOW
	case LIBUSB_TRANSFER_NO_DEVICE:
		r = LIBUSB_ERROR_NO_DEVICE
	case LIBUSB_TRANSFER_ERROR:
		fallthrough
	case LIBUSB_TRANSFER_CANCELLED:
		r = LIBUSB_ERROR_IO
	default:
		// usbi_warn(dev_handle.dev.ctx,
		// "unrecognised status code %d", transfer.status);
		r = LIBUSB_ERROR_OTHER
	}

	return r
}

/** \ingroup libusb_syncio
 * Perform a USB bulk transfer. The direction of the transfer is inferred from
 * the direction bits of the endpoint address.
 *
 * For bulk reads, the <tt>length</tt> field indicates the maximum length of
 * data you are expecting to receive. If less data arrives than expected,
 * this function will return that data, so be sure to check the
 * <tt>transferred</tt> output parameter.
 *
 * You should also check the <tt>transferred</tt> parameter for bulk writes.
 * Not all of the data may have been written.
 *
 * Also check <tt>transferred</tt> when dealing with a timeout error code.
 * libusb may have to split your transfer into a number of chunks to satisfy
 * underlying O/S requirements, meaning that the timeout may expire after
 * the first few chunks have completed. libusb is careful not to lose any data
 * that may have been transferred; do not assume that timeout conditions
 * indicate a complete lack of I/O.
 *
 * \param dev_handle a handle for the device to communicate with
 * \param endpoint the address of a valid endpoint to communicate with
 * \param data a suitably-sized data buffer for either input or output
 * (depending on endpoint)
 * \param length for bulk writes, the number of bytes from data to be sent. for
 * bulk reads, the maximum number of bytes to receive into the data buffer.
 * \param transferred output location for the number of bytes actually
 * transferred. Since version 1.0.21 (\ref LIBUSB_API_VERSION >= 0x01000105),
 * it is legal to pass a NULL pointer if you do not wish to receive this
 * information.
 * \param timeout timeout (in millseconds) that this function should wait
 * before giving up due to no response being received. For an unlimited
 * timeout, use value 0.
 *
 * \returns 0 on success (and populates <tt>transferred</tt>)
 * \returns LIBUSB_ERROR_TIMEOUT if the transfer timed out (and populates
 * <tt>transferred</tt>)
 * \returns LIBUSB_ERROR_PIPE if the endpoint halted
 * \returns LIBUSB_ERROR_OVERFLOW if the device offered more data, see
 * \ref libusb_packetoverflow
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns LIBUSB_ERROR_BUSY if called from event handling context
 * \returns another LIBUSB_ERROR code on other failures
 */
func libusb_bulk_transfer(dev_handle *libusb_device_handle, endpoint uint8,
	data *uint8, length int, transferred *int, timeout uint) int {
	return do_sync_bulk_transfer(dev_handle, endpoint, data, length,
		transferred, timeout, LIBUSB_TRANSFER_TYPE_BULK)
}

/** \ingroup libusb_syncio
 * Perform a USB interrupt transfer. The direction of the transfer is inferred
 * from the direction bits of the endpoint address.
 *
 * For interrupt reads, the <tt>length</tt> field indicates the maximum length
 * of data you are expecting to receive. If less data arrives than expected,
 * this function will return that data, so be sure to check the
 * <tt>transferred</tt> output parameter.
 *
 * You should also check the <tt>transferred</tt> parameter for interrupt
 * writes. Not all of the data may have been written.
 *
 * Also check <tt>transferred</tt> when dealing with a timeout error code.
 * libusb may have to split your transfer into a number of chunks to satisfy
 * underlying O/S requirements, meaning that the timeout may expire after
 * the first few chunks have completed. libusb is careful not to lose any data
 * that may have been transferred; do not assume that timeout conditions
 * indicate a complete lack of I/O.
 *
 * The default endpoint bInterval value is used as the polling interval.
 *
 * \param dev_handle a handle for the device to communicate with
 * \param endpoint the address of a valid endpoint to communicate with
 * \param data a suitably-sized data buffer for either input or output
 * (depending on endpoint)
 * \param length for bulk writes, the number of bytes from data to be sent. for
 * bulk reads, the maximum number of bytes to receive into the data buffer.
 * \param transferred output location for the number of bytes actually
 * transferred. Since version 1.0.21 (\ref LIBUSB_API_VERSION >= 0x01000105),
 * it is legal to pass a NULL pointer if you do not wish to receive this
 * information.
 * \param timeout timeout (in millseconds) that this function should wait
 * before giving up due to no response being received. For an unlimited
 * timeout, use value 0.
 *
 * \returns 0 on success (and populates <tt>transferred</tt>)
 * \returns LIBUSB_ERROR_TIMEOUT if the transfer timed out
 * \returns LIBUSB_ERROR_PIPE if the endpoint halted
 * \returns LIBUSB_ERROR_OVERFLOW if the device offered more data, see
 * \ref libusb_packetoverflow
 * \returns LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
 * \returns LIBUSB_ERROR_BUSY if called from event handling context
 * \returns another LIBUSB_ERROR code on other error
 */
func libusb_interrupt_transfer(dev_handle *libusb_device_handle, endpoint uint8,
	data *uint8, length int, transferred *int, timeout uint32) int {
	return do_sync_bulk_transfer(dev_handle, endpoint, data, length,
		transferred, timeout, LIBUSB_TRANSFER_TYPE_INTERRUPT)
}
