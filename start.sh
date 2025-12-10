cd ~/fabric/mynetwork
export PATH=$PATH:$PWD/bin
export FABRIC_CFG_PATH=${PWD}/config

# 关闭网络（保持持久化数据）
export DOCKER_SOCK=/var/run/docker.sock
docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml -f compose/compose-couch.yaml -f compose/docker/docker-compose-couch.yaml down

# 开启网络
export DOCKER_SOCK=/var/run/docker.sock
docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml -f compose/compose-couch.yaml -f compose/docker/docker-compose-couch.yaml up -d