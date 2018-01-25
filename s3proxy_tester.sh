#!/bin/bash

set -e

trap "docker-compose down" SIGINT SIGQUIT SIGTERM TERM EXIT

docker-compose up -d --build

mkdir /tmp/s3proxy_tester || true

docker run --rm --net=s3proxy_default -v /tmp/s3proxy_tester:/tmp -w /tmp -i centos:7 /bin/bash -s <<EOF
#!/bin/bash

set -xe

yum install -y epel-release &> /dev/null
yum install -y jq &> /dev/null

FILE="file-\${RANDOM}"

dd if=/dev/zero of=\${FILE}  bs=1024  count=10240

API_KEY="3f300bdc-0028-11e8-ba89-0ed5f89f718b"
BUCKET="s3proxy-bucket"

FOLDER="folder-\${RANDOM}"
KEY="\${FOLDER}/\${FILE}"


# Get a presigned URL for an upload
JSON=\$(curl -H "Authorization: \${API_KEY}" -X POST http://s3proxy:8080/api/v1/presigned/url/\${BUCKET}/\${KEY})
UPLOAD_URL=\$(echo "\${JSON}" | jq -r '.url')
HTTP_CODE=\$(curl -sv -o /dev/null -w "%{http_code}" -H 'Expect:' --upload-file "\${FILE}" "\${UPLOAD_URL}")

[[ "\$HTTP_CODE" != "200" ]] && echo "Last call failed : \${HTTP_CODE}" && exit 1


# Get a presigned URL for a download
JSON=\$(curl -H "Authorization: \${API_KEY}" -X GET http://s3proxy:8080/api/v1/presigned/url/\${BUCKET}/\${KEY})
DOWNLOAD_URL=\$(echo "\${JSON}" | jq -r '.url')
HTTP_CODE=\$(curl -sv -o "\${FILE}-downloaded" -w "%{http_code}" "\${DOWNLOAD_URL}")

[[ "\$HTTP_CODE" != "200" ]] && echo "Last call failed : \${HTTP_CODE}" && exit 1


# Delete a file in a bucket
HTTP_CODE=\$(curl -sv -o /dev/null -w "%{http_code}" -H "Authorization: \${API_KEY}" -X DELETE http://s3proxy:8080/api/v1/object/\${BUCKET}/\${KEY})

[[ "\$HTTP_CODE" != "200" ]] && echo "Last call failed : \${HTTP_CODE}" && exit 1

echo " !! Test passed !! "
EOF

rm -fR /tmp/s3proxy_tester
