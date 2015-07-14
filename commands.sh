docker kill buffbus-server
docker rm buffbus-server
docker build -t server .
docker run -d --name buffbus-server -p 8080:8080 server
