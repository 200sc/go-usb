/*
 * Windows backend common header for libusb 1.0
 *
 * This file brings together header code common between
 * the desktop Windows backends.
 * Copyright © 2012-2013 RealVNC Ltd.
 * Copyright © 2009-2012 Pete Batard <pete@akeo.ie>
 * With contributions from Michael Plante, Orin Eman et al.
 * Parts of this code adapted from libusb-win32-v1 by Stephan Meyer
 * Major code testing contribution by Xiaofan Chen
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

// Missing from MinGW
#if !defined(FACILITY_SETUPAPI)
#define FACILITY_SETUPAPI	15
#endif

typedef struct USB_CONFIGURATION_DESCRIPTOR {
  uint8  bLength;
  uint8  bDescriptorType;
  uint16 wTotalLength;
  uint8  bNumInterfaces;
  uint8  bConfigurationValue;
  uint8  iConfiguration;
  uint8  bmAttributes;
  uint8  MaxPower;
} USB_CONFIGURATION_DESCRIPTOR, *PUSB_CONFIGURATION_DESCRIPTOR;

typedef struct libusb_device_descriptor USB_DEVICE_DESCRIPTOR, *PUSB_DEVICE_DESCRIPTOR;