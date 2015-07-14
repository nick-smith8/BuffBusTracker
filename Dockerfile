FROM golang
 
ADD . /go/src/buffbus/
RUN go install buffbus/server/
ENTRYPOINT /go/bin/server
 
EXPOSE 8080
