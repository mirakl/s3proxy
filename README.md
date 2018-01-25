s3proxy
=======

A S3 proxy server between your application and S3 for upload and download of objects. 
Why use s3proxy ? To centralize credentials and access rights in your application infrastructure.


## Architecture

s3proxy has been developped as a REST service. It is generating presigned authentified upload and download URL on a object in a bucket.
This presigned URL has a duration period and can be used by any basic HTTP client in any language.



## Build

* Install [Go](https://golang.org/doc/install)

* Install Go [dep](https://github.com/golang/dep)

* Check GOPATH env. variable, by default it should be something like $HOME/go

* Go to $GOPATH/src and checkout the project : `git clone git@github.com:mirakl/s3proxy.git`

* run `make build`

To build the docker image : `$ docker build --no-cache -t mirakl/s3proxy .`

(If you want to build a linux binaries from your mac  : `$ make build-linux-amd64`)

To ensure dependencies : `make deps`


## s3proxy Configuration

You can use environnement variables or command line options for configuration.


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

* `AWS_REGION` : endpoint used to interact witj S3 (ex: eu-west-1)
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
Minio (https://minio.io) is usefull when you need to run integration tests or develop on local  without access to S3.

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

### Presigned URL API :

* `/` - returns a 200 OK response with the version number (used for health checks)
* `POST /api/v1/presigned/url/:bucket/:key` - returns an 200 OK response : create a URL for upload
* `GET /api/v1/presigned/url/:bucket/:key`  - return an 200 OK response : create a URL for download

### Object API
* `DELETE /api/v1/object/:bucket/:key`  - return an 200 OK response : delete the object defined by the bucket and the key

### Parameters

* bucket : name of the bucket for example : mybucket
* key : relative path to the object for example : folder1/folder2/file.txt

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

You can use the url in the response to download file from the backend

`curl -v -o "${FILE}" "${URL}"`


* Errors : If an error has occured then a reponse code != 20 is sent with a response body

`{"error" : "<message of the error>"}`


## Development / Integration tests

To setup your development environment or to setup an continous integration job, you can run the docker-compose.
This will start s3proxy with minio and a syslog server

To run docker-compose make sure you first have build a docker image of s3proxy 

```
make build-linux-amd64
docker build --no-cache -t mirakl/s3proxy .
```
then run

`docker-compose up -d --build`

To stop the environment, run : 

`docker-compose down`


## Tester

s3proxy_tester.sh is shell script which will test the creation of the uplaod and download url.
It creates s3proxy with all the backends needed, launch the test and destroy the environment.
It exits with 1 if an error has occured or the webservice send a response code != 200 
