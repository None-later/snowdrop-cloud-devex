# Scenario to be validated

   * [Test 0 : Build executable and test it](#test-0--build-executable-and-test-it)
   * [Test 1 : source -&gt; compile -&gt; run](#test-1--source---compile---run)
   * [Test 2 : binary -&gt; run](#test-2--binary---run)
   * [Test 3 : debug](#test-3--debug)
   * [Test 4 : source -&gt; compile -&gt; kill pod -&gt; compile again (m2 repo is back again)](#test-4--source---compile---kill-pod---compile-again-m2-repo-is-back-again)

## Test 0 : Build executable and test it

- Log on to an OpenShift cluster with an `admin` role
- Open or create the following project : `k8s-supervisord`
- Move under the `spring-boot` folder and run these commands

```bash
go build -o sb *.go
export PATH=$PATH:$(pwd)
export CURRENT=$(pwd)

cd spring-boot
mvn clean package
oc delete --force --grace-period=0 all --all
oc delete pvc/m2-data

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

cd spring-boot
go run ../main.go init -n k8s-supervisord
go run ../main.go push --mode source
go run ../main.go compile
go run ../main.go exec start
cd ..
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

cd spring-boot
go run ../main.go init -n k8s-supervisord
go run ../main.go push --mode binary
go run ../main.go exec start
cd ..
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

cd spring-boot
go run ../main.go init -n k8s-supervisord
go run ../main.go push --mode binary
go run ../main.go exec stop
go run ../main.go debug
cd ..
```

## Test 4 : source -> compile -> kill pod -> compile again (m2 repo is back again)

- Log on to an OpenShift cluster with an `admin` role
- Open or create the following project : `k8s-supervisord`
- Move under the `spring-boot` folder and run these commands

```bash
oc delete --force --grace-period=0 all --all
oc delete pvc/m2-data

cd spring-boot
go run ../main.go init -n k8s-supervisord
go run ../main.go push --mode source
go run ../main.go compile
oc delete --grace-period=0 --force=true pod -l app=spring-boot-http 
go run ../main.go push --mode source
go run ../main.go compile
go run ../main.go exec start
cd ..
```