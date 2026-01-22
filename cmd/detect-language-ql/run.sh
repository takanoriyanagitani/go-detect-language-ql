#!/bin/sh

./detect-language-ql \
	-port 12281 \
	-bind-addr 127.0.0.1 \
	-all-spoken \
	-preload \
	-read-timeout 10s \
	-write-timeout 10s \
	-log-format text
