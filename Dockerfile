FROM centos:latest

COPY s3proxy /bin
RUN chmod +x /bin/s3proxy

EXPOSE 8080

USER nobody
ENTRYPOINT ["s3proxy"]
