#!/bin/bash
find . -name "*.go" -type f -exec sed -i 's|github.com/yourusername/logstream|github.com/mariasu11/logstream|g' {} \;
