@echo off
echo Building GoDaddy DNS Updater (visible console version)...
go build -o godaddy-dns-updater.exe .
if %errorlevel% == 0 (
    echo Build successful! Created godaddy-dns-updater.exe
) else (
    echo Build failed!
)
pause
