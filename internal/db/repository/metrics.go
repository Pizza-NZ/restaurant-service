package repository

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	dbQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "db_query_duration_seconds",
		Help: "Duration of database queries.",
	}, []string{"query"})
)
