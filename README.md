# yflicks-yts

`yflicks-yts` is a client library for [YTS](https://yts.mx), it is developed keeping
the needs of the [yflicks](https://github.com/atifcppprogrammer/yflicks) desktop
application in mind, but you can leverage this package in your own project as well
👍. An example program can be found [here](./example/main.go) demonstrating how to 
use this package.

## Installation
```
go get github.com/atifcppprogrammer/yflicks-yts@latest
```

## Development Setup
For working on this project, please ensure that your machine is provisioned with the
following.

1. [GNU Make](https://www.gnu.org/software/make/)
2. Golang version __`1.21.5`__

We recommend using the [asdf](https://github.com/asdf-vm/asdf) version manager and
the corresponding Golang [plugin](https://github.com/asdf-community/asdf-golang) for
managing your installation 👍.

Once you have cloned this repository, please run __`make`__ in the root of this
repository. This will among other things perform the following _important_ tasks.

1. Install [golangci-lint](https://github.com/golangci/golangci-lint)
2. Setup necessry git hooks to ensure code quality.
3. Download all dependencies for this package.

