# php-cgi on Windows: concurrency probe

Experiment for a native PHP runtime on Windows, where PHP-FPM does not exist.
`main.go` is a minimal FastCGI client that fires concurrent requests at a
php-cgi listener and reports which worker PID answered each one.

```
go build -o phpcgi-probe.exe ./experiments/phpcgi
phpcgi-probe.exe -addr 127.0.0.1:9484 -script C:\path\sleep.php -c 8 -n 8 -q ms=1000
```

## Findings (2026-10-08, official php-8.4.26-nts-Win32-vs17-x64, Windows 11)

`sleep.php` sleeps `ms` milliseconds and echoes `getmypid()`.

| Case | Result |
| --- | --- |
| No `PHP_FCGI_CHILDREN`, 8 x 1s concurrent | 1 process, serialised: 8.08s wall |
| `PHP_FCGI_CHILDREN=4`, 8 x 1s concurrent | master + 4 children on one port, 2.02s wall |
| `PHP_FCGI_CHILDREN=4`, `PHP_FCGI_MAX_REQUESTS=3`, 40 requests | 40/40 ok, 16 distinct children, master keeps 4 alive |
| Kill one child | master spawns a replacement, next 8 requests all ok |
| Kill the master | every child exits with it, port released, no orphans |
| OPcache (`php_opcache.dll`), 4 children | shared: first child misses 51, the others hit the same cache |
| FastCGI from `lerd-nginx` to `host.containers.internal:9484` | works with php-cgi bound to 127.0.0.1 (gvproxy forwards to host loopback), Windows `SCRIPT_FILENAME` resolves |
| Per-request overhead through gvproxy | ~1.2ms vs ~0.5ms from the host (50 trivial requests: 58ms vs 25ms) |

So php-cgi.exe's own `PHP_FCGI_CHILDREN` handling is a usable process manager
on Windows: a supervisor only has to keep the master alive.

## Filesystem cost (the reason to do this)

`bench.php`: 2000 small files, `filemtime` each then `include` each, OPcache off.

| Runtime | stat | include |
| --- | --- | --- |
| Container, project over 9p (PHP 8.5.11) | ~2070ms | ~5300ms |
| Native Windows php.exe (PHP 8.4.26) | ~46ms | ~198ms |

About 45x on stat and 27x on include, against 27x and 10x measured on macOS
(see `docs/features/native-runtime.md`). OPcache with revalidation off hides
part of this on warm requests in the container; cold requests, composer,
artisan and anything that stats files do not get that relief.

## Extensions on the official build

Official windows.php.net builds link against `php8.dll`, so they load DLL
extensions; static-php-cli builds cannot load any (its FAQ), which rules out
toggling xdebug and the collector per version the way the native runtime does.

**Xdebug**: the official `php_xdebug-3.5.3-8.4-nts-vs17-x86_64.dll` (sha256
checked against xdebug.org) loads, and a FastCGI request to php-cgi with 4
children opened a real DBGp session on 127.0.0.1:9003. The `fileuri` is a
Windows path (`file:///C:/...`), so a Windows IDE needs no path mapping.

**lerd_devtools**: builds as `php_lerd_devtools.dll` from the same
`lerd_devtools.c`, with `config.w32` and `lerd_devtools_win32.h` (Winsock
transport, `clock_gettime`/`gmtime_r`/`getpid`/`F_OK` stand-ins), all behind
`PHP_WIN32` so the Linux build is unchanged (checked in php:8.4-cli-alpine).
Under php-cgi with 4 children it sent every PDO `exec`, `query` and prepared
`execute` with sql, bindings, time and a Windows-path trace; nothing is sent
with the flag file absent.

- The module must be linked with the same MSVC toolset as the PHP core: a DLL
  from Visual C++ 2026 (linker 14.51) is refused by 8.4.26 (14.44). Build with
  VS 2022 Build Tools, and pin the toolset in CI to whatever php.net used.
- Windows does not refuse a connect to a closed loopback port immediately, so
  with lerd-ui down every query cost ~60ms (4 queries: 250ms per request). The
  Windows transport now backs off for a second after a failed send: 20 requests
  with lerd-ui down took 82ms in total, and capture resumes within a second of
  lerd-ui coming back.

## PECL extensions

Of the 54 extensions `lerd-env/php` builds, the official 8.4.26 zip carries all
but 12. php.net still publishes PECL DLLs at
`downloads.php.net/~windows/pecl/releases/<ext>/<version>/`, for every PHP
lerd supports (8.1 to 8.5, NTS x64; vs16 up to 8.3, vs17 from 8.4):

| Extension | Windows DLL | Same version as lerd's image | Checked on 8.4 |
| --- | --- | --- | --- |
| redis | 6.3.0 | yes | set/get against lerd-redis |
| igbinary | 3.2.16, 3.2.17RC1 | yes (image runs RC1) | serialize round-trip |
| mongodb | 2.5.3 | yes | BSON + Manager, no server queried |
| imagick | 3.8.1 (ImageMagick 7.1.1) | yes | PNG render, 273 formats |
| pcov | 1.0.12 | yes | coverage collected |
| xdebug | 3.5.3 (xdebug.org) | yes | DBGp session |
| spx | none | n/a | SPX supports Linux, macOS and FreeBSD only |
| pcntl, posix, sysvmsg, sysvsem | none | n/a | do not exist on Windows |

All load and work under php-cgi with 4 children too. Notes for packaging:

- imagick needs its ~200 ImageMagick DLLs (`CORE_RL_*`, `IM_MOD_RL_*`,
  `FILTER_*`) beside php.exe or on PATH, or it fails with "module not found".
- No checksums are published for PECL DLLs; the build repo has to record its
  own sha256 when it repackages them.
- phpredis has no igbinary serializer and no compression on Windows, but the
  container image's phpredis has none either, so that is parity.
- The vs16/vs17 split means lerd_devtools needs VS 2019 for 8.1 to 8.3 and VS
  2022 for 8.4 and 8.5, to match each core's linker version.

## Not covered yet

- No pool sizing on demand: children are fixed at start, unlike FPM's `ondemand`.
- nginx vhost still sends VM paths; a native site needs `SCRIPT_FILENAME` and
  `DOCUMENT_ROOT` translated to the Windows path.
- SPX has no Windows support. Decision (2026-10-08): the native Windows
  runtime profiles with Xdebug's `profile` mode first. It loads per request
  through `XDEBUG_TRIGGER` the way SPX does through its cookie, and writes
  cachegrind; lerd needs a cachegrind reader and a flat/call-tree view (no
  timeline). Measured ~3.3x overhead on a CPU-bound script, and no CPU time
  from xhprof on Windows, which is why xhprof was not picked.
- `pcntl`, `posix` and the `sysv*` extensions do not exist on Windows.
- Adding files under `internal/podman/devtools/` moves the source hash the
  Containerfile carries, so `TestDevtoolsSourceMarkerInSync` fails on this
  branch until the marker is updated.
