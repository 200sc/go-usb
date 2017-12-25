package go_libusb

/*
 * Hotplug support for libusb
 * Copyright © 2012-2013 Nathan Hjelm <hjelmn@mac.com>
 * Copyright © 2012-2013 Peter Stuge <peter@stuge.se>
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
  * The hotplug callback structure. The user populates this structure with
  * libusb_hotplug_prepare_callback() and then calls libusb_hotplug_register_callback()
  * to receive notification of hotplug events.
  */
type libusb_hotplug_callback struct {
	 // Context this callback is associated with 
	 ctx *libusb_context

	 // Vendor ID to match or LIBUSB_HOTPLUG_MATCH_ANY 
	 vendor_id int 
 
	 // Product ID to match or LIBUSB_HOTPLUG_MATCH_ANY 
	 product_id int
 
	 // Device class to match or LIBUSB_HOTPLUG_MATCH_ANY 
	dev_class int 
 
	 // Hotplug callback flags 
	flags libusb_hotplug_flag
 
	 // Event(s) that will trigger this callback 
	events libusb_hotplug_event
 
	 // Callback function to invoke for matching event/device 
	cb libusb_hotplug_callback_fn
 
	 // Handle for this callback (used to match on deregister) 
	handle libusb_hotplug_callback_handle
 
	 // User data that will be passed to the callback function 
	user_data interface{}
 
	 // Callback is marked for deletion 
	needs_free int
 
	 // List this callback is registered in (ctx.hotplug_cbs) 
	list list_head
 }
 
 type libusb_hotplug_message struct {
	// The hotplug event that occurred 
	event libusb_hotplug_event
 
	// The device for which this hotplug event occurred 
	device *libusb_device
 
	// List this message is contained in (ctx.hotplug_msgs) 
	list list_head
 }
 
func usbi_hotplug_match_cb(ctx *libusb_context, dev *libusb_device, 
	event libusb_hotplug_event, hotplug_cb *libusb_hotplug_callback) bool {

	/* Handle lazy deregistration of callback */
	if (hotplug_cb.needs_free) {
		/* Free callback */
		return true
	}

	if (!(hotplug_cb.events & event)) {
		return false
	}

	if (LIBUSB_HOTPLUG_MATCH_ANY != hotplug_cb.vendor_id &&
	    hotplug_cb.vendor_id != dev.device_descriptor.idVendor) {
		return false
	}

	if (LIBUSB_HOTPLUG_MATCH_ANY != hotplug_cb.product_id &&
	    hotplug_cb.product_id != dev.device_descriptor.idProduct) {
		return false
	}

	if (LIBUSB_HOTPLUG_MATCH_ANY != hotplug_cb.dev_class &&
	    hotplug_cb.dev_class != dev.device_descriptor.bDeviceClass) {
		return false
	}

	return hotplug_cb.cb(ctx, dev, event, hotplug_cb.user_data)
}

func usbi_hotplug_match(ctx *libusb_context, dev *libusb_device, event libusb_hotplug_event) {

	var next, hotplug_cb *libusb_hotplug_callback
	var ret int

	usbi_mutex_lock(ctx.hotplug_cbs_lock);

	list_for_each_entry_safe(hotplug_cb, next, &ctx.hotplug_cbs, list, libusb_hotplug_callback) {
		usbi_mutex_unlock(ctx.hotplug_cbs_lock);
		ret = usbi_hotplug_match_cb (ctx, dev, event, hotplug_cb);
		usbi_mutex_lock(ctx.hotplug_cbs_lock);

		if (ret) {
			list_del(&hotplug_cb.list);
			
		}
	}

	usbi_mutex_unlock(ctx.hotplug_cbs_lock)

	/* the backend is expected to call the callback for each active transfer */
}

usbi_hotplug_notification(ctx *libusb_context, dev *libusb_device, event libusb_hotplug_event)
{
	int pending_events;
	libusb_hotplug_message *message = calloc(1, sizeof(*message));

	if (!message) {
		usbi_err(ctx, "error allocating hotplug message");
		return;
	}

	message.event = event;
	message.device = dev;

	/* Take the event data lock and add this message to the list.
	 * Only signal an event if there are no prior pending events. */
	usbi_mutex_lock(&ctx.event_data_lock);
	pending_events = usbi_pending_events(ctx);
	list_add_tail(&message.list, &ctx.hotplug_msgs);
	if (!pending_events)
		usbi_signal_event(ctx);
	usbi_mutex_unlock(&ctx.event_data_lock);
}

int  libusb_hotplug_register_callback(libusb_context *ctx,
	libusb_hotplug_event events, libusb_hotplug_flag flags,
	int vendor_id, int product_id, int dev_class,
	libusb_hotplug_callback_fn cb_fn, void *user_data,
	libusb_hotplug_callback_handle *callback_handle)
{
	libusb_hotplug_callback *new_callback;
	static int handle_id = 1;

	/* check for hotplug support */
	if (!libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {
		return LIBUSB_ERROR_NOT_SUPPORTED;
	}

	/* check for sane values */
	if ((LIBUSB_HOTPLUG_MATCH_ANY != vendor_id && (~0xffff & vendor_id)) ||
	    (LIBUSB_HOTPLUG_MATCH_ANY != product_id && (~0xffff & product_id)) ||
	    (LIBUSB_HOTPLUG_MATCH_ANY != dev_class && (~0xff & dev_class)) ||
	    !cb_fn) {
		return LIBUSB_ERROR_INVALID_PARAM;
	}

	USBI_GET_CONTEXT(ctx);

	new_callback = (libusb_hotplug_callback *)calloc(1, sizeof (*new_callback));
	if (!new_callback) {
		return LIBUSB_ERROR_NO_MEM;
	}

	new_callback.ctx = ctx;
	new_callback.vendor_id = vendor_id;
	new_callback.product_id = product_id;
	new_callback.dev_class = dev_class;
	new_callback.flags = flags;
	new_callback.events = events;
	new_callback.cb = cb_fn;
	new_callback.user_data = user_data;
	new_callback.needs_free = 0;

	usbi_mutex_lock(&ctx.hotplug_cbs_lock);

	/* protect the handle by the context hotplug lock. it doesn't matter if the same handle
	 * is used for different contexts only that the handle is unique for this context */
	new_callback.handle = handle_id++;

	list_add(&new_callback.list, &ctx.hotplug_cbs);

	usbi_mutex_unlock(&ctx.hotplug_cbs_lock);


	if (flags & LIBUSB_HOTPLUG_ENUMERATE) {
		int i, len;
		struct libusb_device **devs;

		len = (int) libusb_get_device_list(ctx, &devs);
		if (len < 0) {
			libusb_hotplug_deregister_callback(ctx,
							new_callback.handle);
			return len;
		}

		for (i = 0; i < len; i++) {
			usbi_hotplug_match_cb(ctx, devs[i],
					LIBUSB_HOTPLUG_EVENT_DEVICE_ARRIVED,
					new_callback);
		}

		libusb_free_device_list(devs, 1);
	}


	if (callback_handle)
		*callback_handle = new_callback.handle;

	return LIBUSB_SUCCESS;
}

void  libusb_hotplug_deregister_callback (struct libusb_context *ctx,
	libusb_hotplug_callback_handle callback_handle)
{
	struct libusb_hotplug_callback *hotplug_cb;

	/* check for hotplug support */
	if (!libusb_has_capability(LIBUSB_CAP_HAS_HOTPLUG)) {
		return;
	}

	USBI_GET_CONTEXT(ctx);

	usbi_mutex_lock(&ctx.hotplug_cbs_lock);
	list_for_each_entry(hotplug_cb, &ctx.hotplug_cbs, list,
			    struct libusb_hotplug_callback) {
		if (callback_handle == hotplug_cb.handle) {
			/* Mark this callback for deregistration */
			hotplug_cb.needs_free = 1;
		}
	}
	usbi_mutex_unlock(&ctx.hotplug_cbs_lock);

	usbi_hotplug_notification(ctx, NULL, 0);
}

void usbi_hotplug_deregister_all(struct libusb_context *ctx) {
	struct libusb_hotplug_callback *hotplug_cb, *next;

	usbi_mutex_lock(&ctx.hotplug_cbs_lock);
	list_for_each_entry_safe(hotplug_cb, next, &ctx.hotplug_cbs, list,
				 struct libusb_hotplug_callback) {
		list_del(&hotplug_cb.list);
		
	}

	usbi_mutex_unlock(&ctx.hotplug_cbs_lock);
}
