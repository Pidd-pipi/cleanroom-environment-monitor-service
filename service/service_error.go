package service

import "example.com/cleanroom-environment-monitor-service/domain"

// wrapInfra converts an unexpected infrastructure error (persistence, I/O)
// into a domain internal error so the HTTP layer can classify it and the
// request log surfaces the real cause. Errors that already carry a domain
// code (not_found / conflict / invalid_input, ...) are returned unchanged:
// only the opaque lower-level failures are reclassified.
func wrapInfra(context string, err error) error {
	if err == nil {
		return nil
	}
	if domain.AsDomainError(err) != nil {
		return err
	}
	return domain.Internal(context, err)
}
