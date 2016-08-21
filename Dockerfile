FROM golang:latest
 
WORKDIR /go/src/BuffBusTracker

# General changes
RUN apt-get -qq update
RUN apt-get -yq dist-upgrade
RUN apt-get -yqq install protobuf-compiler wget gawk unzip
RUN echo "America/Denver" > /etc/timezone && dpkg-reconfigure -f noninteractive tzdata

# Go libs
RUN go get -u github.com/golang/protobuf/proto
RUN go get -u github.com/golang/protobuf/protoc-gen-go
RUN go get -u github.com/empatica/csvparser

# Setup Protobufs
RUN wget https://developers.google.com/transit/gtfs-realtime/gtfs-realtime.proto
RUN mkdir -p lib/proto && protoc --go_out=./lib/proto/ ./gtfs-realtime.proto
RUN rm gtfs-realtime.proto

# Get RTD schedule stop data
RUN wget http://www.rtd-denver.com/GoogleFeeder/google_transit.zip
RUN unzip -p google_transit.zip stops.txt > RTDstops.txt
# Trim stop data to important fields and guarantee field data types (no strings as ints RTD...)
RUN awk -i inplace -F "," '$11~/^[0-9]+$/ && $1~/^[0-9\.-]+$/ && $4~/^[0-9\.-]+$/ {printf "%s,%s,%s,%s\r\n",$11,$9,$1,$4}' RTDstops.txt
RUN rm google_transit.zip

ADD . /go/src/BuffBusTracker/
 
EXPOSE 8080

CMD go install BuffBusTracker/server/ && /go/bin/server
