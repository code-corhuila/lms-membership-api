package membership

import "context"

// ActiveLoansChecker is a driven port for asking circulation-service whether a
// student currently has active loans. Membership cannot query the `loans`
// table directly anymore — it lives in circulation-service's own database
// (library-docs/09-microservices/service-boundary-rules.md) — so this is
// implemented by an HTTP adapter, not a repository.
type ActiveLoansChecker interface {
	CountActive(ctx context.Context, studentID string) (int, error)
}
