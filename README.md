### Experimental IBMi Go support (1.22.8)
This repo contains a fork of Go 1.22.8 with modifications merged from [https://github.com/onlysumitg/ibmigo]. 
Infinite thanks to everyone mentioned there who contributed to making it possible.
As noted in this open issue: [https://github.com/golang/go/issues/45017] address space differences cause standard aix/ppc64 built executables to fail on IBMi. 

This build of Go adds the expermimental flag:
```
GOEXPERIMENT=iseriesaix
```
The flag turns on the alternative address spacing to allow the compiler to produce working executables for the IBMi AIX flavor.

Additionally, if you're building directly on an iSeries using the PASE environment, then the ```go``` command relies on detecting this environment variable to change it's behavior to accomodate PASE and IFS quirks.
```
MACHTYPE=powerpc-ibm-os400
```
This repo can be used to cross-compile pure Go code (no cgo) to IBMi from supported Go platforms. 
To build with cgo support see the readme.md in this repository [https://github.com/onlysumitg/ibmigo] for building on the IBMi host itself.


Changes are merged to:

[https://github.com/JasonTashtego/go/tree/release-branch.go1.22]

Checkout the branch and use the standard (all.sh or all.bat) to build a working Go toolkit. See [https://go.dev/doc/install/source] if you're not familiar with building Go.

You can view the enabling commit here:
[https://github.com/golang/go/commit/da3e9e0598974fce745c9a18f55fad2c5fe89054]


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
