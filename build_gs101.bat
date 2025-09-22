@echo off
echo ===================================
echo Building Tensor-USBDL with GS101 Support
echo Based on keyholes.txt endpoint analysis
echo ===================================
echo.

echo [1/4] Cleaning old builds...
if exist tensor-usbdl-gs101.exe del tensor-usbdl-gs101.exe
if exist tensor-usbdl-gs101-debug.exe del tensor-usbdl-gs101-debug.exe

echo [2/4] Downloading Go dependencies...
go mod tidy
if %errorlevel% neq 0 (
    echo ERROR: Failed to download dependencies
    pause
    exit /b 1
)

echo [3/4] Building release version...
go build -ldflags "-s -w" -o tensor-usbdl-gs101.exe main.go
if %errorlevel% neq 0 (
    echo ERROR: Release build failed
    pause
    exit /b 1
)

echo [4/4] Building debug version...  
go build -o tensor-usbdl-gs101-debug.exe main.go
if %errorlevel% neq 0 (
    echo ERROR: Debug build failed
    pause
    exit /b 1
)


echo.
echo ===================================
echo ✅ BUILD SUCCESSFUL!
echo ===================================
echo.
echo Files created:
echo   - tensor-usbdl-gs101.exe       (Release - optimized)
echo   - tensor-usbdl-gs101-debug.exe (Debug - with symbols)
echo.
echo Usage examples:
echo   tensor-usbdl-gs101.exe detect
echo   tensor-usbdl-gs101.exe test
echo   tensor-usbdl-gs101.exe flash ..\gs101\pbl.img
echo   tensor-usbdl-gs101.exe flash ..\gs101\pbl.img usb
echo.
echo Ready for Pixel 6a unbrick attempt!
pause
