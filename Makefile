build-test-dockerfile:
	podman build -f filebrowser-test.Dockerfile --tag filebrowserui-test:latest

spawn-filebrowser:
	podman run --name filebrowser-test --rm --net=host filebrowserui-test:latest

test: build-test-dockerfile
	$(info makefile: Spawning postgres container)
	podman run --name filebrowser-test --rm --net=host filebrowserui-test:latest &

	until curl -s 127.0.0.1:8080 > /dev/null; do echo "makefile: waiting on filebrowser container to finish initalizing" && sleep 0.1; done

	$(info makefile: Starting Go Test)
	-FILEBROWSER_USERNAME=admin FILEBROWSER_PASSWORD=admin FILEBROWSER_HOST=http://127.0.0.1:8080 go test -race ./...

	$(info makefile: Killing postgres)
	podman kill filebrowser-test

test-verbose: build-test-dockerfile
	$(info makefile: Spawning postgres container)
	podman run --name filebrowser-test --rm --net=host filebrowserui-test:latest &

	until curl -s 127.0.0.1:8080 > /dev/null; do echo "makefile: waiting on filebrowser container to finish initalizing" && sleep 0.1; done

	$(info makefile: Starting Go Test)
	-FILEBROWSER_USERNAME=admin FILEBROWSER_PASSWORD=admin FILEBROWSER_HOST=http://127.0.0.1:8080 go test -race -v ./...

	$(info makefile: Killing postgres)
	podman kill filebrowser-test
