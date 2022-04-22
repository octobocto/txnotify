file=proto/txnotify.proto

DIRECTORY=$(dirname "${file}")
echo "Generating protos from ${file}, into ${DIRECTORY}"

# Generate the protos.
protoc -I/usr/local/include -I. \
  --go-grpc_out=paths=source_relative:. \
  "${file}"

# Generate the protos.
protoc -I/usr/local/include -I. \
  --go_out=paths=source_relative:. \
  "${file}"

# Generate the REST reverse proxy.
protoc -I/usr/local/include -I. \
  --grpc-gateway_out=logtostderr=true,paths=source_relative,grpc_api_configuration=proto/rest-annotations.yaml:. \
  "${file}"

# Finally, generate the swagger file which describes the REST API in detail.
protoc -I/usr/local/include -I. \
  --swagger_out=logtostderr=true,grpc_api_configuration=proto/rest-annotations.yaml:. \
  "${file}"

sed -i -e 's/rpc//g' "proto/txnotify.swagger.json"
echo Removed all occurences of the substring "rpc"
