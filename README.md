[![Go Reference](https://pkg.go.dev/badge/github.com/n-r-w/uniqpool.svg)](https://pkg.go.dev/github.com/n-r-w/uniqpool)
[![Go Coverage](https://github.com/n-r-w/uniqpool/wiki/coverage.svg)](https://raw.githack.com/wiki/n-r-w/uniqpool/coverage.html)
![CI Status](https://github.com/n-r-w/uniqpool/actions/workflows/go.yml/badge.svg)
[![Stability](http://badges.github.io/stability-badges/dist/stable.svg)](http://github.com/badges/stability-badges)
[![Go Report](https://goreportcard.com/badge/github.com/n-r-w/uniqpool)](https://goreportcard.com/badge/github.com/n-r-w/uniqpool)

# UniqPool

UniqPool is a wrapper around the worker pool that excludes duplicate tasks.
All incoming tasks first go to the inbound queue, which is processed at a specified interval in FIFO order. If a task with the same identifier is already in the queue, the new task will be ignored.

It is useful when you need to process a large number of tasks, part of which can be duplicated.

Internal worker pool is based on <https://github.com/alitto/pond>.

## Installation

```bash
go get github.com/n-r-w/uniqpool
```

## Usage

Here is a basic example of how to use UniqPool:

```go
package main

import (
    "fmt"
    "time"

    "github.com/n-r-w/uniqpool"
)

func main() {
    // 10 - inbound queue capacity
    // 5 - workers count
    // 100 - worker pool capacity
    // time.Second - interval in milliseconds after which incoming tasks will be sent to the worker pool
    p := uniqpool.New[string](10, 5, 100, time.Second)

    p.Submit("task1", func() {
        fmt.Println("will be executed")
    })

    p.Submit("task2", func() {
        fmt.Println("will be executed")
    })

    p.Submit("task2", func() {
        fmt.Println("will not be executed, because the task with the same identifier has already in the queue")
    })

    p.StopAndWait()
}
```
