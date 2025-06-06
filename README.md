s3proxy
=======

A S3 proxy server between your application and S3 for upload and download of objects. 
Why use s3proxy ? To centralize credentials and access rights in your application infrastructure.

S3Proxy can create presigned urls to allow another application (e.g. mmp) to upload or download objects from S3, and delete or duplicate objects.

Please note that uploads and downloads are using presigned urls, so S3proxy generates the url and returns it to your app, which then uses it to perform the upload or download.
Generating a presigned url is purely local and does not entails communicating with S3 (or Minio). This means that the presigned url may be rejected if the S3proxy credentials do not allow accessing the bucket.

Deletes and copies are proxyfied, and S3Proxy itself sends the HTTP request to S3.

## Architecture

s3proxy has been developed as a REST service. It is generating pre-signed authentified upload and download URL on a object in a bucket.
This presigned URL has a duration period and can be used by any basic HTTP client in any language.

## Requirements 

* Docker version 17.12.0+
* Go 1.17+


## Build

* Install [Go](https://golang.org/doc/install)
* Set $GOPATH variable, by default it should be something like $HOME/go
* Set $GOBIN variable, by default it should be something like $GOPATH/bin
* Make sure $GOBIN is in your $PATH : `export $PATH=$PATH:$GOBIN`

The previous steps are needed for some of the tools this projects uses, mainly goimports.

This project itself uses go modules, and so does NOT need to be in your GOPATH.
It would be better to clone it outside of the $GOPATH (or else you'll need to export `GO111MODULE=on`). 

Clone the project : `git clone git@github.com:mirakl/s3proxy.git` and run `make`.


## Build the docker image

* run `make docker-image`


## Push the docker image

* run `make docker-image-push`


## Add a new release

You only need to add a tag version in this repo without "v". [Dockerhub](https://hub.docker.com/repository/docker/mirakl/s3proxy) will see this tag and create a new artefact (docker image).


## s3proxy Configuration

You can use environment variables or command line options for configuration.


### Command Line Options

```
Usage of s3proxy:
    --http-port : The port that the proxy binds to
    --api-key : Define server side API key for API call authorization
    --use-rsyslog : Add rsyslog as second logging destination by specifying the rsyslog host and port (ex. localhost:514)
    --use-minio : Use minio as backend by specifying the minio server host and port (ex. localhost:9000)
    --minio-access-key : Minion AccessKey equivalent to a AWS_ACCESS_KEY_ID
    --minio-secret-key : Minion AccessKey equivalent to a AWS_SECRET_ACCESS_KEY   
```


### Environment variables

The following environment variables can be used in place of the corresponding command-line arguments:

- `S3PROXY_HTTP_PORT`
- `S3PROXY_API_KEY`
- `S3PROXY_USE_RSYSLOG`
- `S3PROXY_USE_MINIO`
- `S3PROXY_MINIO_ACCESS_KEY`
- `S3PROXY_MINIO_SECRET_KEY`


### Minimum configuration for S3 backend

The minimum configuration for s3proxy is defined by the AWS credentials, to do so define the following env. variables :

* `AWS_REGION` : endpoint used to interact with S3 (ex: eu-west-1)
* `AWS_ACCESS_KEY` : iam user used access key for S3 (from aws console)
* `AWS_SECRET_ACCESS_KEY` : iam user used secret for S3 (from aws console)


### Minimum configuration for Minio backend

The minimum configuration for minio backend, you can set the following env. variables or command line options :

* `AWS_REGION` : you have to define this variable even if it is not used for minio (ex: eu-west-1)
* `S3PROXY_USE_MINIO (or --use-minio)` : minion backend url (ex: localhost:9000)
* `S3PROXY_MINIO_ACCESS_KEY (or --minio-access-key)` : minio access key (check minio server stdout)
* `S3PROXY_MINIO_SECRET_KEY (or --minio-secret-key)` : minio secret key (check minio server stdout)


### Advanced configuration

You can customize the http port, define a remote syslog server for centralized logs or define an s3 compatible backend like minio.
Minio (https://minio.io) is useful when you need to run integration tests or develop on local  without access to S3.

example :

```
./s3proxy \
    --http-port 80 \
    --api-key 3f300bdc-0028-11e8-ba89-0ed5f89f718b \
    --use-rsyslog rsyslog:514 \
    --use-minio minio:9000 \
    --minio-access-key AKIAIOSFODNN7EXAMPLE \
    --minio-secret-key wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY     
```


## Logging Format

By default, s3proxy logs requests to stdout the following format :

```
%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x} %{message}
```


## Endpoints Documentation

s3proxy responds directly to the following endpoints.

### Health check :

* `/` 
    - returns a 200 OK response with the version number (used for health checks)

### Presigned URL API :

* Create URL for upload : `POST /api/v1/presigned/url/:bucket/:key`    
   - returns an 200 OK : create a URL for upload
    
* Create URL for download : `GET /api/v1/presigned/url/:bucket/:key`  
    - return an 200 OK : create a URL for download

### Object API

* Delete object : `DELETE /api/v1/object/:bucket/:key`  
    - return an 200 OK response : delete the object defined by the bucket and the key
    
* Bulk Delete object : `POST /api/v1/object/delete/:bucket` with a body containing list of keys "key=...&key=..."  
    - return an 200 OK response : delete the object defined by the bucket and the key
    - return 400 Bad request if key parameter is missing

* Copy object : `POST /api/v1/object/copy/:bucket/:key?destBucket=...&destKey=...`
    - return an 200 OK : copy the object defined by the bucket and the key to the destBucket and destKey
    - return 400 Bad request if destBucket or destKey are missing
    - return 404 Not Found if the bucket or the key are not found (also destBucket)

### Parameters

* bucket : name of the bucket for example : mybucket
* key : relative path to the object for example : folder1/folder2/file.txt
* destBucket : destination bucket for example : mybucket
* destKey : destination key for example : /folder/file2.txt 

## curl examples

* Create a URL for upload :

```
curl -H "Authorization: ${API_KEY}" -X POST \ 
    http://localhost:8080/api/v1/presigned/url/my-bucket/folder1/file.txt`
```

Response : HTTP CODE 200

`{"url" : "http://..."}`

You can use the url in the response to upload file to the backend

`curl -v -H 'Expect:' --upload-file /tmp/file1.txt "${URL}"`

* Create a URL for download :

```
curl -H "Authorization: ${API_KEY}" -X GET \ 
    http://localhost:8080/api/v1/presigned/url/my-bucket/folder1/file.txt`
```

Response : HTTP CODE 200

`{"url" : "http://..."}`

You can use the url in the response to download file from the backend

`curl -v -o /tmp/file1.txt "${URL}"`

* Delete an object :

```
curl -H "Authorization: ${API_KEY}" -X DELETE \ 
    http://localhost:8080/api/v1/object/my-bucket/folder1/file.txt`
```

Response : HTTP CODE 200

`{"response" : "ok"}`

* Bulk delete an object :

```
curl -H "Authorization: ${API_KEY}" -d "key=/folder1/file1.txt&key=/folder1/file2.txt" \ 
    http://localhost:8080/api/v1/object/my-bucket`
```

Response : HTTP CODE 200

`{"response" : "ok"}`


* Copy an object :

```
curl -H "Authorization: ${API_KEY}" -X POST \ 
    http://localhost:8080/api/v1/object/copy/my-bucket/folder1/file.txt?destBucket=my-bucket&destKey=/folder1/file2.txt`
```

Response : HTTP CODE 200

`{"response" : "ok"}`


* Errors : If an error has occurred then a response code != 200 is sent with a response body

`{"error" : "<message of the error>"}`

If the object does not exists, it returns always 200.

## Development

To setup your development environment you can run the docker-compose.
This will start minio and a syslog server

To run :  

`docker-compose -f ./test/docker-compose.yml up -d minio rsyslog createbuckets`

To stop : 

`docker-compose down`


## Tests

Several kind of tests are available :

* Unit tests

* Integration tests

* End-to-end tests


### Unit tests

Unit tests are used to verify the wanted behaviour of the s3proxy.
They don't need any external components, they use a fake S3 backend for running tests. 

To run the unit tests : `make test`


### Integration tests

Integration tests are used to verify the integration with a real s3 backend and a rsyslog server. 
In our tests we are using minio server which provides a S3 compatible API.

To run the tests : `make integration-test`


### End-to-end tests

End-to-end tests are used to verify the whole integration with a s3proxy running standalone in its container, a S3 backend (minio) and a rsyslog.
The integration tests simulate external call from a program using the s3proxy. 

To run the tests : `make end2end-test VERSION=1.0.0`

where `VERSION` is the container version of the s3proxy (make sure to build the image first)
