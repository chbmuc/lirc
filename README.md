LIRC
====

Go client for the infamous Linux Infrared Remote Control ([LIRC](http://www.lirc.org)) package

#### Usage

```go
package main

import (
  "github.com/chbmuc/lirc"
  "log"
)

func keyPower(event lirc.LircEvent) {
  log.Println("Power Key Pressed")
}

func keyTV(event lirc.LircEvent) {
  log.Println("TV Key Pressed")
}

func keyAll(event lirc.LircEvent) {
  log.Println(event)
}

func main() {
  // Initialize with path to lirc socket
  ir, err := lirc.Init("/var/run/lirc/lircd")
  if err != nil {
    panic(err)
  }

  // Receive Commands

  // attach key press handlers
  ir.Handle("", "KEY_POWER", keyPower)
  ir.Handle("", "KEY_TV", keyTV)
  ir.Handle("", "", keyAll)

  // run the receive service
  go ir.Run()

  // make sure here's some blocking code
}
```
