# build:
# 	protoc -I. --go_out=plugins=grpc:$(GOPATH)/src/github.com/testProject/user-service \
# 		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
# 		proto/auth/auth.proto
# GOOS=linux GOARCH=amd64 go build
# docker build -t shippy-user-service .
# docker build -t eu.gcr.io/shippy-freight/user:latest .
# docker push eu.gcr.io/shippy-freight/user:latest
#\#--net="host"

build: buildApi buildProxy buildSwagger

buildApi:
	protoc -I/usr/local/include -I. \
  	-I$(GOPATH)/src \
  	-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:$(GOPATH)/src/github.com/bege13mot/user-service \
		proto/auth/auth.proto

buildProxy:
	protoc -I/usr/local/include -I. \
  	-I$(GOPATH)/src \
  	-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true:. \
		proto/auth/auth.proto

buildSwagger:
	protoc -I/usr/local/include -I. \
  	-I$(GOPATH)/src \
  	-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--swagger_out=logtostderr=true:. \
		proto/auth/auth.proto

buildDocker:
	# GOOS=linux GOARCH=amd64 go build
	docker build -t user-service .
	# docker build -t eu.gcr.io/shippy-freight/user:latest .
	# docker push eu.gcr.io/shippy-freight/user:latest

run:
	docker run --net="host" \
		-p 50051 \
		-e DB_HOST=localhost \
		-e DB_PASS=password \
		-e DB_USER=postgres \
		-e MICRO_SERVER_ADDRESS=:50051 \
		-e MICRO_REGISTRY=mdns \
		shippy-user-service

deploy:
	sed "s/{{ UPDATED_AT }}/$(shell date)/g" ./deployments/deployment.tmpl > ./deployments/deployment.yml
	kubectl replace -f ./deployments/deployment.yml
