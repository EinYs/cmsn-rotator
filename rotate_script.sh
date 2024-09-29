#!/bin/bash

# Check for the input argument and run the corresponding rotate command
case $1 in
  1)
    echo "Running rotate 1..."
    /app/rotate 1
    ;;
  2)
    echo "Running rotate 2..."
    /app/rotate 2
    ;;
  3)
    echo "Running rotate 3..."
    /app/rotate 3
    ;;
  *)
    echo "Invalid argument. Please use 1, 2, or 3."
    ;;
esac
