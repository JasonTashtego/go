## Experimental IBMi Go support (1.22.8)
#### This repo is EXPERIMENTAL -- use at your own risk. No warranties expressed or implied. YMMV

This repo contains a fork of Go 1.22.8 with modifications merged from [https://github.com/onlysumitg/ibmigo]. 
Infinite thanks to everyone mentioned there who contributed to making it possible.
As noted in this open issue: [https://github.com/golang/go/issues/45017] address space differences cause standard aix/ppc64 built executables to fail on IBMi. 

This build of Go adds the expermimental flag:
```
GOEXPERIMENT=iseriesaix
```
The flag turns on the alternative address spacing to allow the compiler to produce working executables for the IBMi AIX flavor -- as well as compensating for other IBMi PASE environment differences.
This repo can be used to cross-compile pure Go code (no cgo) to IBMi from supported Go platforms. Checkout the branch and use the standard (all.sh or all.bat) to build a working Go toolkit. See [https://go.dev/doc/install/source] if you're not familiar with building Go.

### Bootstrapping a native IBMi Go

Build this source tree as you normally would on a supported platform to produce a Go that supports the new experimental flag. Then to build the Go bootstrap set the following:

```
export GOEXPERIMENT=iseriesaix  [windows: $env:GOEXPERIMENT="iseriesaix"]
export GOOS=aix                 [windows: $env:GOOS="aix"]
export GOARCH=ppc64             [windows: $env:GOARCH="ppc64"]
```

Build again using bootstrap:

```
~/go1.22.8/go/src $ ./bootstrap.bash
```

For windows -- There isn't a bootstrap.bat. However, all.bat will work just as well. It builds the kit and fails out when it reaches the testing phase as it cannot run the IBMi executables.

The cross-compiled tree can then be copied to the ISeries to be used as a Go bootstrap toolkit for building Go on the ISeries.

```
  Transfer bootstrap tree to IBMi.
  Using this as the directory you transferred to: ~/go-boot1.22.8

  chmod to make files executable:
    cd ~/go-boot1.22.8/bin/aix_ppc64 && chmod +x *
    cd ~/go-boot1.22.8/pkg/tool/aix_ppc64 && chmod +x *

  Move platform specific (IBMi) go executables to bin:
     mv ~/go-boot1.22.8/bin/aix_ppc64/* ~/go-boot1.22.8/bin

  Because the bootstrap process ignores GOEXPERIMENT you'll need to force ON [GOEXPERIMENT = iseriesaix] in source:
    Edit: 
      /src/internal/goexperiment/exp_iseriesaix_off.go
       Line 7: const ISeriesAix = true
       Line 8: const ISeriesAixInt = 1

  Duplicate the entire tree into a 2nd directory (eg.  ~/go.1.22.8). This will be where the native build occurs.
    [~]$ cp -R ~/go-boot1.22.8  ~/go-1.22.8

  Create a temp dir for go bootstrap scratch work:
    [~]$ mkdir ~/tmp

  If you want CGO support, install GCC:
    [~]$ yum install gcc-12

  Link the SSL root CAs from PASE to where Go (aix) looks for them:
    [~]$ ln -s /QOpenSys/etc/ssl /var/ssl

  Set environment variables (skip the first 2 if you don't have GCC installed):
    [~/go-1.22.8/src]$ export CC=/QOpenSys/pkgs/bin/gcc-12
    [~/go-1.22.8/src]$ export CGO_ENABLED=1
    [~/go-1.22.8/src]$ export GOMAXPROCS=1
    [~/go-1.22.8/src]$ export GOROOT_BOOTSTRAP=~/go-boot1.22.8
    [~/go-1.22.8/src]$ export GOTMPDIR=~/tmp

  Build Go with the bootstrap kit:  (example output below)
    [~/go-1.22.8/src]$  chmod +x all.bash
    [~/go-1.22.8/src]$  ./all.bash 
    Building Go cmd/dist using /home/JASOND/go-boot1.22.8. (go1.22.8 X:iseriesaix aix/ppc64)
    Building Go toolchain1 using /home/JASOND/go-boot1.22.8.
    Building Go bootstrap cmd/go (go_bootstrap) using Go toolchain1.
    Building Go toolchain2 using go_bootstrap and Go toolchain1.
    Building Go toolchain3 using go_bootstrap and Go toolchain2.
    Building packages and commands for aix/ppc64.

    ##### Test execution environment.
    # GOARCH: ppc64
    # CPU: POWER10
    # GOOS: aix
    # OS Version: OS400 5 7 00780005CAB1

    ##### Testing packages.
    ok      archive/tar     0.213s
    ok      archive/zip     0.205s
    ok      bufio   0.108s
    ok      bytes   0.303s
    ok      cmp     0.062s
    ...

    Note: There are some test failures
      - IFS related (symlinks, permissions)
      - Invalid tests that look for standard aix files in their normal locations -- which do not exist on the PASE environment.
      - profiling tests fail due to setitimer failing.  

```

When running the Go command ( ``` go build ``` etc)  in the PASE environment ensure you have:
```
GOMAXPROCS=1
```
Otherwise you may see the following error:
```
System error - error data is: -1 9 (64002005)
```

#### This repo is EXPERIMENTAL -- use at your own risk. No warranties expressed or implied. YMMV

Changes are merged to:
[https://github.com/JasonTashtego/go/tree/release-branch.go1.22]


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
