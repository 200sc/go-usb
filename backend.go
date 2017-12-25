package usb

/* This is the interface that OS backends need to implement.
 * All fields are mandatory, except ones explicitly noted as optional. */
type usbi_os_backend interface {
	/* A human-readable name for your backend, e.g. "Linux usbfs" */
	Name() string

	/* Binary mask for backend specific capabilities */
	Caps() uint32

	/* Perform initialization of your backend. You might use this function
	 * to determine specific capabilities of the system, allocate required
	 * data structures for later, etc.
	 *
	 * This function is called when a libusb user initializes the library
	 * prior to use.
	 *
	 * Return 0 on success, or a LIBUSB_ERROR code on failure.
	 */
	Init(*libusb_context) libusb_error

	/* Deinitialization. Optional. This function should destroy anything
	 * that was set up by init.
	 *
	 * This function is called when the user deinitializes the library.
	 */
	Exit()

	/* Enumerate all the USB devices on the system, returning them in a list
	 * of discovered devices.
	 *
	 * Your implementation should enumerate all devices on the system,
	 * regardless of whether they have been seen before or not.
	 *
	 * When you have found a device, compute a session ID for it. The session
	 * ID should uniquely represent that particular device for that particular
	 * connection session since boot (i.e. if you disconnect and reconnect a
	 * device immediately after, it should be assigned a different session ID).
	 * If your OS cannot provide a unique session ID as described above,
	 * presenting a session ID of (bus_number << 8 | device_address) should
	 * be sufficient. Bus numbers and device addresses wrap and get reused,
	 * but that is an unlikely case.
	 *
	 * After computing a session ID for a device, call
	 * usbi_get_device_by_session_id(). This function checks if libusb already
	 * knows about the device, and if so, it provides you with a reference
	 * to a libusb_device structure for it.
	 *
	 * If usbi_get_device_by_session_id() returns NULL, it is time to allocate
	 * a new device structure for the device. Call usbi_alloc_device() to
	 * obtain a new libusb_device structure with reference count 1. Populate
	 * the bus_number and device_address attributes of the new device, and
	 * perform any other internal backend initialization you need to do. At
	 * this point, you should be ready to provide device descriptors and so
	 * on through the get_*_descriptor functions. Finally, call
	 * usbi_sanitize_device() to perform some final sanity checks on the
	 * device. Assuming all of the above succeeded, we can now continue.
	 * If any of the above failed, remember to unreference the device that
	 * was returned by usbi_alloc_device().
	 *
	 * At this stage we have a populated libusb_device structure (either one
	 * that was found earlier, or one that we have just allocated and
	 * populated). This can now be added to the discovered devices list
	 * using discovered_devs_append(). Note that discovered_devs_append()
	 * may reallocate the list, returning a new location for it, and also
	 * note that reallocation can fail. Your backend should handle these
	 * error conditions appropriately.
	 *
	 * This function should not generate any bus I/O and should not block.
	 * If I/O is required (e.g. reading the active configuration value), it is
	 * OK to ignore these suggestions :)
	 *
	 * This function is executed when the user wishes to retrieve a list
	 * of USB devices connected to the system.
	 *
	 * If the backend has hotplug support, this function is not used!
	 *
	 * Return 0 on success, or a LIBUSB_ERROR code on failure.
	 */
	Get_device_list(*libusb_context, **discovered_devs) libusb_error

	/* Apps which were written before hotplug support, may listen for
	 * hotplug events on their own and call libusb_get_device_list on
	 * device addition. In this case libusb_get_device_list will likely
	 * return a list without the new device in there, as the hotplug
	 * event thread will still be busy enumerating the device, which may
	 * take a while, or may not even have seen the event yet.
	 *
	 * To avoid this libusb_get_device_list will call this optional
	 * function for backends with hotplug support before copying
	 * ctx->usb_devs to the user. In this function the backend should
	 * ensure any pending hotplug events are fully processed before
	 * returning.
	 *
	 * Optional, should be implemented by backends with hotplug support.
	 */
	Hotplug_poll()

	/* Open a device for I/O and other USB operations. The device handle
	 * is preallocated for you, you can retrieve the device in question
	 * through handle->dev.
	 *
	 * Your backend should allocate any internal resources required for I/O
	 * and other operations so that those operations can happen (hopefully)
	 * without hiccup. This is also a good place to inform libusb that it
	 * should monitor certain file descriptors related to this device -
	 * see the usbi_add_pollfd() function.
	 *
	 * This function should not generate any bus I/O and should not block.
	 *
	 * This function is called when the user attempts to obtain a device
	 * handle for a device.
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_ACCESS if the user has insufficient permissions
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected since
	 *   discovery
	 * - another LIBUSB_ERROR code on other failure
	 *
	 * Do not worry about freeing the handle on failed open, the upper layers
	 * do this for you.
	 */
	Open(*libusb_device_handle) libusb_error

	/* Close a device such that the handle cannot be used again. Your backend
	 * should destroy any resources that were allocated in the open path.
	 * This may also be a good place to call usbi_remove_pollfd() to inform
	 * libusb of any file descriptors associated with this device that should
	 * no longer be monitored.
	 *
	 * This function is called when the user closes a device handle.
	 */
	Close(*libusb_device_handle)

	/* Retrieve the device descriptor from a device.
	 *
	 * The descriptor should be retrieved from memory, NOT via bus I/O to the
	 * device. This means that you may have to cache it in a private structure
	 * during get_device_list enumeration. Alternatively, you may be able
	 * to retrieve it from a kernel interface (some Linux setups can do this)
	 * still without generating bus I/O.
	 *
	 * This function is expected to write DEVICE_DESC_LENGTH (18) bytes into
	 * buffer, which is guaranteed to be big enough.
	 *
	 * This function is called when sanity-checking a device before adding
	 * it to the list of discovered devices, and also when the user requests
	 * to read the device descriptor.
	 *
	 * This function is expected to return the descriptor in bus-endian format
	 * (LE). If it returns the multi-byte values in host-endian format,
	 * set the host_endian output parameter to "1".
	 *
	 * Return 0 on success or a LIBUSB_ERROR code on failure.
	 */
	Get_device_descriptor(*libusb_device, []uint8, *int) libusb_error

	/* Get the ACTIVE configuration descriptor for a device.
	 *
	 * The descriptor should be retrieved from memory, NOT via bus I/O to the
	 * device. This means that you may have to cache it in a private structure
	 * during get_device_list enumeration. You may also have to keep track
	 * of which configuration is active when the user changes it.
	 *
	 * This function is expected to write len bytes of data into buffer, which
	 * is guaranteed to be big enough. If you can only do a partial write,
	 * return an error code.
	 *
	 * This function is expected to return the descriptor in bus-endian format
	 * (LE). If it returns the multi-byte values in host-endian format,
	 * set the host_endian output parameter to "1".
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NOT_FOUND if the device is in unconfigured state
	 * - another LIBUSB_ERROR code on other failure
	 */
	Get_active_config_descriptor(*libusb_device, []uint8, *int) libusb_error

	/* Get a specific configuration descriptor for a device.
	 *
	 * The descriptor should be retrieved from memory, NOT via bus I/O to the
	 * device. This means that you may have to cache it in a private structure
	 * during get_device_list enumeration.
	 *
	 * The requested descriptor is expressed as a zero-based index (i.e. 0
	 * indicates that we are requesting the first descriptor). The index does
	 * not (necessarily) equal the bConfigurationValue of the configuration
	 * being requested.
	 *
	 * This function is expected to write len bytes of data into buffer, which
	 * is guaranteed to be big enough. If you can only do a partial write,
	 * return an error code.
	 *
	 * This function is expected to return the descriptor in bus-endian format
	 * (LE). If it returns the multi-byte values in host-endian format,
	 * set the host_endian output parameter to "1".
	 *
	 * Return the length read on success or a LIBUSB_ERROR code on failure.
	 */
	Get_config_descriptor(*libusb_device, uint8, []uint8, int, *int) libusb_error

	/* Like get_config_descriptor but then by bConfigurationValue instead
	 * of by index.
	 *
	 * Optional, if not present the core will call get_config_descriptor
	 * for all configs until it finds the desired bConfigurationValue.
	 *
	 * Returns a pointer to the raw-descriptor in *buffer, this memory
	 * is valid as long as device is valid.
	 *
	 * Returns the length of the returned raw-descriptor on success,
	 * or a LIBUSB_ERROR code on failure.
	 */
	Get_config_descriptor_by_value(*libusb_device, uint8, *[]uint8, *int) libusb_error

	/* Get the bConfigurationValue for the active configuration for a device.
	 * Optional. This should only be implemented if you can retrieve it from
	 * cache (don't generate I/O).
	 *
	 * If you cannot retrieve this from cache, either do not implement this
	 * function, or return LIBUSB_ERROR_NOT_SUPPORTED. This will cause
	 * libusb to retrieve the information through a standard control transfer.
	 *
	 * This function must be non-blocking.
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected since it
	 *   was opened
	 * - LIBUSB_ERROR_NOT_SUPPORTED if the value cannot be retrieved without
	 *   blocking
	 * - another LIBUSB_ERROR code on other failure.
	 */
	Get_configuration(*libusb_device_handle, *int) libusb_error

	/* Set the active configuration for a device.
	 *
	 * A configuration value of -1 should put the device in unconfigured state.
	 *
	 * This function can block.
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NOT_FOUND if the configuration does not exist
	 * - LIBUSB_ERROR_BUSY if interfaces are currently claimed (and hence
	 *   configuration cannot be changed)
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected since it
	 *   was opened
	 * - another LIBUSB_ERROR code on other failure.
	 */
	Set_configuration(*libusb_device_handle, *int) libusb_error

	/* Claim an interface. When claimed, the application can then perform
	 * I/O to an interface's endpoints.
	 *
	 * This function should not generate any bus I/O and should not block.
	 * Interface claiming is a logical operation that simply ensures that
	 * no other drivers/applications are using the interface, and after
	 * claiming, no other drivers/applications can use the interface because
	 * we now "own" it.
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NOT_FOUND if the interface does not exist
	 * - LIBUSB_ERROR_BUSY if the interface is in use by another driver/app
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected since it
	 *   was opened
	 * - another LIBUSB_ERROR code on other failure
	 */
	Claim_interface(*libusb_device_handle, int) libusb_error

	/* Release a previously claimed interface.
	 *
	 * This function should also generate a SET_INTERFACE control request,
	 * resetting the alternate setting of that interface to 0. It's OK for
	 * this function to block as a result.
	 *
	 * You will only ever be asked to release an interface which was
	 * successfully claimed earlier.
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected since it
	 *   was opened
	 * - another LIBUSB_ERROR code on other failure
	 */
	Release_interface(*libusb_device_handle, int) libusb_error

	/* Set the alternate setting for an interface.
	 *
	 * You will only ever be asked to set the alternate setting for an
	 * interface which was successfully claimed earlier.
	 *
	 * It's OK for this function to block.
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NOT_FOUND if the alternate setting does not exist
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected since it
	 *   was opened
	 * - another LIBUSB_ERROR code on other failure
	 */
	Set_interface_altsetting(*libusb_device_handle, int, int) libusb_error

	/* Clear a halt/stall condition on an endpoint.
	 *
	 * It's OK for this function to block.
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NOT_FOUND if the endpoint does not exist
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected since it
	 *   was opened
	 * - another LIBUSB_ERROR code on other failure
	 */
	Clear_halt(*libusb_device_handle, uint8) libusb_error

	/* Perform a USB port reset to reinitialize a device.
	 *
	 * If possible, the device handle should still be usable after the reset
	 * completes, assuming that the device descriptors did not change during
	 * reset and all previous interface state can be restored.
	 *
	 * If something changes, or you cannot easily locate/verify the resetted
	 * device, return LIBUSB_ERROR_NOT_FOUND. This prompts the application
	 * to close the old handle and re-enumerate the device.
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NOT_FOUND if re-enumeration is required, or if the device
	 *   has been disconnected since it was opened
	 * - another LIBUSB_ERROR code on other failure
	 */
	Reset_device(*libusb_device_handle) libusb_error

	/* Alloc num_streams usb3 bulk streams on the passed in endpoints */
	Alloc_streams(*libusb_device_handle, uint32, *uint8, int) libusb_error

	/* Free usb3 bulk streams allocated with alloc_streams */
	Free_streams(*libusb_device_handle, *uint8, int) libusb_error

	/* Allocate persistent DMA memory for the given device, suitable for
	 * zerocopy. May return NULL on failure. Optional to implement.
	 */
	Dev_mem_alloc(*libusb_device_handle, int) []uint8

	/* Free memory allocated by dev_mem_alloc. */
	Dev_mem_free(*libusb_device_handle, []uint8, int) libusb_error

	/* Determine if a kernel driver is active on an interface. Optional.
	 *
	 * The presence of a kernel driver on an interface indicates that any
	 * calls to claim_interface would fail with the LIBUSB_ERROR_BUSY code.
	 *
	 * Return:
	 * - 0 if no driver is active
	 * - 1 if a driver is active
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected since it
	 *   was opened
	 * - another LIBUSB_ERROR code on other failure
	 */
	Kernel_driver_active(*libusb_device_handle, int) libusb_error

	/* Detach a kernel driver from an interface. Optional.
	 *
	 * After detaching a kernel driver, the interface should be available
	 * for claim.
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NOT_FOUND if no kernel driver was active
	 * - LIBUSB_ERROR_INVALID_PARAM if the interface does not exist
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected since it
	 *   was opened
	 * - another LIBUSB_ERROR code on other failure
	 */
	Detatch_kernel_driver(*libusb_device_handle, int) libusb_error

	/* Attach a kernel driver to an interface. Optional.
	 *
	 * Reattach a kernel driver to the device.
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NOT_FOUND if no kernel driver was active
	 * - LIBUSB_ERROR_INVALID_PARAM if the interface does not exist
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected since it
	 *   was opened
	 * - LIBUSB_ERROR_BUSY if a program or driver has claimed the interface,
	 *   preventing reattachment
	 * - another LIBUSB_ERROR code on other failure
	 */
	Attach_kernel_driver(*libusb_device_handle, int) libusb_error

	/* Submit a transfer. Your implementation should take the transfer,
	 * morph it into whatever form your platform requires, and submit it
	 * asynchronously.
	 *
	 * This function must not block.
	 *
	 * This function gets called with the flying_transfers_lock locked!
	 *
	 * Return:
	 * - 0 on success
	 * - LIBUSB_ERROR_NO_DEVICE if the device has been disconnected
	 * - another LIBUSB_ERROR code on other failure
	 */
	Submit_transfer(*usbi_transfer) libusb_error

	/* Cancel a previously submitted transfer.
	 *
	 * This function must not block. The transfer cancellation must complete
	 * later, resulting in a call to usbi_handle_transfer_cancellation()
	 * from the context of handle_events.
	 */
	Cancel_transfer(*usbi_transfer) libusb_error

	/* Clear a transfer as if it has completed or cancelled, but do not
	 * report any completion/cancellation to the library. You should free
	 * all private data from the transfer as if you were just about to report
	 * completion or cancellation.
	 *
	 * This function might seem a bit out of place. It is used when libusb
	 * detects a disconnected device - it calls this function for all pending
	 * transfers before reporting completion (with the disconnect code) to
	 * the user. Maybe we can improve upon this internal interface in future.
	 */
	Clear_transfer_priv(*usbi_transfer)

	/* Handle any pending events on file descriptors. Optional.
	 *
	 * Provide this function when file descriptors directly indicate device
	 * or transfer activity. If your backend does not have such file descriptors,
	 * implement the handle_transfer_completion function below.
	 *
	 * This involves monitoring any active transfers and processing their
	 * completion or cancellation.
	 *
	 * The function is passed an array of pollfd structures (size nfds)
	 * as a result of the poll() system call. The num_ready parameter
	 * indicates the number of file descriptors that have reported events
	 * (i.e. the poll() return value). This should be enough information
	 * for you to determine which actions need to be taken on the currently
	 * active transfers.
	 *
	 * For any cancelled transfers, call usbi_handle_transfer_cancellation().
	 * For completed transfers, call usbi_handle_transfer_completion().
	 * For control/bulk/interrupt transfers, populate the "transferred"
	 * element of the appropriate usbi_transfer structure before calling the
	 * above functions. For isochronous transfers, populate the status and
	 * transferred fields of the iso packet descriptors of the transfer.
	 *
	 * This function should also be able to detect disconnection of the
	 * device, reporting that situation with usbi_handle_disconnect().
	 *
	 * When processing an event related to a transfer, you probably want to
	 * take usbi_transfer.lock to prevent races. See the documentation for
	 * the usbi_transfer structure.
	 *
	 * Return 0 on success, or a LIBUSB_ERROR code on failure.
	 */
	Handle_events(*libusb_context, *pollfd, POLL_NFDS_TYPE, int) libusb_error

	/* Handle transfer completion. Optional.
	 *
	 * Provide this function when there are no file descriptors available
	 * that directly indicate device or transfer activity. If your backend does
	 * have such file descriptors, implement the handle_events function above.
	 *
	 * Your backend must tell the library when a transfer has completed by
	 * calling usbi_signal_transfer_completion(). You should store any private
	 * information about the transfer and its completion status in the transfer's
	 * private backend data.
	 *
	 * During event handling, this function will be called on each transfer for
	 * which usbi_signal_transfer_completion() was called.
	 *
	 * For any cancelled transfers, call usbi_handle_transfer_cancellation().
	 * For completed transfers, call usbi_handle_transfer_completion().
	 * For control/bulk/interrupt transfers, populate the "transferred"
	 * element of the appropriate usbi_transfer structure before calling the
	 * above functions. For isochronous transfers, populate the status and
	 * transferred fields of the iso packet descriptors of the transfer.
	 *
	 * Return 0 on success, or a LIBUSB_ERROR code on failure.
	 */
	Handle_transfer_completion(*usbi_transfer) libusb_error

	/* Number of bytes to reserve for per-device private backend data.
	 * This private data area is accessible through the "os_priv" field of
	 * struct libusb_device. */
	Device_priv_size() int

	/* Number of bytes to reserve for per-handle private backend data.
	 * This private data area is accessible through the "os_priv" field of
	 * struct libusb_device. */
	Device_handle_priv_size() int

	/* Number of bytes to reserve for per-transfer private backend data.
	 * This private data area is accessible by calling
	 * usbi_transfer_get_os_priv() on the appropriate usbi_transfer instance.
	 */
	Transfer_priv_size() int

	Get_timerfd_clockid() clockid_t
}
