package store

import "errors"

var ErrStackExists = errors.New("stack already exists")
var ErrStackNotFound = errors.New("stack not found")
