package domain

import (
	"errors"
	"fmt"
)

var (
	ErrSystemDisk           = errors.New("target device is an active system disk or contains critical system mounts")
	ErrInsufficientCapacity = errors.New("target device does not have sufficient capacity for the image")
	ErrDeviceBusy           = errors.New("target device or one of its partitions is busy and cannot be unmounted")
	ErrDeviceRemoved        = errors.New("target device is no longer accessible or was disconnected")
	ErrIdentityMismatch     = errors.New("target device identity verification failed; device metadata changed")
	ErrPermissionDenied     = errors.New("administrative privileges required to access block device")
	ErrUnsupportedImage     = errors.New("image format or boot catalog is unsupported")
	ErrUnsupportedFS        = errors.New("requested filesystem is not supported on this platform")
	ErrVerificationFailed   = errors.New("post-write data verification failed; checksum mismatch")
	ErrCancelled            = errors.New("operation was cancelled by user or signal")
	ErrWriteFailed          = errors.New("failed to write data to block device")
	ErrValidationFailed     = errors.New("plan or device validation failed")
)

// OpError wraps an underlying error with contextual subsystem and target device details.
type OpError struct {
	Op     string
	Target string
	Err    error
}

func (e *OpError) Error() string {
	if e.Target != "" {
		return fmt.Sprintf("%s on %s: %v", e.Op, e.Target, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *OpError) Unwrap() error {
	return e.Err
}

func NewOpError(op, target string, err error) error {
	if err == nil {
		return nil
	}
	return &OpError{Op: op, Target: target, Err: err}
}
