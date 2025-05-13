#!/bin/bash
find . -name "*.go" -type f -exec sed -i 's|github.com/mariasu11/logstream|github.com/mariasu11/logstreamApp|g' {} \;
find . -name "*.md" -type f -exec sed -i 's|github.com/mariasu11/logstream|github.com/mariasu11/logstreamApp|g' {} \;
find . -name "*.yml" -type f -exec sed -i 's|github.com/mariasu11/logstream|github.com/mariasu11/logstreamApp|g' {} \;
find . -name "*.yaml" -type f -exec sed -i 's|github.com/mariasu11/logstream|github.com/mariasu11/logstreamApp|g' {} \;
