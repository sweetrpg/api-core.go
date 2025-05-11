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

func HealthHandler(c context.Context) vo.HealthResponseVO {
	var messages []string
	errorCount := 0

	_, span := otel.Tracer("health").Start(c, "list-collections")
	collections, err := database.Db.ListCollectionNames(context.TODO(), bson.D{})
	span.End()
	if err != nil {
		messages = append(messages, err.Error())
		errorCount += 1
	}

	start := time.Now()
	_, span = otel.Tracer("health").Start(c, "ping-database")
	err = database.Db.Client().Ping(c, readpref.Primary())
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
