#!/bin/sh
mockery -dir pkg/model -inpkg -all

mockery -dir pkg/codefresh -inpkg -all