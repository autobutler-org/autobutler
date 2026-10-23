package v0_photos

import "errors"

// errNoAccess is what a caller hears about a photo they may not see. It reads
// the same as a photo that does not exist, because to them it does not.
var errNoAccess = errors.New("photo not found")

// errReadOnly is what a caller hears when they may see a photo but not change
// it or write beside it.
var errReadOnly = errors.New("you do not have permission to change this")
