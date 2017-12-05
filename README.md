[![Build Status][travis-img]][travis]

Prototype implementation of the console daemon described at:

<https://github.com/CCI-MOC/hil/issues/417#issuecomment-303564763>

The details of the API differ slightly from what is describe there. This
document provides an up-to-date description of the server's use. The
intended audience is familiar with current HIL internals; we will likely
change the explanations to avoid this prerequisite in the future.

# Configuration

A config file is needed, whose contents should look like:


```json
{
	"ListenAddr": ":8080",
	"AdminToken": "44d5ebcb1aae23bfefc8dca8314797eb"
}
```

The admin token should be a (cryptographically randomly generated)
128-bit value encoded in hexadecimal. You can generate such a token by
running:

    ./console-service -gen-token

By default, the server looks for the config file at `./config.json`, but
the `-config` command line option can be used to override this.

# Api

The server provides a simple REST api. Most operations are "admin"
operations. This file describes the api in a similar format to that used
by `docs/rest_api.md` in the HIL source tree.

## Admin Operations

Each admin operation requires the client to authenticate using basic
auth, with a username of "admin" and a password equal to the
"AdminToken" in the config file.

### Registering or updating a node

`PUT /node/{node_id}`

Request body:

```json
{
    "type": "ipmi",
    "info": {
        "addr": "10.0.0.4",
        "user": "ipmiuser",
        "pass": "ipmipass"
    }
}
```

Notes:

* The above is for ipmi controllers; right now this is the only
  "real" driver, but there are other possible values of `"type"`
  that are used for testing/development. For those, see the
  relevant source under `./internal/driver`.
* The `node_id` is an arbitrary label.
* The fields in the `info` field are passed directly to ipmitool
* If the node already exists, the information will be updated.

### Checking the version of a node

`GET /node/{node_id}/version`

Response body

```json
{
    "version": 7
}
```

Get the current version of a node.

### Updating the version of a node

`PUT /node/{node_id}/version`

Request body:

```json
{
    "version": 7
}
```

Update the version number of the node, disconnecting any existing
clients and invalidating any tokens.  The version number in the request
body must be `$current_version_number + 1`; otherwise an error will be
returned and the version will not be updated. Returns the updated
version number. (which should be the same as in the body of the
request).

Response body:

```json
{
    "version": 7
}
```

### Getting a new console token

Request body:

`POST /node/{node_id}/console-endpoints`

```json
{
    "version": 3
}
```

Response body (success):

```json
{
    "token": "6119cdf777334998d7068dece09069b8"
}
```

Response body (failure):

```json
{
    "version": 4
}
```

Notes:

* The version in the request must match the current version of the node.
* The token in a successful response is to be used to view the console.
* In the case of a version-mismatch, the response body will return the
  correct version.

## Non-admin operations


### Viewing the console

`GET /node/{node_id}/console?token=<token>`

Notes:

* The `<token>` is fetched as described above
* Data from the console will begin streaming from the response body.

# Extras

* If the `-dummydialer` cli option is passed, rather than launching
  ipmitool, the server will simply open a tcp connection to the
  "addr" specified (in which case it should be of the form required
  by [net.Dial][net.Dial]. This is useful for experimentation.
* There's some preliminary work on supporting a database, but it isn't
  actually used. The `-dbpath` argument sets the path, but the db won't
  be used beyond initializing a schema.

[net.Dial]: https://golang.org/pkg/net/#Dial

[travis]: https://travis-ci.org/zenhack/console-service
[travis-img]: https://travis-ci.org/zenhack/console-service.svg?branch=master
