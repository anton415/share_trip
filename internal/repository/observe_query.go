package repo

import "time"

func (r *PostgresTripRepository) observeQuery(
	operation string,
	result string,
	started time.Time,
) {
	r.metrics.RepositoryQueryTotal.
		WithLabelValues(operation, result).
		Inc()

	r.metrics.RepositoryQueryDuration.
		WithLabelValues(operation, result).
		Observe(time.Since(started).Seconds())
}
