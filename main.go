package main

import (
	"context"
	"rsshub-auto-refresh/component/database"
	"rsshub-auto-refresh/component/http_client"
	"rsshub-auto-refresh/component/logger"
	"rsshub-auto-refresh/job"
)

func main() {
  var ctx context.Context = context.Background()
  logger.Init()
  database.InitDatabase(ctx)
  http_client.InitHttpClient(ctx)
  logger.Log().Info(ctx, "Start job")

  job.RegisterJob()
  select{}
}
