package scr

import (
	"crypto/sha256"
	"encoding/base64"
)

// Address represents a location of data. It could be anything: IP addresses,
// content addressing, mailing addresses, etc.
type Address []byte

// ToString simply Base64 encodes the address, as this particular implementation
// is a hash of the data.
func (a Address) ToString() string {
	return base64.StdEncoding.EncodeToString(a)
}

// DataToAddress creates a content-address based on the data content.
//
// The first steps of handling data in the routing algorithm! It requires:
// - Create the content-address from the data. This is the data-identifier.
// - Turn the data-identifier into a position on a unit sphere. In our case,
//   the data identifier is a content-address (but it could be anything: IP
//   address, DNS name, mailing address).
//
// The actual underlying implementation is not significant to the routing
// protocol. For ease, we pick a SHA256 hash.
//
// A real implementation will need to concretely specify or convey this
// algorithm to peers.
func DataToAddress(b []byte) Address {
	hash := sha256.Sum256([]byte(b))
	return hash[:]
}

// AddressToPosition turns the data into a vector pointing to a location on a
// unit sphere.
//
// The actual underlying implementation is not significant to the routing
// protocol. For ease, we pick the unit vector in X direction, and repeatedly
// rotate it by a quaternion whose values are address bytes available.
//
// A real implementation will need to concretely specify or convey this
// algorithm to peers.
func AddressToPosition(a Address) V {
	v := V{X: 1}
	q := Q{}
	// idx tells us which entry in Q to fill, or to rotate out to a new Q.
	idx := 0
	for _, byt := range a {
		switch idx {
		case 0:
			q.I = float64(byt)
			idx++
		case 1:
			q.J = float64(byt)
			idx++
		case 2:
			q.K = float64(byt)
			idx++
		case 3:
			// Fill in final quaternion value, apply the quaternion,
			// reset idx.
			q.R = float64(byt)
			q = q.Unit()
			v = v.Rotate(q)
			q = Q{}
			idx = 0
		}
	}
	// If trailing values, apply the partial quaternion as-is.
	if idx != 0 {
		v = v.Rotate(q)
	}
	return v
}
