# Developer's section

 * [Scenario to be validated](#scenario-to-be-validated)
    * [Test 0 : Build executable and test it](#test-0--build-executable-and-test-it)
    * [Test 1 : source -&gt; compile -&gt; run](#test-1--source---compile---run)
    * [Test 2 : binary -&gt; run](#test-2--binary---run)
    * [Test 3 : debug](#test-3--debug)
    * [Test 4 : source -&gt; compile -&gt; kill pod -&gt; compile again (m2 repo is back again)](#test-4--source---compile---kill-pod---compile-again-m2-repo-is-back-again)
 * [Build the supervisor and java s2i images](#build-the-supervisor-and-java-s2i-images)
    * [Common step](#common-step)
    * [Supervisord image](#supervisord-image)
    * [Java S2I image](#java-s2i-image)

# Scenario to be validated

## Test 0 : Build executable and test it

- Log on to an OpenShift cluster with an `admin` role
- Open or create the following project : `k8s-supervisord`
- Move under the `spring-boot` folder and run these commands

```bash
oc delete --force --grace-period=0 all --all
oc delete pvc/m2-data

go build -o sb *.go
export PATH=$PATH:$(pwd)
export CURRENT=$(pwd)

cd spring-boot
mvn clean package

sb init -n k8s-supervisord
sb push --mode binary
sb exec start
sb exec stop
cd $CURRENT
```

## Test 1 : source -> compile -> run

- Log on to an OpenShift cluster with an `admin` role
- Open or create the following project : `k8s-supervisord`
- Move under the `spring-boot` folder and run these commands

```bash
oc delete --force --grace-period=0 all --all
oc delete pvc/m2-data
export CURRENT=$(pwd)

cd spring-boot
go run ../main.go init -n k8s-supervisord
go run ../main.go push --mode source
go run ../main.go compile
go run ../main.go exec start
cd $CURRENT
```

- Execute this command within another terminal

```bash
URL="http://$(oc get routes/spring-boot-http -o jsonpath='{.spec.host}')"
curl $URL/api/greeting
```

## Test 2 : binary -> run

- Log on to an OpenShift cluster with an `admin` role
- Open or create the following project : `k8s-supervisord`
- Move under the `spring-boot` folder and run these commands

```bash
oc delete --force --grace-period=0 all --all
oc delete pvc/m2-data

export CURRENT=$(pwd)
cd spring-boot
go run ../main.go init -n k8s-supervisord
go run ../main.go push --mode binary
go run ../main.go exec start
cd $CURRENT
```

- Execute this command within another terminal

```bash
URL="http://$(oc get routes/spring-boot-http -o jsonpath='{.spec.host}')"
curl $URL/api/greeting
```

## Test 3 : debug

- Log on to an OpenShift cluster with an `admin` role
- Open or create the following project : `k8s-supervisord`
- Move under the `spring-boot` folder and run these commands

```bash
oc delete --force --grace-period=0 all --all
oc delete pvc/m2-data

export CURRENT=$(pwd)
cd spring-boot
go run ../main.go init -n k8s-supervisord
go run ../main.go push --mode binary
go run ../main.go exec stop
go run ../main.go debug
cd $CURRENT
```

## Test 4 : source -> compile -> kill pod -> compile again (m2 repo is back again)

- Log on to an OpenShift cluster with an `admin` role
- Open or create the following project : `k8s-supervisord`
- Move under the `spring-boot` folder and run these commands

```bash
oc delete --force --grace-period=0 all --all
oc delete pvc/m2-data

export CURRENT=$(pwd)
cd spring-boot
go run ../main.go init -n k8s-supervisord
go run ../main.go push --mode source
go run ../main.go compile
oc delete --grace-period=0 --force=true pod -l app=spring-boot-http 
go run ../main.go push --mode source
go run ../main.go compile
go run ../main.go exec start
cd $CURRENT
```

# Build the supervisor and java s2i images

## Common step
 
Export the Docker ENV var (`DOCKER_HOST, ....`) to access the docker daemon
```bash
eval $(minishift docker-env)
```
  
## Supervisord image  

To build the `copy-supervisord` docker image containing the `go supervisord` application, then follow these instructions

**WARNING**: In order to build a multi-stages docker image, it is required to install [imagebuilder](https://github.com/openshift/imagebuilder) 
as the docker version packaged with minishift is too old and doesn't support such multi-stage option !

```bash
cd supervisord
imagebuilder -t <username>/copy-supervisord:latest .
```
  
Tag the docker image and push it to `quay.io`

```bash
TAG_ID=$(docker images -q cmoulliard/copy-supervisord:latest)
docker tag $TAG_ID quay.io/snowdrop/supervisord
docker login quai.io
docker push quay.io/snowdrop/supervisord
```
  
## Java S2I image

Due to permissions's issue to access the folder `/tmp/src`, the Red Hat OpenJDK1.8 S2I image must be enhanced to add the permission needed for the group `0`

Here is the snippet's part of the Dockerfile

```docker
USER root
RUN mkdir -p /tmp/src/target

RUN chgrp -R 0 /tmp/src/ && \
    chmod -R g+rw /tmp/src/
```

Execute such commands to build the docker image of the `Java S2I` and pusblish it on `Quay.io`
 
```bash
docker build -t <username>/spring-boot-http:latest .
docker tag 00c6b955c3e1 quay.io/snowdrop/spring-boot-s2i
docker push quay.io/snowdrop/spring-boot-s2i
```   