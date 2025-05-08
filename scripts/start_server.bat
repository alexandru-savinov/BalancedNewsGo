@echo off
echo Cleaning and rebuilding...
go clean
go build -o newbalancer_server.exe cmd/server/main.go

echo Starting the server...
start /B newbalancer_server.exe 