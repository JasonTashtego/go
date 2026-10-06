## Experimental IBMi Go support (1.26.8)
#### This repo is EXPERIMENTAL -- use at your own risk. No warranties expressed or implied. YMMV

This repo contains a fork of Go 1.26.8 with modifications merged from [https://github.com/onlysumitg/ibmigo], and a 1.22.8 port found in this repository.
Infinite thanks to everyone mentioned there who contributed to making it possible.
As noted in this open issue: [https://github.com/golang/go/issues/45017] address space differences cause standard aix/ppc64 built executables to fail on IBMi.

This build of Go adds the experimental flag:
```
GOEXPERIMENT=iseriesaix
```
The flag turns on the alternative address spacing to allow the compiler to produce working executables for the IBMi AIX flavor -- as well as compensating for other IBMi PASE environment differences.
This repo can be used to cross-compile pure Go code (no cgo) to IBMi from supported Go platforms. Checkout the branch and use the standard (make.bash or make.bat; all.bash or all.bat also run the tests) to build a working Go toolkit. See [https://go.dev/doc/install/source] if you're not familiar with building Go.  If all you need to do is cross-compile, then you're done. You can set ```GOEXPERIMENT=iseriesaix``` and build working pure-go executables and then move them to the iseries. For CGO support bootstrapping on the IBMi with a working GCC is necessary.

#### What's patched
All changes are keyed off `GOEXPERIMENT=iseriesaix` (flag added in `src/internal/goexperiment`), except the `os` file split noted below.

* `runtime/malloc.go` -- arena base offset and address hints moved to the IBMi range (`0x07...` instead of `0x0a...`).
* `runtime/tagptr_64bit.go` -- tagged-pointer high bits use `0x7` instead of `0xa` on aix.
* `os/removeall_at_aix.go` (new; `removeall_at.go` now excludes aix) -- RemoveAll walks and removes by path, because PASE lacks a working `unlinkat`.
* `crypto/x509/cert_pool.go` -- `TRUSTED CERTIFICATE` PEM blocks are accepted as `CERTIFICATE` when loading the PASE root CAs.
* `crypto/x509/parser.go` -- trailing data after a certificate is tolerated.
* `cmd/compile/internal/noder/lex.go` -- skips the aix `lib.a/object.o` check on `#cgo` library patterns.

#### Clone and check out the branch
```
git clone https://github.com/JasonTashtego/go.git
cd go
git checkout iseries-1.26
```

#### Steps to build, after cloning repository:

Bootstrap Go: Use Go 1.24.6 or newer (1.26.x is fine) and make sure it's on your PATH. make.bash/make.bat finds it automatically, so you don't need to set GOROOT_BOOTSTRAP.
C compiler: You don't need one. For Windows host build with the default CGO_ENABLED=0 for the cross-compile doesn't use gcc. If you're on linux set CGO_ENABLED=0 as you'd have to build on iSeries to get support.

#### Build the toolchain with the experimental flag set

#### Windows:

```
cd <root>\go\src
$env:GOEXPERIMENT = "iseriesaix"
.\make.bat
```

That builds the toolchain with the new experiment into ..\bin and ..\pkg. Then do the cross-build check (set GOEXPERIMENT again if this is a new shell):

```
$env:GOEXPERIMENT = "iseriesaix"
$env:GOROOT = "<root>\go"
$env:PATH = "$env:GOROOT\bin;$env:PATH"
$env:GOOS = "aix"; $env:GOARCH = "ppc64"; $env:CGO_ENABLED = "0"
go build std
```

Install it somewhere normal. The toolchain is relocatable, so copy the tree without .git, then put the new bin first on your PATH:
```
robocopy <root>\go C:\go\1.26.8-iseries /E /XD .git
$env:PATH = "C:\go\1.26.8-iseries\bin;$env:PATH"
go version
```
(robocopy exit codes below 8 mean success.) To make it permanent, update the Go entry in your user PATH.


#### *nix/bash:

Build the toolchain with the experiment
```
cd src
export GOROOT_BOOTSTRAP=/usr/local/go     # skip if go is already on PATH
export GOEXPERIMENT=iseriesaix
./make.bash
```

make.bash builds the toolchain but skips the test suite. all.bash is the one that also runs the tests. The result goes in ../bin and ../pkg.

Cross-compile for IBM i
```
export GOROOT=$PWD/..
export PATH=$GOROOT/bin:$PATH
export GOOS=aix GOARCH=ppc64 CGO_ENABLED=0
go build std
```

Install it somewhere normal. The toolchain is relocatable, so copy it without .git:
```
sudo rsync -a --exclude .git ../ /usr/local/go-iseries/
export PATH=/usr/local/go-iseries/bin:$PATH
```

### Building for IBMi

If you do not need CGO support then you are done. You can use any supported Go platform to cross compile Go executables for iSeries.


### Bootstrapping a native IBMi Go

To build the toolchain to run on the IBM i itself, set GOOS=aix GOARCH=ppc64 GOEXPERIMENT=iseriesaix and use ./bootstrap.bash from src. It produces a go-aix-ppc64-bootstrap directory and tarball that
you copy to the IBM i. Note: For CGO support you must install GCC on the iSeries, covered below.


The cross-compiled tree can then be copied to the ISeries to be used as a Go bootstrap toolkit for building Go on the ISeries.

```
  Transfer bootstrap tree or tarball to IBMi.
  
  Using this as the directory you transferred to: ~/go-boot1.26.8

  chmod to make files executable:
    cd ~/go-boot1.26.8/bin/aix_ppc64 && chmod +x *
    cd ~/go-boot1.26.8/pkg/tool/aix_ppc64 && chmod +x *

  Move platform specific (IBMi) go executables to bin:
     mv ~/go-boot1.26.8/bin/aix_ppc64/* ~/go-boot1.26.8/bin

  Because the bootstrap process ignores GOEXPERIMENT you'll need to force ON [GOEXPERIMENT = iseriesaix] in source:
    Edit: 
      /src/internal/goexperiment/exp_iseriesaix_off.go
       Line 7: const ISeriesAix = true
       Line 8: const ISeriesAixInt = 1

  Duplicate the entire tree into a 2nd directory (eg.  ~/go.1.26.8). This will be where the native build occurs.
    [~]$ cp -R ~/go-boot1.26.8  ~/go-1.26.8

  Create a temp dir for go bootstrap scratch work:
    [~]$ mkdir ~/tmp

  If you want CGO support, install GCC:
    [~]$ yum install gcc-12

  Link the SSL root CAs from PASE to where Go (aix) looks for them:
    [~]$ ln -s /QOpenSys/etc/ssl /var/ssl

  Set environment variables (skip the first 2 if you don't have GCC installed):
    [~/go-1.26.8/src]$ export CC=/QOpenSys/pkgs/bin/gcc-12
    [~/go-1.26.8/src]$ export CGO_ENABLED=1
    [~/go-1.26.8/src]$ export GOMAXPROCS=1
    [~/go-1.26.8/src]$ export GOEXPERIMENT=iseriesaix 
    [~/go-1.26.8/src]$ export GOROOT_BOOTSTRAP=~/go-boot1.26.8
    [~/go-1.26.8/src]$ export GOTMPDIR=~/tmp

  Build Go with the bootstrap kit:  (example output below)
    [~/go-1.26.8/src]$  chmod +x all.bash
    [~/go-1.26.8/src]$  ./all.bash 
    Building Go cmd/dist using /home/JASOND/go-boot1.26.8. (go1.26.8 X:iseriesaix aix/ppc64)
    Building Go toolchain1 using /home/JASOND/go-boot1.26.8.
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
