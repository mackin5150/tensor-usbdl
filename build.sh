#!/bin/bash
echo "==================================="
echo "Building Tensor-USBDL with GS101 Support"
echo "Based on keyholes.txt endpoint analysis"
echo "==================================="
echo

echo "[1/4] Cleaning old builds..."
rm -f tensor-usbdl-gs101 tensor-usbdl-gs101-debug

echo "[2/4] Downloading Go dependencies..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to download dependencies"
    exit 1
fi

echo "[3/4] Building release version..."
go build -ldflags "-s -w" -o tensor-usbdl-gs101 main.go
if [ $? -ne 0 ]; then
    echo "ERROR: Release build failed"
    exit 1
fi

echo "[4/4] Building debug version..."
go build -o tensor-usbdl-gs101-debug main.go
if [ $? -ne 0 ]; then
    echo "ERROR: Debug build failed"
    exit 1
fi

echo
echo "==================================="
echo "✅ BUILD SUCCESSFUL!"
echo "==================================="
echo
echo "Files created:"
echo "  - tensor-usbdl-gs101       (Release - optimized)"
echo "  - tensor-usbdl-gs101-debug (Debug - with symbols)"
echo
echo "Usage examples:"
echo "  ./tensor-usbdl-gs101 detect"
echo "  ./tensor-usbdl-gs101 test"
echo "  ./tensor-usbdl-gs101 flash ../gs101/pbl.img"
echo "  ./tensor-usbdl-gs101 flash ../gs101/pbl.img usb"
echo
echo "Ready for Pixel 6a unbrick attempt!"
