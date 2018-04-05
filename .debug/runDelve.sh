#!/bin/bash

# dlv debug --listen=0.0.0.0:2345 --headless=true --log=true
dlv debug --listen=localhost:2345 --headless=true --log=true ./cmd -- --log-level=debug --skip-monitor=true server --port=8080