# RESTful API in Golang

In this sample, we create a simple guestbook application to demonstrates how to create a set of CURD RESTful APIs with fission function.

# Installation


1. Install go dependencies

```bash
$ cd restful-api-go/rest-api
$ glide install -v
```

2. Create database pod

```bash
$ cd restful-api-go/
$ kubectl apply -f cockroachdb.yaml
```

3. Use `fission spec` to create application

```bash
$ cd restful-api-go/
$ fission spec apply --wait
```

4. Check application status

After `fission spec apply`, you can check that functions, http triggers and package are successfully created.

```bash
$ fission fn list
NAME           UID                                  ENV EXECUTORTYPE MINSCALE MAXSCALE MINCPU MAXCPU MINMEMORY MAXMEMORY TARGETCPU
restapi-delete 4dd17984-8a65-11e8-a532-ea45a7fa3496 go  poolmgr      0        1        0      0      0         0         80
restapi-get    4dd45630-8a65-11e8-a532-ea45a7fa3496 go  poolmgr      0        1        0      0      0         0         80
restapi-post   4dd73bd5-8a65-11e8-a532-ea45a7fa3496 go  poolmgr      0        1        0      0      0         0         80
restapi-update 4ddb4339-8a65-11e8-a532-ea45a7fa3496 go  poolmgr      0        1        0      0      0         0         80

$ fission route list
NAME                                 METHOD HOST URL                             INGRESS FUNCTION_NAME
0b533b77-bf00-4035-bd37-035a68baa597 POST        /guestbook/messages             false   restapi-post
c68f0789-a0d9-4860-9469-9b5053df44d3 GET         /guestbook/messages/            false   restapi-get
29c87b8b-ffc1-4a13-bb1c-87f66367474a GET         /guestbook/messages/{id:[0-9]+} false   restapi-get
4f917abc-258b-487b-ba09-ed384b4bd7e7 PUT         /guestbook/messages/{id:[0-9]+} false   restapi-update
b94486ca-a7cc-49cd-8fde-822293e21bf3 DELETE      /guestbook/messages/{id:[0-9]+} false   restapi-delete

$ fission pkg list
NAME           BUILD_STATUS ENV
restapi-go-pkg succeeded    go
```

# Usage

1. Create a post

```bash
$ curl -v -X POST \
    http://$FISSION_ROUTER/guestbook/messages \
    -H 'Content-Type: application/json' \
    -d '{"message": "hello world!"}'
```

2. Get all posts/single post

```bash
$ curl -v -X GET http://$FISSION_ROUTER/guestbook/messages/
```

You should see a list posts are returned.

```json
[
    {
        "id": 366456868654284801,
        "message": "hello world!"
    }
]
```

Now, let's try to get a single post with post `id`.

```bash
$ curl -v -X GET http://$FISSION_ROUTER/guestbook/messages/366456868654284801
```

3. Update post

```bash
$ curl -v -X PUT \
    http://192.168.64.4:31314/guestbook/messages/366456868654284801 \
    -H 'Content-Type: application/json' \
    -d '{"message": "hello world again!"}'
```

4. Delete post

```bash
$ curl -X DELETE \
    http://192.168.64.4:31314/guestbook/messages/366456868654284801 \
    -H 'Cache-Control: no-cache'
```
