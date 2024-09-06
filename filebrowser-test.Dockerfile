FROM alpine:latest
RUN mkdir -p /opt/filebrowser
WORKDIR /opt/filebrowser
RUN wget "https://github.com/filebrowser/filebrowser/releases/download/v2.30.0/linux-amd64-filebrowser.tar.gz"
RUN tar xf linux-amd64-filebrowser.tar.gz
RUN ./filebrowser --help
ENTRYPOINT ["./filebrowser", "-a", "0.0.0.0", "-p", "8080"]
