# IBMi (PASE) Test Results — `iseries-1.26.9`

Addendum to [README.md](README.md). Results of `./all.bash` on a native IBMi build of Go 1.26.9 with `GOEXPERIMENT=iseriesaix`.

The C reproducers for the PASE behavior described here, with build flags and commands, are in [pase_go_issues](https://github.com/JasonTashtego/pase_go_issues).

**Contents:** [Test environment](#test-environment) · [Remaining failures](#remaining-test-failures) · [Tests skipped for PASE limits](#tests-skipped-for-pase-limits) · [PASE behavior the port handles](#pase-behavior-the-port-handles) · [Process creation and open fixes](#process-creation-and-open-fixes-for-pase)

*Unverified* marks an inference that has not been confirmed on the machine.

## Test environment

| Item | Value |
|---|---|
| Hardware / OS | POWER10, OS400 V7R5 (PASE `aix/ppc64`) |
| Build | native bootstrap build from a cross-built kit, cgo on (`gcc-12`) |
| Environment | `GOMAXPROCS` unset, `GO_TEST_TIMEOUT_SCALE=3` |
| Last full `all.bash` | `ALL TESTS PASSED`: 378 packages ok, no failures, run as a profile **without** `*ALLOBJ`. This includes the phases after the main package pass (`os/user` with `osusergo`, `hash/maphash` purego, `crypto` purego, `encoding/json` with `GOEXPERIMENT=jsonv2`, `runtime/secret`, external linking, `sync -cpu=10`, the cgo tests, `runtime -cpu=1/2/4 -quick`, `../test`). No `Could not load program`, no `bad file number` in `go vet`, no `sem_init` crash, no `fatal error`. |
| Previous run | 360 packages ok, without `*ALLOBJ`. `dist` stopped after the main package pass, so the later phases did not run. Two failures: `cmd/gofmt` `TestPermissions` (whole-second file times) and `net/http` `TestFileServerDirWithRootFile` (`EACCES` instead of `ENOTDIR`). Both tests are adjusted under the experiment and pass in the last run. |
| Earlier run | 358 packages ok as a profile with `*ALLOBJ`, the only failures being the 10 permission tests (`io/ioutil`, `os` x4, `cmd/go` scripts `build_nocache`, `list_perm`, `mod_cache_rw`, `cmd/gofmt` `TestPermissions`) that cannot fail when the profile can write read-only files. |
| User profile | no special authorities. A profile with `*ALLOBJ` can write read-only files, so the permission tests above fail for it. |
| `/etc/hosts` | must be ASCII (CCSID 819). EBCDIC (CCSID 37) makes Go fall back to DNS and `localhost` fails. Convert with `iconv -f IBM-037 -t ISO8859-1`, then `setccsid 819`. |

## Remaining test failures 

**Intermittent.** These fail in some test runs and pass in others -- purely due to PASE behavior differences.

| Test | Symptom | Cause |
|---|---|---|
| `os` `TestUNIXProcessAlive` | `os: process already finished` | The test starts the `go` tool with an invalid argument, which exits at once, and then signals the child. Linux keeps an exited child as a zombie that still accepts signal 0; on PASE it apparently does not (*unverified*). |

**Profile with `*ALLOBJ`.** Run the suite as a profile without special authorities. With `*ALLOBJ`, writes to read-only files and directories on IFS succeed where the tests expect them to fail, so these tests fail (they pass in the run without `*ALLOBJ`): `io/ioutil` `TestReadOnlyWriteFile`; `os` `TestReadOnlyWriteFile`, `TestFilePermissions`, `TestRemoveAllSymlinkRemovalFailure`, `TestRemoveAllButReadOnlyAndPathError`; `cmd/go` scripts `build_nocache`, `list_perm`, `mod_cache_rw`; `cmd/gofmt` `TestPermissions`.


## Tests skipped for PASE limits

Skipped or relaxed under `GOEXPERIMENT=iseriesaix`.

| Test | PASE behavior |
|---|---|
| `os` `TestRootMultiRename` (symlink-to-directory source with a trailing slash) | `rename("link/", "target")` renames the symlink and returns no error; POSIX requires one. Exempted; the consistency check against `os.Rename` still runs. |
| `os` `TestGetwdDeep`, `TestGetwdDeepWithPWDSet` | `Getwd` works past 1,000 characters, but the path-based `RemoveAll` cannot remove a directory deeper than `PATH_MAX` (`lstat` fails with `ENAMETOOLONG`), so the test's cleanup fails. |
| `os` `TestReadDir` | reading a file as a directory returns `EBADF` instead of `ENOTDIR`; any error is accepted. |
| `os` `TestRemoveAllLongPath` | removing a path longer than `PATH_MAX` needs the `*at` calls, which fail with `EINVAL`. |
| `os` Root tests (`TestRootConsistencySymlink`, the `file slash dot` case, `TestRootChtimes`, the `Link` case of `testRootMoveFrom`, `TestRootMultiLink`, `TestRootMultiRename`) | the name-based `os.Root` does not follow a directory across renames and refuses `Link` of a symlink; PASE accepts `file/.` and a trailing `/` where POSIX requires an error, and keeps whole-second file times. These cases are skipped or relaxed under the experiment. |
| `runtime` `TestSegv` | the program sends itself SIGSEGV with `syscall.Kill`. The kernel picks the thread, and sometimes it is the sender, still in the PASE kernel entry stub (`pc 0x3608`) as `kill` returns. The runtime prints `runtime: ... unknown pc 0x3608` and the test rejects any `runtime: ` line. `runtime.syscall_syscall6` does not record `libcallpc/sp`, which the signal traceback uses to unwind out of C code, so it starts at the stub. The crash itself is reported correctly. Skipped with `SkipFlaky` (issue 49182, as on illumos and solaris). |
| `cmd/gofmt` `TestPermissions` | file times keep whole seconds, so the test's check that the modification time is unchanged compares a truncated time; the test truncates its expected time to a second. |
| `net/http` `TestFileServerDirWithRootFile` (`FileServerFS`) | `open("file/.")` on a regular file fails with `EACCES` (13) where POSIX gives `ENOTDIR`, so the server answers 403 instead of 500. The test expects 403 under the experiment. |
| `internal/syscall/unix` `TestFchmodatSymlinkNofollow` | `fchmodat` with a directory descriptor returns `EINVAL`. |
| `syscall` `TestExec`, `cmd/cgo/internal/test` `test18146` | `execve` fails with `EPERM` in a multi-threaded process. |
| `runtime` `TestMmapErrorSign` | a failed `mmap` returns `EINVAL` (22) where `ENOMEM` (12) is expected. |
| `net` `TestDialError`, `TestListenError`, `TestListenPacketError` | `getaddrinfo(NULL, "☺", ...)` succeeds with port 0 where the tests expect an unknown-service error (`getservbyname` rejects it). The `☺` subtests are skipped. |
| `cmd/cgo/internal/testerrors` `TestBadSymbol` | the XCOFF path of `cgo -dynimport` emits no symbol names, so the symbol check never runs. The injected `-badflag` does not reach the link line, so the security property holds. The same is expected on stock AIX (*unverified*). |

## PASE behavior the port handles

Found with small C programs on the IBMi; none of it is a Go problem. The programs are in [pase_go_issues](https://github.com/JasonTashtego/pase_go_issues), one document per issue. `getaddrinfo` accepting an invalid service is [issue 6](https://github.com/JasonTashtego/pase_go_issues/blob/main/06-getaddrinfo-accepts-invalid-service.md); it needs no handling in the port.

| Area | PASE behavior | How the port handles it | Issue |
|---|---|---|---|
| `*at` calls | `openat`, `fstatat`, `mkdirat`, `unlinkat`, `renameat`, `symlinkat`, `readlinkat`, `linkat`, `fchmodat` and `fchownat` fail with `EINVAL` for a directory descriptor, with any flags. Only `AT_FDCWD` works. | Path-based `RemoveAll` and a name-based `os.Root` (documented in `os/root.go`). That `os.Root` is not safe against a directory being renamed or replaced while in use, and it refuses `Link` of a symlink. | [3](https://github.com/JasonTashtego/pase_go_issues/blob/main/03-at-calls-einval.md) |
| `execve` | fails with `EPERM` ("Not owner") in a multi-threaded process, so `syscall.Exec` cannot work | `cmd/go` starts the new toolchain as a subprocess, `cmd/link` skips exec-archive, tests that exec are skipped | [4](https://github.com/JasonTashtego/pase_go_issues/blob/main/04-execve-fails-in-multithreaded-process.md) |
| Users and groups | come from the IBMi user profiles; there is no `/etc/passwd` or `/etc/group`. `getgrset` lists a user's groups (for example `"102,103"`), returns a static buffer that must not be freed, is reported not to be thread-safe (*unverified*), and takes about 8 ms per call. `getgrouplist` does not exist. | `os/user` uses `getgrset`. The `os` tests that need a sample system file use files from the Go tree. | [5](https://github.com/JasonTashtego/pase_go_issues/blob/main/05-users-and-groups.md) |
| Network interfaces | no `getifaddrs`. The routing-table dump (`getkerninfo(KINFO_RT_IFLIST)`) works. The interface ioctls accept the interface's IP address text as the name, not the line name (`ETHLINE1`, `*LOOPBACK`). | `net.Interfaces` looks the MTU up through the interface's first IPv4 address. The hardware address and multicast list are empty. | [7](https://github.com/JasonTashtego/pase_go_issues/blob/main/07-network-interfaces.md) |
| `malloc`/`free` with `posix_spawn` | a multi-threaded process that calls `malloc` and `free` in some threads while others `posix_spawn` deadlocks. Seen only in a probe, with no minimal reproducer yet. | none needed so far; whether Go can hit it is *unverified* | [8](https://github.com/JasonTashtego/pase_go_issues/blob/main/08-malloc-posix_spawn-deadlock.md) |
| Trailing slashes | `lstat("link/")` returns the link itself. `file/` on a regular file is accepted (it opens the file), and `file/.` on a regular file fails with `EACCES` where POSIX gives `ENOTDIR`. The name-based `os.Root` also accepts `file/.` and a trailing `/` on an absent name where POSIX requires an error, and `link` and `rename` of a symlink or file with a trailing slash succeed likewise. | `os.Lstat` calls `stat` for such paths. The `os.Root` tests carry narrow exemptions under the experiment. | none |
| File times | whole-second precision | none needed | none |
| `setitimer(ITIMER_PROF)` | returns `EINVAL` | the runtime ignores it; CPU profiling collects no samples | none |

## Process creation and open fixes for PASE

Three PASE bugs/issues are handled. With all three fixes, `go build`, `go vet` and `all.bash` run to completion without `GOMAXPROCS=1`.

### A signal that interrupts `close` of a pipe's write end ([issue 1](https://github.com/JasonTashtego/pase_go_issues/blob/main/01-pipe-eof-lost-after-signal-in-close.md))

| | |
|---|---|
| Symptom | `os/exec` hangs in `Wait` with the child already gone: a reader never gets end-of-file |
| Cause | `close` returns 0 and the descriptor is gone, but the pipe keeps a phantom writer. Go sends itself `SIGURG` to preempt goroutines, which is enough to trigger it. Likely a PASE defect (*unverified*). |
| C reproducer | 100,000 pipes with half a million signals delivered to other threads: none lost when the calling thread blocks signals around `close`; without blocking, one is lost within a few dozen. See [issue 1](https://github.com/JasonTashtego/pase_go_issues/blob/main/01-pipe-eof-lost-after-signal-in-close.md). |
| Fix | `syscall/close_aix.go` blocks all signals on the calling thread around the libc `close` |
| Switch | `GOIBMI_CLOSEMASK=0` turns it off (diagnosis only) |

### Another thread changes the descriptor table while a child is created ([issue 2](https://github.com/JasonTashtego/pase_go_issues/blob/main/02-loader-failure-concurrent-fd-changes.md))

| |                                                                                                                                                                                                                                                  |
|---|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Symptom | the child fails with `Could not load program ...: System error - error data is: -1 9 (64002005)` (`EBADF`)                                                                                                                                       |
| Cause | PASE starts each child as a separate job. If another thread of the parent creates, duplicates or closes a descriptor meanwhile, the child's load fails.                                                                                          |
| C reproducer | eight threads starting a large executable: about 15% of `fork` + `exec` starts and about 30% of `posix_spawn` starts fail. None fail with a process-wide read/write lock around every descriptor call and the fork. A single thread never fails. See [issue 2](https://github.com/JasonTashtego/pase_go_issues/blob/main/02-loader-failure-concurrent-fd-changes.md). |
| Fix | `syscall/exec_aix_fdlock.go`: a process-wide lock taken for reading by `open`, `openat`, `close`, `dup`, `dup2`, `pipe`, `socket`, `socketpair`, `accept` and `fcntl(F_DUPFD)`, and for writing by `forkExec` while the child is created         |
| Limitation | a blocking `Open` of a FIFO with no writer, or a blocking `syscall.Accept`, holds the read lock and stalls any fork until it returns. Go's own sockets are non-blocking.  Only code that calls `syscall.Accept` directly on a blocking socket you created yourself would hit this, for example a program using raw syscall.Socket without SetNonblock.                                                                        |
| Switch | `GOIBMI_FDLOCK=0` turns it off (diagnosis only)                                                                                                                                                                                                  |

### `open` calls that run at the same time hand out unusable descriptors ([issue 9](https://github.com/JasonTashtego/pase_go_issues/blob/main/09-fresh-descriptor-dead-after-concurrent-open-close.md))

| | |
|---|---|
| Symptom | `go vet` fails about one package in a hundred with `vet: read <file>.s: bad file number`. The file was opened successfully and the next `read` returned `EBADF`. `fcntl(F_GETFL)` on the descriptor fails too, and a few `open` calls fail outright. |
| Cause | In a multi-threaded process, `open` calls that run at the same time can hand out unusable descriptors when they ask for the same lowest free slot. `go vet` closes descriptor 1 when it replaces `os.Stdout`, and its analyzer goroutines then open files concurrently, nearly all of them into slot 1. Likely a PASE defect (*unverified*). |
| C reproducer | four threads that `open`, `read` and `close` a file with descriptor 1 free (`pipe` on 1 and 2, then `close(1)`): about 2% of the reads fail with `EBADF` and about 0.15% of the `open` calls fail. With a mutex around `open` there are no failures in 2,400,000 opens; with a mutex around `close` the failures remain. With no free low descriptor, or with one thread, none occur. See [issue 9](https://github.com/JasonTashtego/pase_go_issues/blob/main/09-fresh-descriptor-dead-after-concurrent-open-close.md). |
| Fix | `syscall/open_aix.go`: `Open` takes a mutex around the `open` call. The open under the mutex carries `O_NONBLOCK`, so a FIFO cannot hold the mutex while waiting for the other end; the flag is cleared afterwards. A FIFO opened for reading, and an `open` that fails with `ENXIO` or `EAGAIN`, is repeated outside the mutex without `O_NONBLOCK`. |
| Limitation | FIFO opens are not serialized against other `open` calls, so they can still be hit. Each `open` costs two or three extra libc calls: a single-threaded open and close takes 37 microseconds instead of 20. |
| Measured | `go vet` of three packages, 100 runs: 0 failures with the fix, 19 to 29% of runs failing without it (on a build with timing diagnostics; without them about 1% of vet processes fail). A Go program with four goroutines and descriptor 1 free: 0 failures in 400,000 opens with the fix, 3,349 without. |
| Switch | `GOIBMI_OPENLOCK=0` turns it off (diagnosis only) |
