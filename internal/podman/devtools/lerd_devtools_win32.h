/* Windows stand-ins for the POSIX calls the collector makes, so the native
 * runtime on Windows builds the same lerd_devtools.c as the images. */
#ifndef LERD_DEVTOOLS_WIN32_H
#define LERD_DEVTOOLS_WIN32_H

#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>
#include <time.h>
#include <process.h>

#ifndef CLOCK_REALTIME
#define CLOCK_REALTIME 0
#endif
#ifndef CLOCK_MONOTONIC
#define CLOCK_MONOTONIC 1
#endif

static int lerd_clock_gettime(int clk, struct timespec *ts)
{
	if (clk == CLOCK_MONOTONIC) {
		LARGE_INTEGER freq, now;
		QueryPerformanceFrequency(&freq);
		QueryPerformanceCounter(&now);
		ts->tv_sec = (time_t)(now.QuadPart / freq.QuadPart);
		ts->tv_nsec = (long)((now.QuadPart % freq.QuadPart) * 1000000000LL / freq.QuadPart);
		return 0;
	}
	return timespec_get(ts, TIME_UTC) == TIME_UTC ? 0 : -1;
}
#define clock_gettime lerd_clock_gettime
#define getpid _getpid

static struct tm *lerd_gmtime_r(const time_t *t, struct tm *out)
{
	return gmtime_s(out, t) == 0 ? out : NULL;
}
#define gmtime_r lerd_gmtime_r
#ifndef F_OK
#define F_OK 0
#endif

/* lerd_win32_send is lerd_send over Winsock: tcp://host:port only, since the
 * native runtime reaches lerd-ui over loopback and never a unix socket. Same
 * fire-and-forget contract, with the same 50ms connect budget. */
/* Windows answers a connect to a closed loopback port with retried SYNs, not
 * an immediate refusal, so with lerd-ui down every send would burn the whole
 * budget. One failure holds sends off for a second instead. */
static ULONGLONG lerd_win32_down_until = 0;

static void lerd_win32_mark_down(void)
{
	lerd_win32_down_until = GetTickCount64() + 1000;
}

static void lerd_win32_send(const char *host, const char *line, size_t len)
{
	if (!host || !host[0] || strncmp(host, "unix://", 7) == 0) {
		return;
	}
	if (GetTickCount64() < lerd_win32_down_until) {
		return;
	}
	const char *hp = strncmp(host, "tcp://", 6) == 0 ? host + 6 : host;
	char hbuf[256];
	strncpy(hbuf, hp, sizeof(hbuf) - 1);
	hbuf[sizeof(hbuf) - 1] = '\0';
	char *colon = strrchr(hbuf, ':');
	if (!colon || atoi(colon + 1) <= 0) {
		return;
	}
	*colon = '\0';

	struct addrinfo hints, *res = NULL;
	memset(&hints, 0, sizeof(hints));
	hints.ai_family = AF_UNSPEC;
	hints.ai_socktype = SOCK_STREAM;
	if (getaddrinfo(hbuf, colon + 1, &hints, &res) != 0 || !res) {
		return;
	}
	SOCKET s = socket(res->ai_family, res->ai_socktype, res->ai_protocol);
	if (s == INVALID_SOCKET) {
		freeaddrinfo(res);
		return;
	}
	u_long nonblock = 1;
	ioctlsocket(s, FIONBIO, &nonblock);
	if (connect(s, res->ai_addr, (int)res->ai_addrlen) != 0 && WSAGetLastError() != WSAEWOULDBLOCK) {
		freeaddrinfo(res);
		closesocket(s);
		lerd_win32_mark_down();
		return;
	}
	freeaddrinfo(res);

	/* select, not WSAPoll: older WSAPoll never reports a refused connect. */
	fd_set wr, ex;
	FD_ZERO(&wr);
	FD_ZERO(&ex);
	FD_SET(s, &wr);
	FD_SET(s, &ex);
	struct timeval tv = {0, 50000};
	if (select(0, NULL, &wr, &ex, &tv) <= 0 || FD_ISSET(s, &ex) || !FD_ISSET(s, &wr)) {
		closesocket(s);
		lerd_win32_mark_down();
		return;
	}

	size_t off = 0;
	while (off < len) {
		int n = send(s, line + off, (int)(len - off), 0);
		if (n <= 0) {
			break;
		}
		off += (size_t)n;
	}
	if (off >= len) {
		(void)send(s, "\n", 1, 0);
	}
	closesocket(s);
}

#endif
