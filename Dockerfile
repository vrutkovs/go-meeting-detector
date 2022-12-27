# build stage
FROM registry.access.redhat.com/ubi8/ubi:8.5 AS build-env
RUN dnf install -y golang
ADD . /src
RUN cd /src && go build -o video-active-mqtt

# final stage
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5
WORKDIR /app
COPY --from=build-env /src/video-active-mqtt /app/
ENTRYPOINT ./video-active-mqtt
