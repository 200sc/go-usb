package usb

/*
 * libusb list primitives
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

type LinkedList struct {
	prev, next  *LinkedList
	member interface{}
}

func list_empty(ll *LinkedList) bool {
	return ll.next == ll
}

// todo: this shouldn't take in the list to init, it should be a constructor
func list_init(ll *LinkedList) {
	ll.prev = ll
	ll.next = ll
}

func list_add(entry, head *LinkedList) {
	entry.next = head.next
	entry.prev = head

	head.next.prev = entry
	head.next = entry
}

func list_add_tail(entry, head *LinkedList) {
	entry.next = head
	entry.prev = head.prev

	head.prev.next = entry
	head.prev = entry
}

func list_del(entry *LinkedList) {
	entry.next.prev = entry.prev
	entry.prev.next = entry.next
	entry.next = nil
	entry.prev = nil
}

// these 'modular' defines are borked

/* Get an entry from the list
 *  ptr - the address of this list_head element in "type"
 *  type - the data type that contains "member"
 *  member - the list_head element in "type"
 */
 #define list_entry(ptr, type, member) \
 ((type *)((uintptr_t)(ptr) - (uintptr_t)offsetof(type, member)))

#define list_first_entry(ptr, type, member) \
 list_entry((ptr).next, type, member)