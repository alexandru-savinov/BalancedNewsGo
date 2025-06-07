#!/bin/bash
cd /d/Dev/newbalancer_go
go run cmd/server/main.go cmd/server/template_handlers.go 2>&1 | tee server_debug.log
