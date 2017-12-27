package usb

/*
 * libusb strerror code
 * Copyright © 2013 Hans de Goede <hdegoede@redhat.com>
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

var (
	usbi_locale_supported = []string{"en", "nl", "fr", "ru"}
	usbi_localized_errors = [][]string{
		{ /* English (en) */
			"Success",
			"Input/Output Error",
			"Invalid parameter",
			"Access denied (insufficient permissions)",
			"No such device (it may have been disconnected)",
			"Entity not found",
			"Resource busy",
			"Operation timed out",
			"Overflow",
			"Pipe error",
			"System call interrupted (perhaps due to signal)",
			"Insufficient memory",
			"Operation not supported or unimplemented on this platform",
			"Other error",
		}, { /* Dutch (nl) */
			"Gelukt",
			"Invoer-/uitvoerfout",
			"Ongeldig argument",
			"Toegang geweigerd (onvoldoende toegangsrechten)",
			"Apparaat bestaat niet (verbinding met apparaat verbroken?)",
			"Niet gevonden",
			"Apparaat of hulpbron is bezig",
			"Bewerking verlopen",
			"Waarde is te groot",
			"Gebroken pijp",
			"Onderbroken systeemaanroep",
			"Onvoldoende geheugen beschikbaar",
			"Bewerking wordt niet ondersteund",
			"Andere fout",
		}, { /* French (fr) */
			"Succès",
			"Erreur d'entrée/sortie",
			"Paramètre invalide",
			"Accès refusé (permissions insuffisantes)",
			"Périphérique introuvable (peut-être déconnecté)",
			"Elément introuvable",
			"Resource déjà occupée",
			"Operation expirée",
			"Débordement",
			"Erreur de pipe",
			"Appel système abandonné (peut-être à cause d’un signal)",
			"Mémoire insuffisante",
			"Opération non supportée or non implémentée sur cette plateforme",
			"Autre erreur",
		}, { /* Russian (ru) */
			"Успех",
			"Ошибка ввода/вывода",
			"Неверный параметр",
			"Доступ запрещён (не хватает прав)",
			"Устройство отсутствует (возможно, оно было отсоединено)",
			"Элемент не найден",
			"Ресурс занят",
			"Истекло время ожидания операции",
			"Переполнение",
			"Ошибка канала",
			"Системный вызов прерван (возможно, сигналом)",
			"Память исчерпана",
			"Операция не поддерживается данной платформой",
			"Неизвестная ошибка"
		}
	}
}

type Locale string
const (
	ENGLISH Locale = "en"
	DUTCH Locale = "nl" 
	FRENCH Locale = "fr"
	RUSSIAN Locale = "ru"
)

var (
	localeIndex = map[Locale]int{
		ENGLISH: 0,
		DUTCH: 1,
		FRENCH: 2,
		RUSSIAN: 3,
	}
	usbi_locale = 0
)

/** \ingroup libusb_misc
 * Set the language, and only the language, not the encoding! used for
 * translatable libusb messages.
 *
 * This takes a locale string in the default setlocale format: lang[-region]
 * or lang[_country_region][.codeset]. Only the lang part of the string is
 * used, and only 2 letter ISO 639-1 codes are accepted for it, such as "de".
 * The optional region, country_region or codeset parts are ignored. This
 * means that functions which return translatable strings will NOT honor the
 * specified encoding. 
 * All strings returned are encoded as UTF-8 strings.
 *
 * If libusb_setlocale() is not called, all messages will be in English.
 *
 * The following functions return translatable strings: libusb_strerror().
 * Note that the libusb log messages controlled through libusb_set_debug()
 * are not translated, they are always in English.
 *
 * For POSIX UTF-8 environments if you want libusb to follow the standard
 * locale settings, call libusb_setlocale(setlocale(LC_MESSAGES, NULL)),
 * after your app has done its locale setup.
 *
 * \param locale locale-string in the form of lang[_country_region][.codeset]
 * or lang[-region], where lang is a 2 letter ISO 639-1 code
 * \returns LIBUSB_SUCCESS on success
 * \returns LIBUSB_ERROR_INVALID_PARAM if the locale doesn't meet the requirements
 * \returns LIBUSB_ERROR_NOT_FOUND if the requested language is not supported
 * \returns a LIBUSB_ERROR code on other errors
 */

func libusb_setlocale(locale Locale) libusb_error {
	found, ok := localeIndex[locale]
	if !ok {
		return LIBUSB_ERROR_NOT_FOUND
	}
	usbi_locale = found;
	return LIBUSB_SUCCESS;
}

/** \ingroup libusb_misc
 * Returns a constant string with a short description of the given error code,
 * this description is intended for displaying to the end user and will be in
 * the language set by libusb_setlocale().
 *
 * The returned string is encoded in UTF-8.
 *
 * The messages always start with a capital letter and end without any dot.
 * The caller must not free() the returned string.
 *
 * \param errcode the error code whose description is desired
 * \returns a short description of the error code in UTF-8 encoding
 */
func libusb_strerror(errcode libusb_error) string {
	errcode_index := -errcode

	if (errcode_index < 0) || (errcode_index >= LIBUSB_ERROR_COUNT) {
		/* "Other Error", which should always be our last message, is returned */
		errcode_index = LIBUSB_ERROR_COUNT - 1
	}

	return usbi_localized_errors[usbi_locale][errcode_index]
}

/** \ingroup libusb_misc
 * Returns a constant NULL-terminated string with the ASCII name of a libusb
 * error or transfer status code. The caller must not free() the returned
 * string.
 *
 * \param error_code The \ref libusb_error or libusb_transfer_status code to
 * return the name of.
 * \returns The error name, or the string **UNKNOWN** if the value of
 * error_code is not a known error / status code.
 */
func libusb_error_name(error_code libusb_error) string {
	 switch (error_code) {
	 case LIBUSB_ERROR_IO:
		 return "LIBUSB_ERROR_IO";
	 case LIBUSB_ERROR_INVALID_PARAM:
		 return "LIBUSB_ERROR_INVALID_PARAM";
	 case LIBUSB_ERROR_ACCESS:
		 return "LIBUSB_ERROR_ACCESS";
	 case LIBUSB_ERROR_NO_DEVICE:
		 return "LIBUSB_ERROR_NO_DEVICE";
	 case LIBUSB_ERROR_NOT_FOUND:
		 return "LIBUSB_ERROR_NOT_FOUND";
	 case LIBUSB_ERROR_BUSY:
		 return "LIBUSB_ERROR_BUSY";
	 case LIBUSB_ERROR_TIMEOUT:
		 return "LIBUSB_ERROR_TIMEOUT";
	 case LIBUSB_ERROR_OVERFLOW:
		 return "LIBUSB_ERROR_OVERFLOW";
	 case LIBUSB_ERROR_PIPE:
		 return "LIBUSB_ERROR_PIPE";
	 case LIBUSB_ERROR_INTERRUPTED:
		 return "LIBUSB_ERROR_INTERRUPTED";
	 case LIBUSB_ERROR_NO_MEM:
		 return "LIBUSB_ERROR_NO_MEM";
	 case LIBUSB_ERROR_NOT_SUPPORTED:
		 return "LIBUSB_ERROR_NOT_SUPPORTED";
	 case LIBUSB_ERROR_OTHER:
		 return "LIBUSB_ERROR_OTHER";
	 case LIBUSB_TRANSFER_ERROR:
		 return "LIBUSB_TRANSFER_ERROR";
	 case LIBUSB_TRANSFER_TIMED_OUT:
		 return "LIBUSB_TRANSFER_TIMED_OUT";
	 case LIBUSB_TRANSFER_CANCELLED:
		 return "LIBUSB_TRANSFER_CANCELLED";
	 case LIBUSB_TRANSFER_STALL:
		 return "LIBUSB_TRANSFER_STALL";
	 case LIBUSB_TRANSFER_NO_DEVICE:
		 return "LIBUSB_TRANSFER_NO_DEVICE";
	 case LIBUSB_TRANSFER_OVERFLOW:
		 return "LIBUSB_TRANSFER_OVERFLOW";
	 case 0:
		 return "LIBUSB_SUCCESS / LIBUSB_TRANSFER_COMPLETED";
	 default:
		 return "**UNKNOWN**";
	 }
 }