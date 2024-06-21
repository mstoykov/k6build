package k6build

import "errors"

var (
	ErrAccessingObject   = errors.New("accessing object")      //nolint:revive
	ErrAccessingServer   = errors.New("making request")        //nolint:revive
	ErrBuildFailed       = errors.New("build failed")          //nolint:revive
	ErrCreatingObject    = errors.New("creating object")       //nolint:revive
	ErrInitializingCache = errors.New("initializing cache")    //nolint:revive
	ErrInvalidConfig     = errors.New("invalid configuration") //nolint:revive
	ErrInvalidResponse   = errors.New("invalid response")      //nolint:revive
	ErrInvalidURL        = errors.New("invalid object URL")    //nolint:revive
	ErrObjectNotFound    = errors.New("object not found")      //nolint:revive
	ErrRequestFailed     = errors.New("request failed")        //nolint:revive
)
