#!/bin/sh

go \
	build \
	-v \
	./...

go \
	build \
	-v \
	-o cmd/detect-language-ql/detect-language-ql \
	./cmd/detect-language-ql/
