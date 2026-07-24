package server

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sweetrpg/api-core.go/vo"
	"github.com/sweetrpg/mongodb.go/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/otel"
)

// healthCheckTimeout bounds each dependency check below so a stalled Mongo
// operation fails the health check instead of hanging past the caller's
// (e.g. a Kubernetes readiness probe's) own timeout indefinitely.
const healthCheckTimeout = 5 * time.Second

func HealthHandler(c context.Context) vo.HealthResponseVO {
	var messages []string
	errorCount := 0

	listCtx, cancel := context.WithTimeout(c, healthCheckTimeout)
	defer cancel()
	_, span := otel.Tracer("health").Start(listCtx, "list-collections")
	collections, err := database.Db.ListCollectionNames(listCtx, bson.D{})
	span.End()
	if err != nil {
		messages = append(messages, err.Error())
		errorCount += 1
	}

	pingCtx, cancel := context.WithTimeout(c, healthCheckTimeout)
	defer cancel()
	start := time.Now()
	_, span = otel.Tracer("health").Start(pingCtx, "ping-database")
	err = database.Db.Client().Ping(pingCtx, readpref.Primary())
	span.End()
	duration := time.Since(start)
	if err != nil {
		messages = append(messages, err.Error())
		errorCount += 1
	}

	return vo.HealthResponseVO{
		Database:    database.Db.Name(),
		Ping:        fmt.Sprintf("%dms", duration.Milliseconds()),
		Collections: collections,
		Messages:    messages,
		Errors:      errorCount,
	}
}

func PingHandler() vo.PingResponseVO {
	hostname, _ := os.Hostname()
	return vo.PingResponseVO{
		Date:     time.Now(),
		Hostname: hostname,
	}
}
