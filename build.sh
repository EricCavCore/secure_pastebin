TAG=v1.0.0

docker build -t registry.cnsmanaged.com/cns/spb:$TAG .
docker push registry.cnsmanaged.com/cns/spb:$TAG