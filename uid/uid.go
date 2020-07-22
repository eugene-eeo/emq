package uid

import "errors"
import "math/rand"
import "encoding/base64"
import "encoding/json"

var ErrInvalidLength = errors.New("invalid length")

var encoding = base64.RawStdEncoding

// Size of a UID in bytes
const Size = 16

type UID [Size]byte

// Generate generates a new UID
func Generate() UID {
	var buf [Size]byte
	rand.Read(buf[:])
	return buf
}

// Bytes returns the UID as a byte slice
func (u UID) Bytes() []byte {
	return u[:]
}

// String returns the UID as a string
func (u UID) String() string {
	return encoding.EncodeToString(u.Bytes())
}

func (u UID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u *UID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := FromString(s)
	if err != nil {
		return err
	}
	*u = v
	return nil
}

// FromBytes returns a UID from bytes
func FromBytes(b []byte) (UID, error) {
	u := UID{}
	if len(b) != Size {
		return u, ErrInvalidLength
	}
	copy(u[:], b)
	return u, nil
}

// FromString returns a UID from string
func FromString(s string) (UID, error) {
	b, err := encoding.DecodeString(s)
	if err != nil {
		return UID{}, err
	}
	return FromBytes(b)
}
