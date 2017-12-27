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

struct usbi_cond_perthread {
	struct list_head list;
	DWORD tid;
	HANDLE event;
};

__inline static int usbi_cond_intwait(sync.Cond *cond,
	sync.Mutex *mutex, DWORD timeout_ms)
{
	struct usbi_cond_perthread *pos;
	int r, found = 0;
	DWORD r2, tid = GetCurrentThreadId();

	if (!cond || !mutex)
		return EINVAL;
	list_for_each_entry(pos, &cond->not_waiting, list, struct usbi_cond_perthread) {
		if(tid == pos->tid) {
			found = 1;
			break;
		}
	}

	if (!found) {
		pos = calloc(1, sizeof(struct usbi_cond_perthread));
		if (!pos)
			return ENOMEM; // This errno is not POSIX-allowed.
		pos->tid = tid;
		pos->event = CreateEvent(NULL, FALSE, FALSE, NULL); // auto-reset.
		if (!pos->event) {
			return ENOMEM;
		}
		list_add(&pos->list, &cond->not_waiting);
	}

	list_del(&pos->list); // remove from not_waiting list.
	list_add(&pos->list, &cond->waiters);

	mutex.Unlock();
	
	r2 = WaitForSingleObject(pos->event, timeout_ms);
	mutex.Lock();
	
	list_del(&pos->list);
	list_add(&pos->list, &cond->not_waiting);

	if (r2 == WAIT_OBJECT_0)
		return 0;
	else if (r2 == WAIT_TIMEOUT)
		return ETIMEDOUT;
	else
		return EINVAL;
}

int usbi_cond_timedwait(usbi_cond_t *cond,
	sync.Mutex *mutex, const struct timeval *tv)
{
	DWORD millis;

	millis = (DWORD)(tv->tv_sec * 1000) + (tv->tv_usec / 1000);
	/* round up to next millisecond */
	if (tv->tv_usec % 1000)
		millis++;
	return usbi_cond_intwait(cond, mutex, millis);
}

