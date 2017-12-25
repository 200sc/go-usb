/*
 * libusb synchronization on Microsoft Windows
 *
 * Copyright © 2010 Michael Plante <michael.plante@gmail.com>
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

#ifndef LIBUSB_THREADS_WINDOWS_H
#define LIBUSB_THREADS_WINDOWS_H

#define usbi_mutex_static_t	volatile LONG
#define USBI_MUTEX_INITIALIZER	0

typedef struct usbi_cond {
	// Every time a thread touches the CV, it winds up in one of these lists.
	//   It stays there until the CV is destroyed, even if the thread terminates.
	struct list_head waiters;
	struct list_head not_waiting;
} usbi_cond_t;

// We *were* getting ETIMEDOUT from pthread.h:
#ifndef ETIMEDOUT
#  define ETIMEDOUT 10060     /* This is the value in winsock.h. */
#endif

#define usbi_tls_key_t		DWORD

#endif /* LIBUSB_THREADS_WINDOWS_H */
