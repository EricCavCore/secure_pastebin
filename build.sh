TAG=v1.0.1

docker build -t registry.cnsmanaged.com/cns/spb:$TAG .
docker push registry.cnsmanaged.com/cns/spb:$TAG
docker tag registry.cnsmanaged.com/cns/spb:$TAG registry.cnsmanaged.com/cns/spb:latest
docker push registry.cnsmanaged.com/cns/spb:latest
