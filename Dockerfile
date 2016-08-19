FROM golang:latest
 
RUN apt-get -qq update
RUN apt-get -yq dist-upgrade
RUN apt-get -yqq install protobuf-compiler wget

RUN echo "America/Denver" > /etc/timezone && dpkg-reconfigure -f noninteractive tzdata

ADD . /go/src/BuffBusTracker/
WORKDIR /go/src/BuffBusTracker

# Protobufs
RUN go get -u github.com/golang/protobuf/proto
RUN go get -u github.com/golang/protobuf/protoc-gen-go
RUN wget https://developers.google.com/transit/gtfs-realtime/gtfs-realtime.proto
RUN mkdir lib/proto && protoc --go_out=./lib/proto/ ./gtfs-realtime.proto
RUN rm gtfs-realtime.proto
 
EXPOSE 8080

CMD go install BuffBusTracker/server/ && /go/bin/server
