# IBMi Go Support (1.26.9)

>>> **This repo is EXPERIMENTAL -- use at your own risk. No warranties expressed or implied. YMMV.**

## Overview

The [`iseries-1.26.9`](https://github.com/JasonTashtego/go/tree/iseries-1.26.9) branch of this repo is a fork of Go 1.26.9 with modifications that originated with [onlysumitg/ibmigo](https://github.com/onlysumitg/ibmigo), plus an existing 1.22.8 port found in this repository. Infinite thanks to everyone mentioned there who contributed to making it possible.

As noted in [golang/go#45017](https://github.com/golang/go/issues/45017), address space differences cause standard `aix/ppc64` executables to fail on IBMi.

This build adds an experimental flag:

```
GOEXPERIMENT=iseriesaix
```

The flag turns on the alternative address spacing so the compiler produces working executables for the IBMi AIX flavor (PASE), and compensates for other PASE environment differences.

If you're not familiar with building Go -- see [Installing Go from source](https://go.dev/doc/install/source)

You can use the [`iseries-1.26.9`](https://github.com/JasonTashtego/go/tree/iseries-1.26.9) branch to cross-compile pure Go code (no CGO) to IBMi from any supported Go platform. Check out the branch and build with the standard `make.bash`/`make.bat` (`all.bash`/`all.bat` also run the tests). If all you need is cross-compiling, set `GOEXPERIMENT=iseriesaix`, build pure-Go executables, and move them to the IBMi.  

For CGO support, you must bootstrap on a supported platform, then build natively on the IBMi with a working GCC (see [Native IBMi build](#native-ibmi-build)).

**Contents:** [Overview](#overview) · [What's Patched](#whats-patched) · [Getting the Source](#getting-the-source) · [Prerequisites](#prerequisites) · [Cross-Compiling from a Host Machine](#cross-compiling-from-a-host-machine) · [Native IBMi Build](#native-ibmi-build) · [Testing on IBMi](#testing-on-ibmi) · [Limitations](#limitations-of-running-go-code-on-ibmi)

## What's Patched

All changes are keyed off `GOEXPERIMENT=iseriesaix` (flag added in `src/internal/goexperiment`). Build tags `goexperiment.iseriesaix` select the IBMi variants of some `os` files.

The commit stream can be reviewed here [Commits](https://github.com/JasonTashtego/go/commits/iseries-1.26.9/?author=JasonTashtego)

### Address space and runtime

| File | Change                                                                                                                                                                                                                                                                                                                                                                                              |
| --- |-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `runtime/malloc.go`, `runtime/tagptr_64bit.go` | IBMi address space: the arena base and tagged pointers use `0x07...` instead of `0x0a...`. The randomized heap base that Go 1.26 uses on AIX now adds the arena base to its hints; before, the hints started at address 0, and one process start in about 65,536 got a hint of exactly 0 that was taken for a successful allocation ([golang/go#82006](https://github.com/golang/go/issues/82006)). |
| `runtime/os2_aix.go` | `setitimer(ITIMER_PROF)` fails on PASE; the error is ignored, so CPU profiling is not available, as it collects no samples.                                                                                                                                                                                                                                                                         |

### Process creation

| File | Change |
| --- | --- |
| `syscall/exec_aix_fdlock.go` (new), `exec_unix.go`, `zsyscall_aix_ppc64.go` | A process-wide descriptor lock: calls that create, duplicate or close a descriptor take it for reading, creating a child process takes it for writing. Without it, starting several processes at once fails with `Could not load program ... -1 9`. |
| `syscall/open_aix.go` (new), `syscall_aix.go`, `zsyscall_aix_ppc64.go` | `open` calls that run at the same time can hand out unusable descriptors (`read` fails with `EBADF`) when several threads ask for the same lowest free descriptor, as `go vet` does after it closes descriptor 1. `Open` therefore serializes the `open` call; a FIFO open is not held under the lock, so it can still wait for the other end. |
| `syscall/close_aix.go` (new), `runtime/closemask_aix.go` (new) | `close` runs with signals blocked on the calling thread. A signal that interrupts `close` of a pipe's write end otherwise leaves a writer that never goes away, and a reader never sees end-of-file. |
| `cmd/go/internal/toolchain/exec.go`, `cmd/link/internal/ld/execarchive.go` | `syscall.Exec` cannot work in a multi-threaded process on PASE (`execve` returns `EPERM`), so the `go` toolchain switch starts the new toolchain as a subprocess and `cmd/link` skips exec-archive. |

### Files and directories

| File | Change |
| --- | --- |
| `os/removeall_*.go`, `os/root_*.go`, `os/root_iseries_aix.go` (new), `os/root.go` | PASE's `*at` calls fail with `EINVAL` for a directory descriptor, so IBMi uses the path-based `RemoveAll` and a name-based `os.Root` (as `js` does). That `os.Root` is not safe against directories being renamed or replaced while it is in use, and it does not follow a root across renames. |
| `os/stat_unix.go`, `os/statat_unix.go` | `Lstat` of a path with a trailing slash follows the symlink; `Lstat` fallback for `File.ReadDir`. |

### Network, users and certificates

| File | Change |
| --- | --- |
| `syscall/syscall_aix.go` | `Accept` on a unix socket with an unnamed peer returns an empty `SockaddrUnix`. |
| `net/interface_aix.go` | `net.Interfaces` and `net.InterfaceAddrs` work. The hardware address and multicast list are empty. |
| `os/user`, `internal/syscall/unix/getgrset_aix.go` (new) | `User.GroupIds` uses libc's `getgrset`; PASE has no `/etc/group`. |
| `crypto/x509` | `TRUSTED CERTIFICATE` PEM blocks are accepted as `CERTIFICATE`; trailing data after a certificate is tolerated. |

### Build tools

| File | Change |
| --- | --- |
| `cmd/compile/internal/noder/lex.go` | Skips the AIX `lib.a/object.o` check on `#cgo` library patterns. |
| `cmd/dist/build.go` | Keeps the experiment on for toolchain1 and `go_bootstrap`. |

Test results for the native IBMi build, with the PASE limits behind the remaining failures, are in [IBMI-TEST-RESULTS.md](IBMI-TEST-RESULTS.md).

## Getting the Source

```
git clone https://github.com/JasonTashtego/go.git
cd go
git checkout iseries-1.26.9
```

## Prerequisites

- **A bootstrap Go:** Go 1.24.6 or newer (1.26.x is recommended) on your `PATH`. `make.bash`/`make.bat` finds it automatically, so you don't need to set `GOROOT_BOOTSTRAP`.
- **C compiler:** Not needed for bootstrap kit or cross-compile. A Windows host builds with the default `CGO_ENABLED=0` and doesn't use gcc. On Linux, you can set `CGO_ENABLED=0`; you must bootstrap on a supported platform and then build on IBMi to enable CGO support.

## Cross-Compiling from a Host Machine

Build the toolchain with the experimental flag set, then cross-build the standard library as a validation check.

### Linux / macOS (bash)

Build the toolchain with the experiment:

```bash
cd src
export GOROOT_BOOTSTRAP=/usr/local/go     # skip if go is already on PATH, or set to your go executable
export GOEXPERIMENT=iseriesaix
./make.bash
```

`make.bash` builds the toolchain but skips the test suite; `all.bash` also runs the tests. The result goes in `../bin` and `../pkg`.

Cross-compile for IBMi:

```bash
export GOROOT=$PWD/..
export PATH=$GOROOT/bin:$PATH
export GOOS=aix GOARCH=ppc64 CGO_ENABLED=0 GOEXPERIMENT=iseriesaix
go build std
```

Install it somewhere normal if needed. The toolchain is relocatable, so copy it without `.git`:

```bash
sudo rsync -a --exclude .git ../ /usr/local/go-iseries/
export PATH=/usr/local/go-iseries/bin:$PATH
```

### Windows

```powershell
cd <root>\go\src
$env:GOEXPERIMENT = "iseriesaix"
.\make.bat
```

This builds the toolchain with the new experiment into `..\bin` and `..\pkg`. Then do the cross-build check (set `GOEXPERIMENT` again if this is a new shell):

```powershell
$env:GOEXPERIMENT = "iseriesaix"
$env:GOROOT = "<root>\go"
$env:PATH = "$env:GOROOT\bin;$env:PATH"
$env:GOOS = "aix"; $env:GOARCH = "ppc64"; $env:CGO_ENABLED = "0"
go build std
```

Install it somewhere normal if needed. The toolchain is relocatable, so copy the tree without `.git`, then put the new `bin` first on your `PATH`:

```powershell
robocopy <root>\go C:\go\1.26.9-iseries /E /XD .git
$env:PATH = "C:\go\1.26.9-iseries\bin;$env:PATH"
go version
```

`robocopy` exit codes below 8 mean success. To make the change permanent, update the Go entry in your user `PATH`.



### Using the Result

If you do not need CGO support, you are **done**. Any supported Go platform can cross-compile Go executables for IBMi. Running the `go version` should report the experimental flag is on.

``` bash
$ go version
go version go1.26.9-X:iseriesaix
```

## Native IBMi Build

To build a toolchain that runs on the IBMi itself, first cross-compile a bootstrap kit from a host machine, then use it to build Go on the IBMi. The bootstrap kit is used read-only as the compiler. For CGO support you must install GCC on the IBMi (step 3).

**1. Create the bootstrap kit** (from Linux/macOS or WSL2 on Windows):

```bash
cd ./go/src
GOOS=aix GOARCH=ppc64 GOEXPERIMENT=iseriesaix ./bootstrap.bash
```

This produces a `go-aix-ppc64-bootstrap` directory.

**2. Transfer and extract the kit on the IBMi:**

```bash
# on the host
tar czf go-aix-ppc64-bootstrap.tgz go-aix-ppc64-bootstrap
# transfer to the IBMi, then on the IBMi:

# unzip the on the IBMi
gunzip -c go-aix-ppc64-bootstrap.tgz | tar xf -
export GOROOT_BOOTSTRAP=$PWD/go-aix-ppc64-bootstrap
```

**3. Get the source and set up the environment.** Use a separate checkout; don't build inside the bootstrap directory. On the IBMi you need either `git` or a copy of the repo tree transferred over.

```bash
git clone -b iseries-1.26.9 https://github.com/JasonTashtego/go.git   # or transfer a new copy
```

For CGO support, install GCC:

```bash
yum install gcc-12
```

Link the SSL root CAs from PASE to where Go (aix) looks for them:

```bash
ln -s /QOpenSys/etc/ssl /var/ssl
```

**4. Build** from `src` (CGO instructions expect that you have gcc-12 installed, if not set `CGO_ENABLED=0`):

```bash
cd go/src
export CC=/QOpenSys/pkgs/bin/gcc-12
export GOEXPERIMENT=iseriesaix
export GOROOT_BOOTSTRAP=~/go-aix-ppc64-bootstrap
export CGO_ENABLED=1
export GO_TEST_TIMEOUT_SCALE=3  #allow more time for long running tests.

./all.bash        # make.bash, then the test suite
```

`make.bash` alone builds without testing; `all.bash` runs it and then the tests. The output goes in `go/bin` and `go/pkg`. Use `-v` or `GOFLAGS` for more detail if you need it.

**5. Install.** There is no `make install`; the built tree is the install:

```bash
export PATH=$PWD/../bin:$PATH     # add <checkout>/bin to PATH
go version
```

## Testing on IBMi

Run the suite with `./all.bash`  (Use `GO_TEST_TIMEOUT_SCALE=3` to avoid package timeouts on a busy machine). Details are in [IBMI-TEST-RESULTS.md](IBMI-TEST-RESULTS.md).

| Category | What to expect                                                                                                                                                                            |
| --- |-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Environment | A profile with `*ALLOBJ` special authority can write read-only files, so ten permission tests will fail. Run the suite as an ordinary profile.   |
| PASE limits | Tests that depend on behaviour PASE lacks (very deep directories, some error codes, `getaddrinfo` strictness, `cgo -dynimport` symbol names) are skipped or relaxed under the experiment. |
| Intermittent | `os` `TestUNIXProcessAlive` sometimes fails with `os: process already finished`: it sends a signal to a child that has already exited.   |

## Limitations of running Go code on IBMi

`syscall.Exec` cannot work in a multi-threaded process on PASE (`execve` returns `EPERM`). Start a subprocess instead.

Three environment variables turn parts of the PASE support off, for diagnosis only: `GOIBMI_FDLOCK=0` (the descriptor lock; `Could not load program` then returns), `GOIBMI_CLOSEMASK=0` (the signal blocking around `close`) and `GOIBMI_OPENLOCK=0` (the serialized `open`; `go vet` then fails about one package in a hundred with `read: bad file number`).

Opening a FIFO is not serialized against other `open` calls, so a FIFO open can still be hit by the `open` problem described in [IBMI-TEST-RESULTS.md](IBMI-TEST-RESULTS.md#process-creation-and-open-fixes-for-pase).


## This repo remains EXPERIMENTAL -- use at your own risk. No warranties expressed or implied. YMMV.

---


# The Go Programming Language

Go is an open source programming language that makes it easy to build simple,
reliable, and efficient software.

![Gopher image](https://golang.org/doc/gopher/fiveyears.jpg)
*Gopher image by [Renee French][rf], licensed under [Creative Commons 4.0 Attribution license][cc4-by].*

Our canonical Git repository is located at https://go.googlesource.com/go.
There is a mirror of the repository at https://github.com/golang/go.

Unless otherwise noted, the Go source files are distributed under the
BSD-style license found in the LICENSE file.

### Download and Install

#### Binary Distributions

Official binary distributions are available at https://go.dev/dl/.

After downloading a binary release, visit https://go.dev/doc/install
for installation instructions.

#### Install From Source

If a binary distribution is not available for your combination of
operating system and architecture, visit
https://go.dev/doc/install/source
for source installation instructions.

### Contributing

Go is the work of thousands of contributors. We appreciate your help!

To contribute, please read the contribution guidelines at https://go.dev/doc/contribute.

Note that the Go project uses the issue tracker for bug reports and
proposals only. See https://go.dev/wiki/Questions for a list of
places to ask questions about the Go language.

[rf]: https://reneefrench.blogspot.com/
[cc4-by]: https://creativecommons.org/licenses/by/4.0/
