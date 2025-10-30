#!/bin/bash
# GUI Test Script for SDL3 Programs
# Run this on a system with a display

set -e

echo "=== SDL3 GUI Test Suite ==="
echo "Testing SDL3 programs with actual display..."
echo ""

# Compile test programs
echo "Compiling SDL3 test programs..."
./flapc testprograms/sdl3_window.flap -o sdl3_window
./flapc testprograms/sdl3_texture_demo.flap -o sdl3_texture_demo 2>/dev/null || echo "Note: sdl3_texture_demo may require additional setup"

echo ""
echo "=== Test 1: SDL3 Window Creation ==="
echo "This should create a window and close it after a few seconds..."
./sdl3_window
if [ $? -eq 0 ]; then
    echo "✓ SDL3 window test passed"
else
    echo "✗ SDL3 window test failed with exit code $?"
fi

echo ""
echo "=== Test 2: Simple SDL3 Init/Quit ==="
cat > /tmp/test_sdl3_init.flap << 'FLAP'
import sdl3 as sdl
printf("Initializing SDL with VIDEO...\n")
result := sdl.SDL_Init(sdl.SDL_INIT_VIDEO)
result == 1 {
    printf("✓ SDL initialized successfully\n")
} ~> {
    printf("✗ SDL initialization failed\n")
}
sdl.SDL_Quit()
printf("✓ SDL quit successfully\n")
FLAP

./flapc /tmp/test_sdl3_init.flap -o /tmp/test_sdl3_init
/tmp/test_sdl3_init
if [ $? -eq 0 ]; then
    echo "✓ SDL3 init/quit test passed"
else
    echo "✗ SDL3 init/quit test failed with exit code $?"
fi

echo ""
echo "=== All GUI tests complete ==="
