FROM golang:latest
 
RUN apt-get -qq update
RUN apt-get -yq dist-upgrade

ADD . /go/src/BuffBusTracker/
WORKDIR /go/
 
EXPOSE 8080

CMD go install BuffBusTracker/server/ && /go/bin/server
