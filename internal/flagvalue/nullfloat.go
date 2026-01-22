package flagvalue

import (
	"database/sql"
	"errors"
	"strconv"
)

var (
	ErrInvalidMinRelDistValue = errors.New("value must be between 0.0 and 0.99")
)

// NullFloat64 is a custom flag.Value for sql.Null[float64].
type NullFloat64 struct {
	Nullable sql.Null[float64]
}

func (n *NullFloat64) String() string {
	if !n.Nullable.Valid {
		return "unset"
	}
	return strconv.FormatFloat(n.Nullable.V, 'f', -1, 64)
}

func (n *NullFloat64) Set(value string) error {
	parsedFloat, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	if parsedFloat < 0.0 || parsedFloat > 0.99 {
		return ErrInvalidMinRelDistValue
	}

	n.Nullable.V = parsedFloat
	n.Nullable.Valid = true
	return nil
}
