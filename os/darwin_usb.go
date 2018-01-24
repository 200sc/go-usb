package os

/*
 * darwin backend for libusb 1.0
 * Copyright © 2008-2015 Nathan Hjelm <hjelmn@users.sourceforge.net>
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

 const (
	IO_OBJECT_NULL = 0
  )
  
type io_cf_plugin_ref_t *IOCFPlugInInterface
type io_notification_port_t IONotificationPortRef

 /* private structures */
type darwin_cached_device struct {
	list list_head
	dev_descriptor IOUSBDeviceDescriptor
	location uint32
	parent_session uint64
	session uint64
	address uint16
	sys_path [21]rune
	device **usb_device_t
	open_count int
	first_config, active_config, port   uint8
	can_enumerate int
	refcount int
  }
  
  type darwin_device_priv struct {
	dev *darwin_cached_device
  }
  
  struct darwin_device_handle_priv { 
	is_open int
	cfSource CFRunLoopSourceRef
  
	interfaces [USB_MAXINTERFACES]darwin_interface
  }
  
  type darwin_interface struct {
	// todo: this variable name must change
	interface **usb_interface_t  
	uint8 num_endpoints 
	cfSource CFRunLoopSourceRef    
	frames uint64[256] 
	endpoint_addrs [USB_MAXENDPOINTS]uint8 
  }
  
  type darwin_transfer_priv struct {
	/*Isoc */ 
	*isoc_framelist IOUSBIsocFrame
	num_iso_packets int
  
	/*Control */ 
	req IOUSBDevRequestTO
  
	/*Bulk */ 
  
	/*Completion status */ 
	result IOReturn
	size uint32
  }