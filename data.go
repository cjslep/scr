package scr

import (
	"fmt"
)

// Data is content-addressed data.
//
// The actual implementation for how to map "content" => "address" doesn't
// matter to this routing simulation.
//
// Note we don't actually store the random data generated, as it's uninteresting
// and a waste of RAM.
type Data struct {
	Address  Address
	Location V
	DataSize int
}

func NewData(b []byte) *Data {
	d := &Data{DataSize: len(b)}
	d.Address = DataToAddress(b)
	d.Location = AddressToPosition(d.Address)
	return d
}

func (d Data) String() string {
	return fmt.Sprintf("%s@%s", d.Address, d.Location)
}
