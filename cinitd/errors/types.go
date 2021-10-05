package errors

// ErrTypes for handling error cases
type ErrTypes int

const (
	// BInternal Backend Internal error
	BInternal ErrTypes = iota
	// UserExists Client Registration Error
	UserExists
	// Unauthorized Client login error
	Unauthorized
	// NotFound Client resource error
	NotFound
	// BadRequest Client request error
	BadRequest
)
