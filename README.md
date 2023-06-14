# MT-SICS
A (partial) implementation of the Mettler Toledo MT-SICS interface commands in Go

See [pkg.go.dev](https://pkg.go.dev/github.com/bas-dehaan/MT-SICS) for the full documentation

## Installation
To install MT-SICS, you need to have Go installed on your system. Make sure you have at least Go version 1.16 or later.

To install the package, use the `go get` command:
```bash
go get github.com/bas-dehaan/MT-SICS
```
This command will download and install the package and its dependencies in your Go workspace.

## Usage
To use MT-SICS in your Go program, import the package in your source code:
```go
import "github.com/bas-dehaan/MT-SICS"
```
Then, you can use the package's functions and types in your code.

Here's a simple example of how to use MT-SICS to communicate with a Mettler Toledo SICS-capable device:
```go
package main

import (
	"fmt"
	"github.com/bas-dehaan/MT-SICS"
)
  
func main() {  
	// Connect to the scale via COM1  
	connection, err := MT_SICS.Connect("COM1")  
	defer connection.Close() // Close the connection when the program ends
	if err != nil {
		panic(err)
	}

	// Get the scale out of standby
	err = MT_SICS.PowerOn(connection)
	if err != nil {
		panic(err)
	}

	// Weigh a sample
	measurement, err := MT_SICS.Weight(connection)
	if err != nil {
		panic(err)
	}

	// Print the measurement
	fmt.Println(measurement)
}
```
Make sure to replace the port in the `Connect` call with the actual port of your device. Keep in mind that UNIX-based systems (e.g. Linux and macOS) require a full device path, like `/dev/ttyCOM1`.

## Contributing
Contributions to MT-SICS are welcome! If you find any issues or have suggestions for improvements, please create an issue on the [GitHub repository](https://github.com/bas-dehaan/MT-SICS). Pull requests are also appreciated.
