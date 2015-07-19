FROM golang:latest
 
RUN apt-get -qq update
RUN apt-get -yq dist-upgrade

ADD . /go/src/buffbus/
WORKDIR /go/
 
EXPOSE 8080

CMD go install buffbus/server/ && /go/bin/server
