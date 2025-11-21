@echo off
echo Building probe.exe...
go build -o ./dist/probe.exe main.go
if %errorlevel% equ 0 (
    echo Build successful!
) else (
    echo Build failed!
)
