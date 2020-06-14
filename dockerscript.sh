docker build -t forum .
docker container run --publish 8080:8080 --name fcontainer forum
