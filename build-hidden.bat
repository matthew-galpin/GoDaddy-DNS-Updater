@echo off
echo Building GoDaddy DNS Updater (hidden console version)...
go build -ldflags "-H windowsgui" -o godaddy-dns-updater-hidden.exe .
if %errorlevel% == 0 (
    echo Build successful! Created godaddy-dns-updater-hidden.exe
    echo This version runs without showing a console window.
) else (
    echo Build failed!
)
pause
