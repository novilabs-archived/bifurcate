.PHONY: build
NAME = bifurcate
VERSION = 0.6.0-dev

build: install-gox
	@rm -rf build/
	@mkdir -p build
	gox \
		-os="darwin" \
		-os="linux" \
		-output="build/{{.Dir}}_$(VERSION)_{{.OS}}_{{.Arch}}/$(NAME)"

package: build
	$(eval FILES := $(shell ls build))
	@mkdir -p build/tgz
	for f in $(FILES); do \
		(cd $(shell pwd)/build/$$f && tar -zcvf ../tgz/$$f.tar.gz bifurcate); \
		echo $$f; \
	done

install-gox:
	go get github.com/mitchellh/gox
