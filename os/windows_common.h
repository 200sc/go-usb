#if defined(__CYGWIN__ )
#define _stricmp strcasecmp
#define _snprintf snprintf
#define _strdup strdup
// _beginthreadex is MSVCRT => unavailable for cygwin. Fallback to using CreateThread
#define _beginthreadex(a, b, c, d, e, f) CreateThread(a, b, (LPTHREAD_START_ROUTINE)c, d, e, (LPDWORD)f)
#endif

#define safe_closehandle(h) do {if (h != INVALID_HANDLE_VALUE) {CloseHandle(h); h = INVALID_HANDLE_VALUE;}} while(0)
#define safe_strcp(dst, dst_max, src, count) do {memcpy(dst, src, safe_min(count, dst_max)); \
	((char*)dst)[safe_min(count, dst_max)-1] = 0;} while(0)
#define safe_strcpy(dst, dst_max, src) safe_strcp(dst, dst_max, src, safe_strlen(src)+1)
#define safe_strncat(dst, dst_max, src, count) strncat(dst, src, safe_min(count, dst_max - safe_strlen(dst) - 1))
#define safe_strcat(dst, dst_max, src) safe_strncat(dst, dst_max, src, safe_strlen(src)+1)
#define safe_strcmp(str1, str2) strcmp(((str1==NULL)?"<NULL>":str1), ((str2==NULL)?"<NULL>":str2))
#define safe_stricmp(str1, str2) _stricmp(((str1==NULL)?"<NULL>":str1), ((str2==NULL)?"<NULL>":str2))
#define safe_strncmp(str1, str2, count) strncmp(((str1==NULL)?"<NULL>":str1), ((str2==NULL)?"<NULL>":str2), count)
#define safe_strlen(str) ((str==NULL)?0:strlen(str))
#define safe_sprintf(dst, count, ...) do {_snprintf(dst, count, __VA_ARGS__); (dst)[(count)-1] = 0; } while(0)
#define safe_stprintf _sntprintf
#define safe_tcslen(str) ((str==NULL)?0:_tcslen(str))
#define safe_unref_device(dev) do {if (dev != NULL) {libusb_unref_device(dev); dev = NULL;}} while(0)
#define wchar_to_utf8_ms(wstr, str, strlen) WideCharToMultiByte(CP_UTF8, 0, wstr, -1, str, strlen, NULL, NULL)

#define ERR_BUFFER_SIZE	256


/*
 * API macros - leveraged from libusb-win32 1.x
 */
#ifndef _WIN32_WCE
#define DLL_STRINGIFY(s) #s
#define DLL_LOAD_LIBRARY(name) LoadLibraryA(DLL_STRINGIFY(name))
#else
#define DLL_STRINGIFY(s) L#s
#define DLL_LOAD_LIBRARY(name) LoadLibrary(DLL_STRINGIFY(name))
#endif

/*
 * Macros for handling DLL themselves
 */
#define DLL_DECLARE_HANDLE(name)				\
	static HMODULE __dll_##name##_handle = NULL

#define DLL_GET_HANDLE(name)					\
	do {							\
		__dll_##name##_handle = DLL_LOAD_LIBRARY(name);	\
		if (!__dll_##name##_handle)			\
			return LIBUSB_ERROR_OTHER;		\
	} while (0)

#define DLL_FREE_HANDLE(name)					\
	do {							\
		if (__dll_##name##_handle) {			\
			FreeLibrary(__dll_##name##_handle);	\
			__dll_##name##_handle = NULL;		\
		}						\
	} while(0)


#define DLL_LOAD_FUNC_PREFIXNAME(dll, prefixname, name, ret_on_failure)	\
	do {								\
		HMODULE h = __dll_##dll##_handle;			\
		prefixname = (__dll_##name##_func_t)GetProcAddress(h,	\
				DLL_STRINGIFY(name));			\
		if (prefixname)						\
			break;						\
		prefixname = (__dll_##name##_func_t)GetProcAddress(h,	\
				DLL_STRINGIFY(name) DLL_STRINGIFY(A));	\
		if (prefixname)						\
			break;						\
		prefixname = (__dll_##name##_func_t)GetProcAddress(h,	\
				DLL_STRINGIFY(name) DLL_STRINGIFY(W));	\
		if (prefixname)						\
			break;						\
		if (ret_on_failure)					\
			return LIBUSB_ERROR_NOT_FOUND;			\
	} while(0)

#define DLL_LOAD_FUNC(dll, name, ret_on_failure)			\
	DLL_LOAD_FUNC_PREFIXNAME(dll, name, name, ret_on_failure)
#define DLL_LOAD_FUNC_PREFIXED(dll, prefix, name, ret_on_failure)	\
	DLL_LOAD_FUNC_PREFIXNAME(dll, prefix##name, name, ret_on_failure)
