#!/bin/bash
cd ../../
swag init --g ./cmd/server/main.go --exclude config,docs,pkg,test,internal/event,internal/global,internal/middleware,internal/repository,internal/router,internal/service --o ./swagger