# WebFileShare

![codeql badge](https://github.com/marshallhumble/WebFileShare/actions/workflows/codeql.yml/badge.svg?branch=main)
![go build badge](https://github.com/marshallhumble/WebFileShare/actions/workflows/go.yml/badge.svg?branch=main)

### Purpose 

This project came about from a need at work. We wanted to find an easy and secure way to share files with other people outside our organization. 

### Status

This is current a work in progress, at the moment I have it setup so that it can upload files and email users of the new file. Users can sign up, login/logout/update profiles. There is also an admin user that can admin the users. Middleware is configured, and various security controls are in place. 

### Next Steps

1. Add in session timeout set by constant
2. Add in remembering the last *X* passwords
3. Finish/Add more tests on new features

### How to Run

#### Make TLS certs:

These certs will go in the ./tls/ folder off the root, you will get a "site not trusted" error in the browser since they are not signed by an authority.

```shell
go run /usr/local/go/src/crypto/tls/generate_cert.go --rsa-bits=2048 --host=localhost
```

OR OSX With Homebrew ex with go 1.22.5

```shell
go run /opt/homebrew/Cellar/go/1.22.5/libexec/src/crypto/tls/generate_cert.go --rsa-bits=2048 --host=localhost
```

#### Setup Env File
Fill out the .env file (template provided, but must be named .env for Docker and application to see it automatically) with sql database and password, these will be used by Docker and the web app for the MySQL Db. 

#### Setup Docker

If you don't have the MySQL image, then:

```shell
docker pull mysql
```

Once that is done then 

```shell
docker compose up --build
```
Then use the ```databaseSchema.sql``` file to create the local tables needed to run. 

#### Running the Application

```shell
go run ./cmd/web
```
Will start the application
