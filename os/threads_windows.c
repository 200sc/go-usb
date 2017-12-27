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

int usbi_mutex_static_lock(usbi_mutex_static_t *mutex)
{
	if (!mutex)
		return EINVAL;
	while (InterlockedExchange(mutex, 1) == 1)
		SleepEx(0, TRUE);
	return 0;
}

int usbi_mutex_static_unlock(usbi_mutex_static_t *mutex)
{
	if (!mutex)
		return EINVAL;
	InterlockedExchange(mutex, 0);
	return 0;
}

int usbi_cond_init(usbi_cond_t *cond)
{
	if (!cond)
		return EINVAL;
	list_init(&cond->waiters);
	list_init(&cond->not_waiting);
	return 0;
}

int usbi_cond_broadcast(usbi_cond_t *cond)
{
	// Assumes mutex is locked; this is not in keeping with POSIX spec, but
	//   libusb does this anyway, so we simplify by not adding more sync
	//   primitives to the CV definition!
	int fail = 0;
	struct usbi_cond_perthread *pos;

	if (!cond)
		return EINVAL;
	list_for_each_entry(pos, &cond->waiters, list, struct usbi_cond_perthread) {
		if (!SetEvent(pos->event))
			fail = 1;
	}
	// The wait function will remove its respective item from the list.
	return fail ? EINVAL : 0;
}

__inline static int usbi_cond_intwait(usbi_cond_t *cond,
	usbi_mutex_t *mutex, DWORD timeout_ms)
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

	r  = usbi_mutex_unlock(mutex);
	if (r)
		return r;

	r2 = WaitForSingleObject(pos->event, timeout_ms);
	r = usbi_mutex_lock(mutex);
	if (r)
		return r;

	list_del(&pos->list);
	list_add(&pos->list, &cond->not_waiting);

	if (r2 == WAIT_OBJECT_0)
		return 0;
	else if (r2 == WAIT_TIMEOUT)
		return ETIMEDOUT;
	else
		return EINVAL;
}
// N.B.: usbi_cond_*wait() can also return ENOMEM, even though pthread_cond_*wait cannot!
int usbi_cond_wait(usbi_cond_t *cond, usbi_mutex_t *mutex)
{
	return usbi_cond_intwait(cond, mutex, INFINITE);
}

int usbi_cond_timedwait(usbi_cond_t *cond,
	usbi_mutex_t *mutex, const struct timeval *tv)
{
	DWORD millis;

	millis = (DWORD)(tv->tv_sec * 1000) + (tv->tv_usec / 1000);
	/* round up to next millisecond */
	if (tv->tv_usec % 1000)
		millis++;
	return usbi_cond_intwait(cond, mutex, millis);
}

int usbi_tls_key_create(usbi_tls_key_t *key)
{
	if (!key)
		return EINVAL;
	*key = TlsAlloc();
	if (*key == TLS_OUT_OF_INDEXES)
		return ENOMEM;
	else
		return 0;
}

void *usbi_tls_key_get(usbi_tls_key_t key)
{
	return TlsGetValue(key);
}

int usbi_tls_key_set(usbi_tls_key_t key, void *value)
{
	if (TlsSetValue(key, value))
		return 0;
	else
		return EINVAL;
}

int usbi_tls_key_delete(usbi_tls_key_t key)
{
	if (TlsFree(key))
		return 0;
	else
		return EINVAL;
}

int usbi_get_tid(void)
{
	return (int)GetCurrentThreadId();
}
